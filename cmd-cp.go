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
	"fmt"
	"io"
	"math"
	"runtime"
	"sync"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/countlock"
	"github.com/minio/mc/pkg/yielder"
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
      $ mc {{.Name}} https://play.minio.io:9000/photos/burningman2011... https://s3.amazonaws.com/private-photos/burningman/

   3. Copy multiple local folders recursively to Minio object storage.
      $ mc {{.Name}} backup/2014/... backup/2015/... https://play.minio.io:9000/archive/

   4. Copy a bucket recursively from aliased Amazon S3 object storage to local filesystem on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014

   5. Copy an object of non english characters to Amazon S3 object storage.
      $ mc {{.Name}} 本語 s3:andoria/本語

`,
}

// doCopy - Copy a singe file from source to destination
func doCopy(cURLs cpURLs, bar *barSend) error {
	if !globalQuietFlag {
		sourceContentParse, _ := client.Parse(cURLs.SourceContent.Name)
		bar.SetCaption(caption{message: cURLs.SourceContent.Name + ": ", separator: sourceContentParse.Separator})
	}
	reader, length, err := getSource(cURLs.SourceContent.Name)
	if err != nil {
		if !globalQuietFlag {
			bar.ErrorGet(int64(length))
		}
		return iodine.New(err, map[string]string{"URL": cURLs.SourceContent.Name})
	}
	defer reader.Close()

	var newReader io.Reader
	switch globalQuietFlag {
	case true:
		console.Infoln(fmt.Sprintf("‘%s’ -> ‘%s’", cURLs.SourceContent.Name, cURLs.TargetContent.Name))
		newReader = yielder.NewReader(reader)
	default:
		// set up progress
		newReader = bar.NewProxyReader(yielder.NewReader(reader))
	}
	err = putTarget(cURLs.TargetContent.Name, length, newReader)
	if err != nil {
		if !globalQuietFlag {
			bar.ErrorPut(int64(length))
		}
		return iodine.New(err, map[string]string{"URL": cURLs.TargetContent.Name})
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

func doCopyInRoutine(cURLs cpURLs, bar *barSend, cpQueue chan bool, errCh chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	if err := doCopy(cURLs, bar); err != nil {
		errCh <- err
	}
	<-cpQueue // Signal that this copy routine is done.
}

func doPrepareCopyURLs(sourceURLs []string, targetURL string, bar barSend, lock countlock.Locker) {
	for cURLs := range prepareCopyURLs(sourceURLs, targetURL) {
		if cURLs.Error != nil {
			// no need to print errors here, any error here
			// will be printed later during Copy()
			continue
		}
		if !globalQuietFlag {
			bar.Extend(cURLs.SourceContent.Size)
			lock.Up() // Let copy routine know that it has to catch up.
		}
	}
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

		// Wait for all copy routines to complete.
		wg := new(sync.WaitGroup)

		// Pool limited copy routines in parallel.
		cpQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
		defer close(cpQueue)

		go doPrepareCopyURLs(sourceURLs, targetURL, bar, lock)

		for cURLs := range prepareCopyURLs(sourceURLs, targetURL) {
			if cURLs.Error != nil {
				errCh <- cURLs.Error
				continue
			}
			cpQueue <- true // Wait for existing pool to drain.
			wg.Add(1)       // keep track of all the goroutines
			if !globalQuietFlag {
				lock.Down() // Do not jump ahead of the progress bar builder above.
			}
			go doCopyInRoutine(cURLs, &bar, cpQueue, errCh, wg)
		}
		wg.Wait() // wait for the go routines to complete
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
			Error:   iodine.New(errNotConfigured{}, nil),
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
