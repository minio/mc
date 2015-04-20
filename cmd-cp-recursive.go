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
	"strings"

	"fmt"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

const (
	pathSeparator = "/"
)

// getSourceObjectURLMap - get list of all source object and its URLs in a map
func getSourceObjectURLMap(manager clientManager, sourceURL string) (sourceObjectURLMap map[string]string, err error) {
	sourceClnt, err := manager.getNewClient(sourceURL, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	objects, err := sourceClnt.List()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	sourceObjectURLMap = make(map[string]string)
	for _, object := range objects {
		sourceObjectURLMap[object.Name] = strings.TrimSuffix(sourceURL, pathSeparator) + pathSeparator + object.Name
	}
	return sourceObjectURLMap, nil
}

// getRecursiveTargetWriter - recursively get target writers to initiate copying data
func getRecursiveTargetWriter(manager clientManager, targetURL, md5Hex string, length int64) (io.WriteCloser, error) {
	targetClnt, err := manager.getNewClient(targetURL, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"URL": targetURL})
	}

	// For object storage URL's do a StatBucket() and PutBucket(), not necessary for fs client
	if client.GetURLType(targetURL) != client.URLFilesystem {
		// check if bucket is valid, if not found create it
		if err := targetClnt.StatBucket(); err != nil {
			switch iodine.ToError(err).(type) {
			case client.BucketNotFound:
				err := targetClnt.PutBucket()
				if err != nil {
					iodine.New(err, map[string]string{"URL": targetURL})
				}
			default:
				iodine.New(err, map[string]string{"URL": targetURL})
			}
		}
	}
	return targetClnt.Put(md5Hex, length)
}

// getRecursiveTargetWriters - convenient wrapper around getRecursiveTarget
func getRecursiveTargetWriters(manager clientManager, targetURLs []string, md5Hex string, length int64) ([]io.WriteCloser, error) {
	var targetWriters []io.WriteCloser
	for _, targetURL := range targetURLs {
		writer, err := getRecursiveTargetWriter(manager, targetURL, md5Hex, length)
		if err != nil {
			// on error close all writers
			for _, targetWriter := range targetWriters {
				targetWriter.Close()
			}
			// return error so that Multiwriter doesn't get control
			return nil, iodine.New(err, nil)
		}
		targetWriters = append(targetWriters, writer)
	}
	return targetWriters, nil
}

// doCopyCmdRecursive - copy bucket to bucket
func doCopyCmdRecursive(manager clientManager, sourceURL string, targetURLs []string) (string, error) {
	sourceObjectURLMap, err := getSourceObjectURLMap(manager, sourceURL)
	if err != nil {
		err = iodine.New(err, nil)
		msg := fmt.Sprintf("mc: Getting list of objects from source URL: [%s] failed with following reason: [%s]\n",
			sourceURL, iodine.ToError(err))
		return msg, err
	}
	// do not exit, continue even for failures
	for sourceObjectName, sourceObjectURL := range sourceObjectURLMap {
		sourceClnt, err := manager.getNewClient(sourceObjectURL, globalDebugFlag)
		if err != nil {
			err := iodine.New(err, nil)
			msg := fmt.Sprintf("mc: instantiating a new client for URL [%s] failed with following reason: [%s]\n",
				sourceObjectURL, iodine.ToError(err))
			return msg, err
		}
		reader, length, md5hex, err := sourceClnt.Get()
		if err != nil {
			err = iodine.New(err, nil)
			msg := fmt.Sprintf("mc: Reading from source URL: [%s] failed with following reason: [%s]\n",
				sourceURL, iodine.ToError(err))
			return msg, err
		}
		// Construct full target URL path based on source object name
		var newTargetURLs []string
		for _, targetURL := range targetURLs {
			targetURL := strings.TrimSuffix(targetURL, pathSeparator) + pathSeparator + sourceObjectName
			newTargetURLs = append(newTargetURLs, targetURL)
		}
		writeClosers, err := getRecursiveTargetWriters(manager, newTargetURLs, md5hex, length)
		if err != nil {
			err = iodine.New(err, nil)
			msg := fmt.Sprintf("mc: Writing to target URLs failed with following reason: [%s]\n", iodine.ToError(err))
			return msg, err
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
			err := iodine.New(err, nil)
			msg := fmt.Sprintf("mc: Copying data from source to target(s) failed with following reason: [%s]\n",
				iodine.ToError(err))
			return msg, err
		}

		// close writers
		for _, writer := range writeClosers {
			err := writer.Close()
			if err != nil {
				err := iodine.New(err, nil)
				msg := fmt.Sprintf("mc: Connections still active, one or more writes have failed with following reason: [%s]\n",
					iodine.ToError(err))
				return msg, err
			}
		}
		if !globalDebugFlag {
			bar.Finish()
			console.Infoln("Success!")
		}
	}
	return "", nil
}
