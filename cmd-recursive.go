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

func getSourceObjectList(sourceClnt client.Client, sourceURLParser *parsedURL) ([]*client.Item, error) {
	sourceObjects, err := sourceClnt.ListObjects(sourceURLParser.bucketName, sourceURLParser.objectName)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return sourceObjects, nil
}

func getRecursiveTargetWriter(targetURLParser *parsedURL, targetObject, md5Hex string, length int64) (io.WriteCloser, error) {
	targetClnt, err := getNewClient(targetURLParser, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"failedURL": targetURLParser.String()})
	}
	targetBucket := targetURLParser.bucketName
	// check if bucket is valid, if not found create it
	if err := targetClnt.StatBucket(targetBucket); err != nil {
		switch iodine.ToError(err).(type) {
		case client.BucketNotFound:
			err := targetClnt.PutBucket(targetBucket)
			if err != nil {
				return nil, iodine.New(err, map[string]string{"failedURL": targetURLParser.String()})
			}
		default:
			return nil, iodine.New(err, map[string]string{"failedURL": targetURLParser.String()})
		}
	}
	return targetClnt.Put(targetBucket, targetObject, md5Hex, length)
}

// doCopyCmdRecursive - copy bucket to bucket
func doCopyCmdRecursive(ctx *cli.Context) {
	// Convert arguments to URLs: expand alias, fix format...
	urlParsers, err := parseURLs(ctx)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: unable to parse urls: %s\n", err)
	}
	sourceURLParser := urlParsers[0] // First arg is source
	targetURLParser := urlParsers[1] // 1 target for now - TODO(y4m4): 2 or more targets

	sourceClnt, err := getNewClient(sourceURLParser, globalDebugFlag)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: unable to get source: %s\n", err)
	}
	sourceObjectList, err := getSourceObjectList(sourceClnt, sourceURLParser)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: unable to list source objects: %s\n", err)
	}
	// do not exit, continue even for failures
	for _, sourceObject := range sourceObjectList {
		reader, length, md5hex, err := sourceClnt.Get(sourceURLParser.bucketName, sourceObject.Key)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorf("mc: unable to read source: %s\n", err)
		}
		writeCloser, err := getRecursiveTargetWriter(targetURLParser, sourceObject.Key, md5hex, length)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorf("mc: unable to read target: %s\n", err)
		}
		var writers []io.Writer
		writers = append(writers, writeCloser)

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
		err = writeCloser.Close()
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorln("mc: Unable to close writer, object may not of written properly.")
		}
	}
	return
}
