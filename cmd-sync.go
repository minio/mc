/*
 * Minio Client, (C) 2015 Minio, Inc.
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
	"github.com/minio/mc/pkg/yielder"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
var syncCmd = cli.Command{
	Name:   "sync",
	Usage:  "Copy files and folders from a single source to many destinations",
	Action: runSyncCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}}{{if .Flags}} [ARGS...]{{end}} SOURCE TARGET [TARGET...] {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Sync an object from local filesystem to Amazon S3 object storage.
      $ mc {{.Name}} star-trek-episode-10-season4.ogg https://s3.amazonaws.com/trekarchive

   2. Sync a bucket recursively from Minio object storage to multiple buckets on Amazon S3 object storage.
      $ mc {{.Name}} https://play.minio.io:9000/photos/2014... https://s3.amazonaws.com/backup-photos https://s3-west-1.amazonaws.com/local-photos

   3. Sync a local folder recursively to Minio object storage and Amazon S3 object storage.
      $ mc {{.Name}} backup/... https://play.minio.io:9000/archive https://s3.amazonaws.com/archive

   4. Sync a bucket from aliased Amazon S3 object storage to multiple folders on Windows.
      $ mc {{.Name}} s3:documents/2014/... C:\backup\2014 C:\shared\volume\backup\2014

   5. Sync a local directory of non english character recursively to Amazon s3 object storage and Minio object storage.
      $ mc {{.Name}} 本語/... s3:mylocaldocuments play:backup

`,
}

// doSync - Sync an object to multiple destination
func doSync(sURLs syncURLs, bar *barSend, syncQueue chan bool, errCh chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	if !globalQuietFlag {
		bar.SetPrefix(sURLs.SourceContent.Name + ": ")
	}
	reader, length, err := getSource(sURLs.SourceContent.Name)
	if err != nil {
		if !globalQuietFlag {
			bar.ErrorGet(int64(length))
		}
		errCh <- iodine.New(err, map[string]string{"URL": sURLs.SourceContent.Name})
	}
	defer reader.Close()

	var targetURLs []string
	for _, targetContent := range sURLs.TargetContents {
		targetURLs = append(targetURLs, targetContent.Name)
	}

	var newReader io.Reader
	switch globalQuietFlag {
	case true:
		newReader = yielder.NewReader(reader)
	default:
		// set up progress
		newReader = bar.NewProxyReader(yielder.NewReader(reader))
	}
	for err := range putTargets(targetURLs, length, newReader) {
		if err != nil {
			if !globalQuietFlag {
				bar.ErrorPut(int64(length))
			}
			errCh <- iodine.New(err, nil)
		}
	}
	<-syncQueue // Signal that this copy routine is done.
}

func doSyncCmd(sourceURL string, targetURLs []string, bar barSend) <-chan error {
	errCh := make(chan error)
	go func(sourceURL string, targetURLs []string, bar barSend, errCh chan error) {
		defer close(errCh)
		var lock countlock.Locker
		if !globalQuietFlag {
			// Keep progress-bar and copy routines in sync.
			lock = countlock.New()
			defer lock.Close()
		}
		go func(sourceURL string, targetURLs []string) {
			for sURLs := range prepareSyncURLs(sourceURL, targetURLs) {
				if sURLs.Error != nil {
					// no need to print errors here, any error here
					// will be printed later during Sync()
					continue
				}
				if !globalQuietFlag {
					bar.Extend(sURLs.SourceContent.Size)
					lock.Up() // Let copy routine know that it has to catch up.
				}
			}
		}(sourceURL, targetURLs)

		syncQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
		wg := new(sync.WaitGroup)

		for sURLs := range prepareSyncURLs(sourceURL, targetURLs) {
			if sURLs.Error != nil {
				errCh <- iodine.New(sURLs.Error, nil)
				continue
			}
			syncQueue <- true
			wg.Add(1)
			if !globalQuietFlag {
				lock.Down() // Do not jump ahead of the progress bar builder above.
			}
			go doSync(sURLs, &bar, syncQueue, errCh, wg)
		}
		wg.Wait()
	}(sourceURL, targetURLs, bar, errCh)
	return errCh
}

func runSyncCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "sync", 1) // last argument is exit code
	}

	if !isMcConfigExist() {
		console.Fatals(ErrorMessage{
			Message: "Please run \"mc config generate\"",
			Error:   iodine.New(errors.New("\"mc\" is not configured"), nil),
		})
	}

	URLs, err := args2URLs(ctx.Args())
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: fmt.Sprintf("Unknown URL types found: ‘%s’", URLs),
			Error:   iodine.New(err, nil),
		})
	}

	// Separate source and target. 'sync' can take only one source.
	// but any number of targets, even the recursive URLs mixed in-between.
	sourceURL := URLs[0] // first one is source
	targetURLs := URLs[1:]

	var bar barSend
	// set up progress bar
	if !globalQuietFlag {
		bar = newCpBar()
	}

	for err := range doSyncCmd(sourceURL, targetURLs, bar) {
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
