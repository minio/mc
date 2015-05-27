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
	"runtime"
	"sync"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// doCopy - Copy a singe file from source to destination
func doCopy(sourceURL string, sourceConfig *hostConfig, targetURL string, targetConfig *hostConfig, bar *barSend) error {
	reader, length, err := getSource(sourceURL, sourceConfig)
	if err != nil {
		return iodine.New(err, nil)
	}
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
		reader.Close()
		return iodine.New(err, nil)
	}
	reader.Close()
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

func doCopyInRoutine(cpurls *cpURLs, bar *barSend, cpQueue chan bool, ch chan error, wg *sync.WaitGroup) {
	defer wg.Done()
	srcConfig, err := getHostConfig(cpurls.SourceContent.Name)
	if err != nil {
		ch <- err
		return
	}
	tgtConfig, err := getHostConfig(cpurls.TargetContent.Name)
	if err != nil {
		ch <- err
		return
	}
	if err := doCopy(cpurls.SourceContent.Name, srcConfig, cpurls.TargetContent.Name, tgtConfig, bar); err != nil {
		ch <- err
	}
	<-cpQueue
}

func doCopyCmd(sourceURLs []string, targetURL string, bar barSend) <-chan error {
	ch := make(chan error)
	go func(sourceURLs []string, targetURL string, bar barSend, ch chan error) {
		defer close(ch)
		go func(sourceURLs []string, targetURL string) {
			for cpURLs := range prepareCopyURLs(sourceURLs, targetURL) {
				if cpURLs.Error != nil {
					// no need to print errors here, any error here
					// will be printed later during Copy()
					continue
				}
				if !globalQuietFlag {
					bar.Extend(cpURLs.SourceContent.Size)
				}
			}
		}(sourceURLs, targetURL)

		cpQueue := make(chan bool, intMax(runtime.NumCPU()-1, 1))
		wg := new(sync.WaitGroup)
		for cpURLs := range prepareCopyURLs(sourceURLs, targetURL) {
			if cpURLs.Error != nil {
				ch <- cpURLs.Error
				continue
			}
			cpQueue <- true
			wg.Add(1)
			go doCopyInRoutine(cpURLs, &bar, cpQueue, ch, wg)
		}
		close(cpQueue)
		wg.Wait()
	}(sourceURLs, targetURL, bar, ch)
	return ch
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
			if !globalQuietFlag {
				bar.Error()
			}
		}
	}

	if !globalQuietFlag {
		bar.Finish()
	}
}
