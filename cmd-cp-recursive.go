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
	"github.com/minio-io/minio/pkg/iodine"
)

// getSourceObjectList - get list of all source objects to copy
func getSourceObjectList(sourceClnt client.Client, urlStr string) (bucket string, objects []*client.Item, err error) {
	bucket, object, err := client.URL2Object(urlStr)
	if err != nil {
		return "", nil, iodine.New(err, nil)
	}

	objects, err = sourceClnt.ListObjects(bucket, object)
	if err != nil {
		return "", nil, iodine.New(err, nil)
	}
	return bucket, objects, nil
}

// getRecursiveTargetWriter - recursively get target writers to initiate copying data
func getRecursiveTargetWriter(manager clientManager, urlStr, md5Hex string, length int64) (io.WriteCloser, error) {
	targetClnt, err := manager.getNewClient(urlStr, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"URL": urlStr})
	}
	bucket, object, err := client.URL2Object(urlStr)
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

// getRecursiveTargetWriters - convenient wrapper around getRecursiveTarget
func getRecursiveTargetWriters(manager clientManager, urls []string, md5Hex string, length int64) ([]io.WriteCloser, error) {
	var targetWriters []io.WriteCloser
	for _, url := range urls {
		writer, err := getRecursiveTargetWriter(manager, url, md5Hex, length)
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
func doCopyCmdRecursive(manager clientManager, sourceURL string, targetURLs []string) (string, error) {
	sourceClnt, err := manager.getNewClient(sourceURL, globalDebugFlag)
	if err != nil {
		err = iodine.New(err, nil)
		humanReadable := fmt.Sprintf("mc: Reading from source URL: [%s] failed with following reason: [%s]\n", sourceURL, iodine.ToError(err))
		return humanReadable, err
	}
	sourceBucket, sourceObjectList, err := getSourceObjectList(sourceClnt, sourceURL)
	if err != nil {
		err = iodine.New(err, nil)
		humanReadable := fmt.Sprintf("mc: listing objects for URL [%s] failed with following reason: [%s]\n", sourceURL, iodine.ToError(err))
		return humanReadable, err
	}

	// do not exit, continue even for failures
	for _, sourceObject := range sourceObjectList {
		reader, length, md5hex, err := sourceClnt.Get(sourceBucket, sourceObject.Key)
		if err != nil {
			err = iodine.New(err, nil)
			humanReadable := fmt.Sprintf("mc: Reading from source URL: [%s] failed with following reason: [%s]\n", sourceURL, iodine.ToError(err))
			return humanReadable, err
		}
		// Construct full target URL path based on source object name
		var newTargetURLs []string
		for _, targetURL := range targetURLs {
			targetURL := strings.TrimSuffix(targetURL, "/") + "/" + sourceObject.Key
			newTargetURLs = append(newTargetURLs, targetURL)
		}
		writeClosers, err := getRecursiveTargetWriters(manager, newTargetURLs, md5hex, length)
		if err != nil {
			err = iodine.New(err, nil)
			humanReadable := fmt.Sprintf("mc: Writing to target URLs failed with following reason: [%s]\n", iodine.ToError(err))
			return humanReadable, err
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
			humanReadable := fmt.Sprintf("mc: Copying data from source to target(s) failed with following reason: [%s]\n", iodine.ToError(err))
			return humanReadable, err
		}

		// close writers
		for _, writer := range writeClosers {
			err := writer.Close()
			if err != nil {
				err := iodine.New(err, nil)
				humanReadable := fmt.Sprintf("mc: Connections still active, one or more writes have failed with following reason: [%s]\n", iodine.ToError(err))
				return humanReadable, err
			}
		}
	}
	return "", nil
}
