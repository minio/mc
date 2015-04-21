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
	"runtime"
	"strings"

	"fmt"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

const (
	pathSeparator        = "/"
	pathSeparatorWindows = "\\"
)

func getSourceURL(sourceURL, objectName string) string {
	if client.GetType(sourceURL) == client.Filesystem && runtime.GOOS == "windows" {
		return strings.TrimSuffix(sourceURL, pathSeparator) + pathSeparatorWindows + objectName
	}
	return strings.TrimSuffix(sourceURL, pathSeparator) + pathSeparator + objectName
}

// getSourceObjectURLMap - get list of all source object and its URLs in a map
func getSourceObjectURLMap(manager clientManager, sourceURL string, sourceConfig map[string]string) (
	sourceObjectURLMap map[string]string, err error) {

	sourceClnt, err := manager.getNewClient(sourceURL, sourceConfig, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	objects, err := sourceClnt.List()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	sourceObjectURLMap = make(map[string]string)
	for _, object := range objects {
		sourceObjectURLMap[object.Name] = getSourceURL(sourceURL, object.Name)
	}
	return sourceObjectURLMap, nil
}

// getRecursiveTargetWriter - recursively get target writers to initiate copying data
func getRecursiveTargetWriter(manager clientManager, targetURL string, targetConfig map[string]string, md5Hex string, length int64) (
	io.WriteCloser, error) {

	targetClnt, err := manager.getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"failedURL": targetURL})
	}

	// For object storage URL's do a StatBucket() and PutBucket(), not necessary for fs client
	if client.GetType(targetURL) != client.Filesystem {
		// check if bucket is valid, if not found create it
		if err := targetClnt.Stat(); err != nil {
			switch iodine.ToError(err).(type) {
			case client.BucketNotFound:
				err := targetClnt.PutBucket()
				if err != nil {
					return nil, iodine.New(err, map[string]string{"failedURL": targetURL})
				}
			default:
				return nil, iodine.New(err, map[string]string{"failedURL": targetURL})
			}
		}
	}
	return targetClnt.Put(md5Hex, length)
}

// getRecursiveTargetWriters - convenient wrapper around getRecursiveTarget
func getRecursiveTargetWriters(manager clientManager, targetURLConfigMap map[string]map[string]string, md5Hex string, length int64) (
	[]io.WriteCloser, error) {

	var targetWriters []io.WriteCloser
	for targetURL, targetConfig := range targetURLConfigMap {
		writer, err := getRecursiveTargetWriter(manager, targetURL, targetConfig, md5Hex, length)
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

type urlConfig map[string]map[string]string

// doCopyCmdRecursive - copy bucket to bucket
func doCopyCmdRecursive(manager clientManager, sourceURLConfigMap urlConfig, targetURLConfigMap urlConfig) (string, error) {
	for sourceURL, sourceConfig := range sourceURLConfigMap {
		sourceObjectURLMap, err := getSourceObjectURLMap(manager, sourceURL, sourceConfig)
		if err != nil {
			msg := fmt.Sprintf("Getting list of objects from source URL: [%s] failed", sourceURL)
			return msg, iodine.New(err, nil)
		}
		// do not exit, continue even for failures
		for sourceObjectName, sourceObjectURL := range sourceObjectURLMap {
			sourceClnt, err := manager.getNewClient(sourceObjectURL, sourceConfig, globalDebugFlag)
			if err != nil {
				err := iodine.New(err, nil)
				msg := fmt.Sprintf("Instantiating a new client for URL [%s] failed", sourceObjectURL)
				return msg, err
			}
			reader, length, md5hex, err := sourceClnt.Get()
			if err != nil {
				err = iodine.New(err, nil)
				msg := fmt.Sprintf("Reading from source URL: [%s] failed", sourceURL)
				return msg, err
			}
			newTargetURLConfigMap := make(map[string]map[string]string)
			// Construct full target URL path based on source object name
			for targetURL, targetConfig := range targetURLConfigMap {
				targetURL := strings.TrimSuffix(targetURL, pathSeparator) + pathSeparator + sourceObjectName
				newTargetURLConfigMap[targetURL] = targetConfig
			}
			writeClosers, err := getRecursiveTargetWriters(manager, newTargetURLConfigMap, md5hex, length)
			if err != nil {
				return "Writing to target URLs failed", iodine.New(err, nil)
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
				return "Copying data from source to target(s) failed", err
			}

			// close writers
			for _, writer := range writeClosers {
				err := writer.Close()
				if err != nil {
					err := iodine.New(err, nil)
					return "Connections still active, one or more writes have failed", err
				}
			}
			if !globalDebugFlag {
				bar.Finish()
				console.Infoln("Success!")
			}
		}
	}
	return "", nil
}
