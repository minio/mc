/*
 * Mini Copy, (C) 2014,2015 Minio, Inc.
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
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

type clientManager interface {
	getSourceReader(urlStr string) (reader io.ReadCloser, length int64, md5hex string, err error)
	getTargetWriter(urlStr string, md5Hex string, length int64) (io.WriteCloser, error)
}

type mcClientManager struct{}

func (manager mcClientManager) getSourceReader(urlStr string) (reader io.ReadCloser, length int64, md5hex string, err error) {
	sourceClnt, err := getNewClient(urlStr, globalDebugFlag)
	if err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"sourceURL": urlStr})
	}
	// Get a reader for the source object
	bucket, object, err := url2Object(urlStr)
	if err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"sourceURL": urlStr})
	}

	// check if the bucket is valid
	if err := sourceClnt.StatBucket(bucket); err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"sourceURL": urlStr})
	}
	return sourceClnt.Get(bucket, object)
}

func (manager mcClientManager) getTargetWriter(urlStr string, md5Hex string, length int64) (io.WriteCloser, error) {
	targetClnt, err := getNewClient(urlStr, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	bucket, object, err := url2Object(urlStr)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	// check if bucket is valid
	if err := targetClnt.StatBucket(bucket); err != nil {
		return nil, iodine.New(err, map[string]string{"failedURL": urlStr})
	}
	return targetClnt.Put(bucket, object, md5Hex, length)
}

func getTargetWriters(manager clientManager, urls []string, md5Hex string, length int64) ([]io.WriteCloser, error) {
	var targetWriters []io.WriteCloser
	for _, u := range urls {
		writer, err := manager.getTargetWriter(u, md5Hex, length)
		if err != nil {
			// close all writers
			for _, targetWriter := range targetWriters {
				targetWriter.Close()
			}
			return nil, iodine.New(errInvalidURL{url: u}, nil)
		}
		targetWriters = append(targetWriters, writer)
	}
	return targetWriters, nil
}

// doCopyCmd copies objects into and from a bucket or between buckets
func runCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
	}

	config, err := loadMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: Unable to load config file")
	}

	// Convert arguments to URLs: expand alias, fix format...
	urls, err := parseURLs(ctx.Args(), config.Aliases)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: Unable to parse URL")
	}
	sourceURL := urls[0]   // First arg is source
	targetURLs := urls[1:] // 1 or more targets

	// perform copy
	if ctx.Bool("recursive") {
		doCopyCmdRecursive(ctx)
	} else {
		humanReadableError, err := doCopyCmd(mcClientManager{}, sourceURL, targetURLs)
		err = iodine.New(err, nil)
		if err != nil {
			if humanReadableError == "" {
				humanReadableError = "No error message present, please rerun with --debug and report a bug."
			}
			log.Debug.Println(err)
			console.Errorln("mc: " + humanReadableError)
		}
	}
}

func doCopyCmd(manager clientManager, sourceURL string, targetURLs []string) (string, error) {
	reader, length, hexMd5, err := manager.getSourceReader(sourceURL)
	if err != nil {
		return "Unable to read from source", iodine.New(err, nil)
	}
	defer reader.Close()

	writeClosers, err := getTargetWriters(manager, targetURLs, hexMd5, length)
	if err != nil {
		return "Unable to write to target", iodine.New(err, nil)
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
		return "Copying data from source to target(s) failed", iodine.New(err, nil)
	}

	// close writers
	for _, writer := range writeClosers {
		err := writer.Close()
		if err != nil {
			err = iodine.New(err, nil)
		}
	}
	if err != nil {
		return "Unable to close all connections, write may of failed.", iodine.New(err, nil)
	}
	return "", nil
}
