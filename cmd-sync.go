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
	"math"
	"runtime"
	"sync"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/countlock"
	"github.com/minio/minio/pkg/iodine"
)

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
	var lock countlock.Locker

	// set up progress bar
	if !globalQuietFlag {
		bar = newCpBar()

		// Keep progress-bar and copy routines in sync.
		lock = countlock.New()
		defer lock.Close()
	}

	go func(sourceURL string, targetURLs []string) {
		for syncURLs := range prepareSyncURLs(sourceURL, targetURLs) {
			if syncURLs.Error != nil {
				// no need to print errors here, any error here
				// will be printed later during Sync()
				continue
			}
			if !globalQuietFlag {
				bar.Extend(syncURLs.SourceContent.Size)
				lock.Up() // Let copy routine know that it is catch up.
			}
		}
	}(sourceURL, targetURLs)

	syncQueue := make(chan bool, int(math.Max(float64(runtime.NumCPU())-1, 1)))
	var wg sync.WaitGroup

	for syncURLs := range prepareSyncURLs(sourceURL, targetURLs) {
		if syncURLs.Error != nil {
			console.Errors(ErrorMessage{
				Message: "Failed with",
				Error:   iodine.New(syncURLs.Error, nil),
			})
			continue
		}

		runtime.Gosched() // Yield more CPU time to progress-bar builder.

		syncQueue <- true
		wg.Add(1)
		if !globalQuietFlag {
			lock.Down() // Do not jump ahead of the progress bar builder above.
		}
		go func(syncURLs *cpURLs, bar *barSend) {
			defer wg.Done()
			srcConfig, err := getHostConfig(syncURLs.SourceContent.Name)
			if err != nil {
				console.Fatals(ErrorMessage{
					Message: "Failed with",
					Error:   iodine.New(err, nil),
				})
				return
			}
			tgtConfig, err := getHostConfig(syncURLs.TargetContent.Name)
			if err != nil {
				console.Fatals(ErrorMessage{
					Message: "Failed with",
					Error:   iodine.New(err, nil),
				})
				return
			}
			if err := doCopy(syncURLs.SourceContent.Name, srcConfig, syncURLs.TargetContent.Name, tgtConfig, bar); err != nil {
				console.Errors(ErrorMessage{
					Message: "Failed with",
					Error:   iodine.New(err, nil),
				})
			}
			<-syncQueue
		}(syncURLs, &bar)
	}
	wg.Wait()
	if !globalQuietFlag {
		bar.Finish()
	}
}
