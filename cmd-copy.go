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
	"io"
	"runtime"
	"sync"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// doCopy - Copy a singe file from source to destination
func doCopy(sourceURL string, sourceConfig *hostConfig, targetURL string, targetConfig *hostConfig, bar *barSend) error {
	if sourceURL == targetURL {
		return iodine.New(errSameURLs{source: sourceURL, target: targetURL}, nil)
	}
	readCloser, length, md5hex, err := getSourceReader(sourceURL, sourceConfig)
	if err != nil {
		return iodine.New(err, nil)
	}
	defer readCloser.Close()

	writeCloser, err := getTargetWriter(targetURL, targetConfig, md5hex, length)
	if err != nil {
		return iodine.New(err, nil)
	}

	var writers []io.Writer
	writers = append(writers, writeCloser)

	// set up progress bar
	writers = append(writers, bar)

	// write progress bar
	multiWriter := io.MultiWriter(writers...)
	// copy data to writers
	_, copyErr := io.CopyN(multiWriter, readCloser, int64(length))
	// close to see the error, verify it later
	err = writeCloser.Close()
	if copyErr != nil {
		return iodine.New(copyErr, nil)
	}
	if err != nil {
		return iodine.New(err, nil)
	}

	return nil
}

// runCopyCmd is bound to sub-command
func runCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
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

	// Separate source and target. 'cp' can take only one target,
	// but any number of sources, even the recursive URLs mixed in-between.
	targetURL := URLs[len(URLs)-1] // Last one is target
	sourceURLs := URLs[:len(URLs)-1]

	// set up progress bar
	bar := newCopyBar(globalQuietFlag)

	go func(sourceURLs []string, targetURL string) {
		for cpURLs := range prepareCopyURLs(sourceURLs, targetURL) {
			if cpURLs.Error != nil {
				// no need to print errors here, any error here
				// will be printed later during Copy()
				continue
			}
			bar.Extend(cpURLs.SourceContent.Size)
		}
	}(sourceURLs, targetURL)

	var cpQueue = make(chan bool, runtime.NumCPU()-1)
	var wg sync.WaitGroup

	for cpURLs := range prepareCopyURLs(sourceURLs, targetURL) {
		if cpURLs.Error != nil {
			console.Errorln(iodine.ToError(cpURLs.Error))
			continue
		}
		cpQueue <- true
		wg.Add(1)
		go func(cpURLs copyURLs) {
			defer wg.Done()
			srcConfig, err := getHostConfig(cpURLs.SourceContent.Name)
			if err != nil {
				console.Errorln(iodine.ToError(err))
				return
			}
			tgtConfig, err := getHostConfig(cpURLs.TargetContent.Name)
			if err != nil {
				console.Errorln(iodine.ToError(err))
				return
			}
			if err := doCopy(cpURLs.SourceContent.Name, srcConfig, cpURLs.TargetContent.Name, tgtConfig, &bar); err != nil {
				console.Errorln(iodine.ToError(err))
			}
			<-cpQueue
		}(*cpURLs)
	}
	wg.Wait()
	bar.Finish()
}
