/*
 * Minimalist Object Storage, (C) 2014,2015 Minio, Inc.
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
	"io/ioutil"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
)

// doCopyCmd copies objects into and from a bucket or between buckets
func multiCopy(targetURLs []string, sourceURL string) (err error) {
	numTargets := len(targetURLs)

	// Parse URL to bucket and object names
	sourceBucket, sourceObject, err := url2Object(sourceURL)
	if err != nil {
		return err
	}

	// Initialize a new client object for the source
	sourceClnt, err := getNewClient(globalDebugFlag, sourceURL)
	if err != nil {
		return err
	}

	// Get a reader for the source object
	sourceReader, sourceSize, err := sourceClnt.Get(sourceBucket, sourceObject)
	if err != nil {
		return err
	}

	targetReaders := make([]io.Reader, numTargets)
	targetWriters := make([]io.Writer, numTargets)
	doneCh := make(chan int)

	for i := 0; i < numTargets; i++ {
		targetReaders[i], targetWriters[i] = io.Pipe()
	}

	go func() {
		var bar *pb.ProgressBar
		if !globalQuietFlag {
			bar = startBar(sourceSize)
			bar.Start()
			sourceReader = ioutil.NopCloser(io.TeeReader(sourceReader, bar))
		}

		io.CopyN(io.MultiWriter(targetWriters...), sourceReader, sourceSize)
		if !globalQuietFlag {
			bar.Finish()
		}
	}()

	var routineCount int
	for i := 0; i < numTargets; i++ {
		targetBucket, targetObject, err := url2Object(targetURLs[i])

		if err != nil {
			return err
		}

		targetClnt, err := getNewClient(globalDebugFlag, targetURLs[i])
		if err != nil {
			return err
		}

		routineCount = i // Keep track of which routine
		go func(index int) {
			// return err via external variable
			err = targetClnt.Put(targetBucket, targetObject, sourceSize, targetReaders[index])

			doneCh <- index // Let the caller know that we are done
		}(i)

		if err != nil { // err is set inside go routine
			break
		}
	}

	for i := routineCount; i >= 0; i-- {
		chIndex := <-doneCh
		if err != nil {
			info(fmt.Sprintf("%s: Failed Error: %v", targetURLs[chIndex], err))
		} else {
			info(fmt.Sprintf("%s: Done", targetURLs[chIndex]))
		}
	}

	return nil
}

// doCopyCmd copies objects into and from a bucket or between buckets
func doCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
	}
	// Convert arguments to URLs: expand alias, fix format...
	urlList, err := parseURLs(ctx)
	if err != nil {
		fatal(err.Error())
		return
	}
	sourceURL := urlList[0]   // First arg is source
	targetURLs := urlList[1:] // 1 or more targets

	err = multiCopy(targetURLs, sourceURL)
	if err != nil {
		fatal(err.Error())
		return
	}

	return
}
