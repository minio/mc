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
	"runtime"
	"sync"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

func runSyncCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "sync", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	URLs, err := args2URLs(ctx.Args())
	if err != nil {
		console.Fatalln(iodine.ToError(err))
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

	go func(sourceURL string, targetURLs []string) {
		for syncURLs := range prepareSyncURLs(sourceURL, targetURLs) {
			if syncURLs.Error != nil {
				// no need to print errors here, any error here
				// will be printed later during Sync()
				continue
			}
			if !globalQuietFlag {
				bar.Extend(syncURLs.SourceContent.Size)
			}
		}
	}(sourceURL, targetURLs)

	var syncQueue = make(chan bool, runtime.NumCPU()-1)
	var wg sync.WaitGroup

	for syncURLs := range prepareSyncURLs(sourceURL, targetURLs) {
		if syncURLs.Error != nil {
			console.Errorln(iodine.ToError(syncURLs.Error))
			continue
		}
		syncQueue <- true
		wg.Add(1)
		go func(syncURLs *cpURLs, bar *barSend) {
			defer wg.Done()
			srcConfig, err := getHostConfig(syncURLs.SourceContent.Name)
			if err != nil {
				console.Errorln(iodine.ToError(err))
				return
			}
			tgtConfig, err := getHostConfig(syncURLs.TargetContent.Name)
			if err != nil {
				console.Errorln(iodine.ToError(err))
				return
			}
			if err := doCopy(syncURLs.SourceContent.Name, srcConfig, syncURLs.TargetContent.Name, tgtConfig, bar); err != nil {
				console.Errorln(iodine.ToError(err))
			}
			<-syncQueue
		}(syncURLs, &bar)
	}
	wg.Wait()
	if !globalQuietFlag {
		bar.Finish()
	}
}
