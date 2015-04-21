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
	"strings"

	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	pathSeparator = "/"

//	pathSeparatorWindows = "\\"
)

// doCopyCmdRecursive - copy bucket to bucket
func doCopyCmdRecursive(manager clientManager, sourceURLConfigMap map[string]*hostConfig, targetURLConfigMap map[string]*hostConfig) (string, error) {
	for sourceURL, sourceConfig := range sourceURLConfigMap {
		// get source list
		clnt, err := manager.getNewClient(sourceURL, sourceConfig, false)
		log.Debug.Println(iodine.New(err, nil))
		items, err := clnt.List()
		log.Debug.Println(iodine.New(err, nil))
		for _, item := range items {
			// get source url
			sourceURLs := make(map[string]*hostConfig)
			sourceObjectURL := strings.TrimSuffix(sourceURL, pathSeparator) + pathSeparator + item.Name
			sourceURLs[sourceObjectURL] = sourceConfig
			// get target urls
			targetURLs := make(map[string]*hostConfig)
			for targetURL, targetConfig := range targetURLConfigMap {
				targetObjectURL := strings.TrimSuffix(targetURL, pathSeparator) + pathSeparator + item.Name
				targetURLs[targetObjectURL] = targetConfig
			}
			humanReadable, err := doCopyCmd(manager, sourceURLs, targetURLs)
			// TODO work out how to report errors
			log.Debug.Println(humanReadable)
			log.Debug.Println(iodine.New(err, nil))
		}
	}
	return "", nil
}
