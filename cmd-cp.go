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

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

type message struct {
	err error
}

func multiTargetGoroutine(targetURL string, md5Hex string, sourceSize int64, targetReader io.Reader, ch chan message) {
	errParams := map[string]string{
		"url": targetURL,
	}
	targetBucket, targetObject, err := url2Object(targetURL)
	if err != nil {
		ch <- message{err: iodine.New(err, errParams)}
	}
	targetClnt, err := getNewClient(globalDebugFlag, targetURL)
	if err != nil {
		ch <- message{err: iodine.New(err, errParams)}
	}
	if err := targetClnt.Put(targetBucket, targetObject, md5Hex, sourceSize, targetReader); err != nil {
		ch <- message{err: iodine.New(err, errParams)}
	}
	ch <- message{err: nil}
	close(ch)
}

func doPutMultiTarget(targetURLs []string, md5Hex string, sourceSize int64, targetReaders []io.Reader) <-chan message {
	ch := make(chan message)
	for i := 0; i < len(targetURLs); i++ {
		go multiTargetGoroutine(targetURLs[i], md5Hex, sourceSize, targetReaders[i], ch)
	}
	return ch
}

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
	defer sourceReader.Close()

	targetReaders := make([]io.Reader, numTargets)
	targetWriters := make([]io.WriteCloser, numTargets)

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
		// convert targetWriters to []io.Writer
		multiWriters := make([]io.Writer, numTargets)
		for i, writeCloser := range targetWriters {
			defer writeCloser.Close()
			multiWriters[i] = io.Writer(writeCloser)
		}
		io.CopyN(io.MultiWriter(multiWriters...), sourceReader, sourceSize)
		if !globalQuietFlag {
			bar.Finish()
		}
	}()

	for msg := range doPutMultiTarget(targetURLs, md5Hex, sourceSize, targetReaders) {
		if msg.err != nil {
			if globalDebugFlag {
				log.Debug.Println(msg.err)
			}
			fatal(msg.err)
		}
		info("Done")
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
		if globalDebugFlag {
			log.Debug.Println(iodine.New(err, nil))
		}
		fatal(err)
		return
	}
	sourceURL := urlList[0]   // First arg is source
	targetURLs := urlList[1:] // 1 or more targets

	err = multiCopy(targetURLs, sourceURL)
	if err != nil {
		if globalDebugFlag {
			log.Debug.Println(iodine.New(err, nil))
		}
		fatal(err)
		return
	}
	return
}
