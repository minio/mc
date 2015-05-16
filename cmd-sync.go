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
	config, err := getMcConfig()
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalf("Unable to read config file [%s]. Reason: [%s].\n", mustGetMcConfigPath(), iodine.ToError(err))
	}

	// Convert arguments to URLs: expand alias, fix format...
	URLs, err := getExpandedURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("Unknown type of URL(s).\n")
		default:
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("Unable to parse arguments. Reason: [%s].\n", e)
		}
	}
	// Separate source and target. 'sync' can take only one source.
	// but any number of targets, even the recursive URLs mixed in-between.
	sourceURL := URLs[0] // first one is source
	targetURLs := URLs[1:]

	// set up progress bar
	bar := newCopyBar(globalQuietFlag)

	go func(sourceURL string, targetURLs []string) {
		for syncURLs := range prepareSyncURLs(sourceURL, targetURLs) {
			if syncURLs.Error != nil {
				// no need to print errors here, any error here
				// will be printed later during Sync()
				continue
			}
			bar.Extend(syncURLs.SourceContent.Size)
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
		go func(syncURLs copyURLs) {
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
			if err := doCopy(syncURLs.SourceContent.Name, srcConfig, syncURLs.TargetContent.Name, tgtConfig, &bar); err != nil {
				console.Errorln(iodine.ToError(err))
			}
			<-syncQueue
		}(*syncURLs)
	}
	wg.Wait()
	bar.Finish()
}
