/*
 * Minio Client (C) 2015 Minio, Inc.
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
	"os"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/minio/pkg/iodine"
)

/// mc cp - related internal functions

// doCopy
func doCopy(methods clientMethods, reader io.ReadCloser, md5hex string, length int64, targetURL string, targetConfig *hostConfig) error {
	writeCloser, err := methods.getTargetWriter(targetURL, targetConfig, md5hex, length)
	if err != nil {
		return iodine.New(err, nil)
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
	_, copyErr := io.CopyN(multiWriter, reader, length)
	// close to see the error, verify it later
	err = writeCloser.Close()
	if copyErr != nil {
		return iodine.New(copyErr, nil)
	}
	if err != nil {
		return iodine.New(err, nil)
	}
	if !globalQuietFlag {
		bar.Finish()
	}
	return nil
}

// doCopySingleSource
func doCopySingleSource(methods clientMethods, sourceURL, targetURL string, sourceConfig, targetConfig *hostConfig) error {
	reader, length, md5hex, err := methods.getSourceReader(sourceURL, sourceConfig)
	if err != nil {
		return iodine.New(err, nil)
	}
	// check if its a folder, construct the new TargetURL, if not fallback
	newTargetURL, err := getNewTargetURL(targetURL, sourceURL)
	switch iodine.ToError(err).(type) {
	case errIsNotFolder:
		return doCopy(methods, reader, md5hex, length, targetURL, targetConfig)
	case errIsNotBucket:
		return doCopy(methods, reader, md5hex, length, targetURL, targetConfig)
	case nil:
		return doCopy(methods, reader, md5hex, length, newTargetURL, targetConfig)
	default:
		return iodine.New(err, nil)
	}
}

// doCopySingleSourceRecursive
func doCopySingleSourceRecursive(methods clientMethods, sourceURL, targetURL string, sourceConfig, targetConfig *hostConfig) error {
	sourceClnt, err := methods.getNewClient(sourceURL, sourceConfig, globalDebugFlag)
	if err != nil {
		return iodine.New(err, nil)
	}
	for contentCh := range sourceClnt.ListRecursive() {
		if contentCh.Err != nil {
			continue
		}
		newSourceURL, newTargetURL := getNewURLRecursive(sourceURL, targetURL, contentCh.Content.Name)
		if err := doCopySingleSource(methods, newSourceURL, newTargetURL, sourceConfig, targetConfig); err != nil {
			// verify for directory related errors, if "open" failed on directories ignore those errors
			switch e := iodine.ToError(err).(type) {
			case *os.PathError:
				switch true {
				// even with in PathError specific error related to directory reads is ignored
				// do not ignore any other errors, since they might be valid problems on the filesystem
				case e.Op == "read" && e.Err.Error() == "is a directory":
					continue
				default:
					return iodine.New(err, nil)
				}
			default:
				return iodine.New(err, nil)
			}
		}
	}
	return nil
}

// doCopyMultipleSources -
func doCopyMultipleSources(methods clientMethods, sourceURLConfigMap map[string]*hostConfig, targetURL string, targetConfig *hostConfig) error {
	sourceURLReaderMap, err := getSourceReaders(methods, sourceURLConfigMap)
	if err != nil {
		return iodine.New(err, nil)
	}
	for sourceURL, sourceReader := range sourceURLReaderMap {
		newTargetURL, err := getNewTargetURL(targetURL, sourceURL)
		if err != nil {
			return iodine.New(err, nil)
		}
		err = doCopy(methods, sourceReader.reader, sourceReader.md5hex, sourceReader.length, newTargetURL, targetConfig)
		if err != nil {
			return iodine.New(err, map[string]string{"Source": sourceURL})
		}
	}
	return nil
}
