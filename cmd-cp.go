/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"sync"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/countlock"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
var cpCmd = cli.Command{
	Name:   "cp",
	Usage:  "Copy files and folders from many sources to a single destination",
	Action: runCopyCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE [SOURCE...] TARGET {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Copy list of objects from local file system to Amazon S3 object storage.
      $ mc {{.Name}} Music/*.ogg https://s3.amazonaws.com/jukebox/

   2. Copy a bucket recursively from Minio object storage to Amazon S3 object storage.
      $ mc {{.Name}} http://play.minio.io:9000/photos/burningman2011... https://s3.amazonaws.com/private-photos/burningman/

   3. Copy multiple local folders recursively to Minio object storage.
      $ mc {{.Name}} backup/2014/... backup/2015/... http://play.minio.io:9000/archive/

   4. Copy a bucket recursively from aliased Amazon S3 object storage to local filesystem on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014

   5. Copy an object of non english characters to Amazon S3 object storage.
      $ mc {{.Name}} 本語 s3:andoria/本語

`,
}

// doCopy - Copy a singe file from source to destination
func doCopy(sourceURL string, sourceConfig *hostConfig, targetURL string, targetConfig *hostConfig, bar *barSend) error {
	if !globalQuietFlag {
		bar.SetPrefix(sourceURL + ": ")
	}
	reader, length, err := getSource(sourceURL, sourceConfig)
	if err != nil {
		if !globalQuietFlag {
			bar.ErrorGet(int64(length))
		}
		return iodine.New(err, map[string]string{"URL": sourceURL})
	}
	defer reader.Close()

	var newReader io.Reader
	switch globalQuietFlag {
	case true:
		console.Infoln(fmt.Sprintf("‘%s’ -> ‘%s’", sourceURL, targetURL))
		newReader = reader
	default:
		// set up progress
		newReader = bar.NewProxyReader(reader)
	}

	err = putTarget(targetURL, targetConfig, length, newReader)
	if err != nil {
		if !globalQuietFlag {
			bar.ErrorPut(int64(length))
		}
		return iodine.New(err, nil)
	}
	return nil
}

// args2URLs extracts source and target URLs from command-line args.
func args2URLs(args cli.Args) ([]string, error) {
	config, err := getMcConfig()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	// Convert arguments to URLs: expand alias, fix format...
	URLs, err := getExpandedURLs(args, config.Aliases)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return URLs, nil
}

func doCopyInRoutine(cpurls *cpURLs, bar *barSend, cpQueue chan bool, errCh chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	srcConfig, err := getHostConfig(cpurls.SourceContent.Name)
	if err != nil {
		errCh <- err
		return
	}
	tgtConfig, err := getHostConfig(cpurls.TargetContent.Name)
	if err != nil {
		errCh <- err
		return
	}
	if err := doCopy(cpurls.SourceContent.Name, srcConfig, cpurls.TargetContent.Name, tgtConfig, bar); err != nil {
		errCh <- err
	}
	<-cpQueue // Signal that this copy routine is done.
}

func doCopyCmd(sourceURLs []string, targetURL string, bar barSend) <-chan error {
	errCh := make(chan error)

	go func(sourceURLs []string, targetURL string, bar barSend, errCh chan error) {
		defer close(errCh)

		var lock countlock.Locker
		if !globalQuietFlag {
			// Keep progress-bar and copy routines in sync.
			lock = countlock.New()
			defer lock.Close()
		}

		go func(sourceURLs []string, targetURL string) {
			for cpURLs := range prepareCopyURLs(sourceURLs, targetURL) {
				if cpURLs.Error != nil {
					// no need to print errors here, any error here
					// will be printed later during Copy()
					continue
				}
				if !globalQuietFlag {
					bar.Extend(cpURLs.SourceContent.Size)
					lock.Up() // Let copy routine know that it is catch up.
				}
			}
		}(sourceURLs, targetURL)

		// Pool limited copy routines in parallel.
		cpQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
		defer close(cpQueue)

		// Wait for all copy routines to complete.
		wg := new(sync.WaitGroup)
		for cpURLs := range prepareCopyURLs(sourceURLs, targetURL) {
			if cpURLs.Error != nil {
				errCh <- cpURLs.Error
				continue
			}

			runtime.Gosched() // Yield more CPU time to progress-bar builder.

			cpQueue <- true // Wait for existing pool to drain.
			wg.Add(1)
			if !globalQuietFlag {
				lock.Down() // Do not jump ahead of the progress bar builder above.
			}
			go doCopyInRoutine(cpURLs, &bar, cpQueue, errCh, wg)
		}
		wg.Wait()
	}(sourceURLs, targetURL, bar, errCh)
	return errCh
}

// runCopyCmd is bound to sub-command
func runCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
	}

	if !isMcConfigExist() {
		console.Fatals(ErrorMessage{
			Message: "Please run \"mc config generate\"",
			Error:   iodine.New(errors.New("\"mc\" is not configured"), nil),
		})
	}

	// extract URLs.
	URLs, err := args2URLs(ctx.Args())
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: fmt.Sprintf("Unknown URL types: ‘%s’", URLs),
			Error:   iodine.New(err, nil),
		})
	}

	// Separate source and target. 'cp' can take only one target,
	// but any number of sources, even the recursive URLs mixed in-between.
	sourceURLs := URLs[:len(URLs)-1]
	targetURL := URLs[len(URLs)-1] // Last one is target

	var bar barSend
	// set up progress bar
	if !globalQuietFlag {
		bar = newCpBar()
	}

	for err := range doCopyCmd(sourceURLs, targetURL, bar) {
		if err != nil {
			console.Errors(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(err, nil),
			})
		}
	}
	if !globalQuietFlag {
		bar.Finish()
	}
}
