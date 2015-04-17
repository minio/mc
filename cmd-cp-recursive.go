/*
 * Mini Copy, (C) 2015 Minio, Inc.
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

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

func getSourceObjectList(sourceClnt client.Client, urlStr string) (bucket string, objects []*client.Item, err error) {
	bucket, object, err := url2Object(urlStr)
	if err != nil {
		return "", nil, iodine.New(err, nil)
	}

	objects, err = sourceClnt.ListObjects(bucket, object)
	if err != nil {
		return "", nil, iodine.New(err, nil)
	}
	return bucket, objects, nil
}

func getRecursiveTargetWriter(urlStr, md5Hex string, length int64) (io.WriteCloser, error) {
	targetClnt, err := getNewClient(urlStr, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"URL": urlStr})
	}

	bucket, object, err := url2Object(urlStr)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"URL": urlStr})
	}

	// check if bucket is valid, if not found create it
	if err := targetClnt.StatBucket(bucket); err != nil {
		switch iodine.ToError(err).(type) {
		case client.BucketNotFound:
			err := targetClnt.PutBucket(bucket)
			if err != nil {
				iodine.New(err, map[string]string{"URL": urlStr})
			}
		default:
			iodine.New(err, map[string]string{"URL": urlStr})

		}
	}
	return targetClnt.Put(bucket, object, md5Hex, length)
}

func getRecursiveTargetWriters(urls []string, md5Hex string, length int64) ([]io.WriteCloser, error) {
	var targetWriters []io.WriteCloser
	for _, url := range urls {
		writer, err := getRecursiveTargetWriter(url, md5Hex, length)
		if err != nil {
			// close all writers
			for _, targetWriter := range targetWriters {
				targetWriter.Close()
			}

		}
		targetWriters = append(targetWriters, writer)
	}
	return targetWriters, nil
}

// doCopyCmdRecursive - copy bucket to bucket
func doCopyCmdRecursive(ctx *cli.Context) {
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: unable to load config")
	}

	// Convert arguments to URLs: expand alias, fix format...
	urls, err := parseURLs(ctx.Args(), config.Aliases)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: unable to parse urls: %s\n", err)
	}
	sourceURL := urls[0]   // First arg is source
	targetURLs := urls[1:] // Rest are targets

	sourceClnt, err := getNewClient(sourceURL, globalDebugFlag)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: unable to get source: %s\n", err)
	}
	sourceBucket, sourceObjectList, err := getSourceObjectList(sourceClnt, sourceURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: unable to list source objects: %s\n", err)
	}

	// do not exit, continue even for failures
	for _, sourceObject := range sourceObjectList {
		reader, length, md5hex, err := sourceClnt.Get(sourceBucket, sourceObject.Key)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorf("mc: unable to read source: %s\n", err)
		}
		// Construct full target URL path based on source object name
		var newTargetURLs []string
		for _, targetURL := range targetURLs {
			targetURL := targetURL + sourceObject.Key
			// TODO: strip trailing slashes and properly append
			newTargetURLs = append(newTargetURLs, targetURL)
		}
		writeClosers, err := getRecursiveTargetWriters(newTargetURLs, md5hex, length)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorf("mc: unable to read target: %s\n", err)
		}

		var writers []io.Writer
		for _, writer := range writeClosers {
			writers = append(writers, writer)
		}

		// set up progress bar
		var bar *pb.ProgressBar
		if !globalQuietFlag {
			bar = startBar(length)
			bar.Start()
			writers = append(writers, bar)
		}

		// write progress bar
		multiWriter := io.MultiWriter(writers...)

		// copy data to writers
		_, err = io.CopyN(multiWriter, reader, length)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorln("mc: Unable to write to target")
		}

		// close writers
		for _, writer := range writeClosers {
			err := writer.Close()
			if err != nil {
				log.Debug.Println(iodine.New(err, nil))
				console.Errorln("mc: Unable to close writer, object may not of written properly.")
			}
		}
	}
	return
}
