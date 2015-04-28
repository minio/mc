/*
 * Mini Copy (C) 2015 Minio, Inc.
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
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// getObjectKey - get object name from source url
func getObjectKey(sourceURL string) (objectName string) {
	u, _ := url.Parse(sourceURL)
	splits := strings.SplitN(u.Path, "/", 3)
	if len(splits) == 3 {
		return splits[2]
	}
	return ""
}

// getNewURLRecursive - get new source and target URL for recursive copy
func getNewURLRecursive(sourceURL, targetURL, url string) (newSourceURL, newTargetURL string) {
	switch client.GetType(sourceURL) {
	case client.Object:
		sourceURL = strings.TrimSuffix(sourceURL, getObjectKey(sourceURL))
		newSourceURL = strings.TrimSuffix(sourceURL, pathSeparator) + pathSeparator + url
		newTargetURL = strings.TrimSuffix(targetURL, pathSeparator) + pathSeparator + url
	case client.Filesystem:
		newSourceURL = url
		prefix := strings.TrimSuffix(sourceURL, pathSeparator) + pathSeparator
		newTargetURL = strings.TrimSuffix(targetURL, pathSeparator) + pathSeparator + strings.TrimPrefix(url, prefix)
	}
	return newSourceURL, newTargetURL
}

// getNewTargetURL - construct a new target URL object/fs alike
func getNewTargetURL(targetURL string, sourceURL string) (newTargetURL string, err error) {
	switch client.GetType(targetURL) {
	case client.Object:
		return getNewTargetURLObject(targetURL, sourceURL)
	case client.Filesystem:
		return getNewTargetURLFilesystem(targetURL, sourceURL)
	}
	return "", iodine.New(errInvalidURL{url: targetURL}, nil)
}

// getNewTargetURLObject - construct a new targetURL for object API
func getNewTargetURLObject(targetURL string, sourceURL string) (newTargetURL string, err error) {
	if getObjectKey(targetURL) != "" {
		return "", iodine.New(errIsNotBucket{path: targetURL}, nil)
	}
	switch client.GetType(sourceURL) {
	case client.Filesystem:
		_, s := filepath.Split(sourceURL)
		if s == "" {
			return "", iodine.New(errInvalidSource{path: sourceURL}, nil)
		}
		newTargetURL = strings.TrimSuffix(targetURL, pathSeparator) + pathSeparator + s
	case client.Object:
		object := getObjectKey(sourceURL)
		if object == "" {
			return "", iodine.New(errInvalidSource{path: sourceURL}, nil)
		}
		_, s := filepath.Split(object)
		newTargetURL = filepath.Join(targetURL, s)
	}
	return newTargetURL, nil
}

// getNewTargetURLFilesystem - construct a new targetURL for fs API
func getNewTargetURLFilesystem(targetURL string, sourceURL string) (newTargetURL string, err error) {
	st, err := os.Stat(targetURL)
	if err != nil {
		return "", iodine.New(errIsNotDIR{path: targetURL}, nil)
	}
	if !st.IsDir() {
		return "", iodine.New(errIsNotDIR{path: targetURL}, nil)
	}
	switch client.GetType(sourceURL) {
	case client.Filesystem:
		_, s := filepath.Split(sourceURL)
		if s == "" {
			return "", iodine.New(errInvalidSource{path: sourceURL}, nil)
		}
		newTargetURL = filepath.Join(targetURL, s)
	case client.Object:
		object := getObjectKey(sourceURL)
		if object == "" {
			return "", iodine.New(errInvalidSource{path: sourceURL}, nil)
		}
		_, s := filepath.Split(object)
		newTargetURL = filepath.Join(targetURL, s)
	}
	return newTargetURL, nil
}
