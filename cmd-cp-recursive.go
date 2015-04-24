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
	"fmt"
	"strings"

	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	pathSeparator = "/"
)

// doCopyCmdRecursive - copy bucket to bucket
func doCopyCmdRecursive(manager clientManager, sourceURLConfigMap map[string]*hostConfig, targetURLConfigMap map[string]*hostConfig) (string, error) {
	for sourceURL, sourceConfig := range sourceURLConfigMap {
		// get source list
		clnt, err := manager.getNewClient(sourceURL, sourceConfig, false)
		if err != nil {
			return fmt.Sprintf("instantiating a new client for URL [%s]", sourceURL), iodine.New(err, nil)
		}
		for itemCh := range clnt.List() {
			if itemCh.Err != nil {
				return fmt.Sprintf("listing objects failed for URL [%s]", sourceURL), iodine.New(itemCh.Err, nil)
			}
			if itemCh.Item.FileType.IsDir() {
				continue
			}
			// populate source urls
			sourceURLs := make(map[string]*hostConfig)
			sourceObjectURL := itemCh.Item.Name
			sourceURLs[sourceObjectURL] = sourceConfig
			// populate target urls
			targetURLs := make(map[string]*hostConfig)
			for targetURL, targetConfig := range targetURLConfigMap {
				targetObjectURL := strings.TrimSuffix(targetURL, pathSeparator) + pathSeparator + itemCh.Item.Name
				targetURLs[targetObjectURL] = targetConfig
			}
			humanReadable, err := doCopyCmd(manager, sourceURLs, targetURLs)
			if err != nil {
				err := iodine.New(err, nil)
				log.Debug.Println(err)
				console.Errorln(humanReadable)
			}
		}
	}
	return "", nil
}
