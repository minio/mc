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

	"bytes"
	"sync"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

// doCopyCmd copies objects into and from a bucket or between buckets
func multiCopy(targetURLs []string, sourceURL string) (err error) {
	var targetBuffer bytes.Buffer
	fmt.Fprint(&targetBuffer, targetURLs)
	errParams := map[string]string{
		"targetURLs": targetBuffer.String(),
		"sourceURL":  sourceURL,
	}
	numTargets := len(targetURLs)

	// Parse URL to bucket and object names
	sourceBucket, sourceObject, err := url2Object(sourceURL)
	if err != nil {
		return iodine.New(err, errParams)
	}

	// Initialize a new client object for the source
	sourceClnt, err := getNewClient(globalDebugFlag, sourceURL)
	if err != nil {
		return iodine.New(err, errParams)
	}

	// Get a reader for the source object
	sourceReader, sourceSize, md5Hex, err := sourceClnt.Get(sourceBucket, sourceObject)
	if err != nil {
		return iodine.New(err, errParams)
	}

	targetReaders := make([]io.Reader, numTargets)
	targetWriters := make([]io.Writer, numTargets)

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

	var wg sync.WaitGroup
	for i := 0; i < numTargets; i++ {
		targetBucket, targetObject, err := url2Object(targetURLs[i])
		if err != nil {
			return iodine.New(err, errParams)
		}
		targetClnt, err := getNewClient(globalDebugFlag, targetURLs[i])
		if err != nil {
			return iodine.New(err, errParams)
		}
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := targetClnt.Put(targetBucket, targetObject, md5Hex, sourceSize, targetReaders[index]); err != nil {
				fatal(iodine.New(err, nil).Error())
			}
			info(fmt.Sprintf("Done: %s", targetURLs[index]))
		}(i)
	}
	wg.Wait()
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
		log.Fatalln(iodine.New(err, nil))
		return
	}
	sourceURL := urlList[0]   // First arg is source
	targetURLs := urlList[1:] // 1 or more targets

	err = multiCopy(targetURLs, sourceURL)
	if err != nil {
		log.Fatalln(iodine.New(err, nil))
		return
	}

	return
}
