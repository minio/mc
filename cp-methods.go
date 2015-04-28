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

	"github.com/minio-io/minio/pkg/iodine"
)

// cpMethods - methods only valid for cp
type cpMethods interface {
	getSourceReader(sourceURL string, sourceConfig *hostConfig) (reader io.ReadCloser, length int64, md5hex string, err error)
	getTargetWriter(targetURL string, targetConfig *hostConfig, md5Hex string, length int64) (io.WriteCloser, error)
}

// getSourceReaders -
func getSourceReaders(methods cpMethods, sourceURLConfigMap map[string]*hostConfig) (map[string]sourceReader, error) {
	sourceURLReaderMap := make(map[string]sourceReader)
	for sourceURL, sourceConfig := range sourceURLConfigMap {
		reader, length, md5hex, err := methods.getSourceReader(sourceURL, sourceConfig)
		if err != nil {
			for _, sourceReader := range sourceURLReaderMap {
				sourceReader.reader.Close()
			}
			return nil, iodine.New(err, nil)
		}
		sr := sourceReader{
			reader: reader,
			length: length,
			md5hex: md5hex,
		}
		sourceURLReaderMap[sourceURL] = sr
	}
	return sourceURLReaderMap, nil
}
