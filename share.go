/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/minio/minio/pkg/probe"
)

func newSharedURLs() *sharedURLsV2 {
	return newSharedURLsV2()
}

func migrateSharedURLsV1ToV2() {
	if !isSharedURLsDataFileExists() {
		return
	}

	// try to load latest version if possible
	sURLsV2, err := loadSharedURLsV2()
	if err != nil {
		switch err.ToGoError().(type) {
		case *json.UnmarshalTypeError:
			// try to load V1 if possible
			var sURLsV1 *sharedURLsV1
			sURLsV1, err = loadSharedURLsV1()
			fatalIf(err.Trace(), "Unable to load shared url version ‘1.0.0’.")
			if sURLsV1.Version != "1.0.0" {
				fatalIf(errDummy().Trace(), "Invalid version loaded ‘"+sURLsV1.Version+"’.")
			}
			sURLsV2 = newSharedURLsV2()
			for key, value := range sURLsV1.URLs {
				value.Message.Key = key
				sURLsV2.URLs = append(sURLsV2.URLs, value)
			}
			err = saveSharedURLsV2(sURLsV2)
			fatalIf(err.Trace(), "Unable to save new shared url version ‘1.1.0’.")
		default:
			fatalIf(err.Trace(), "Unable to load shared url version ‘1.1.0’.")
		}
	}
}

func getSharedURLsDataDir() (string, *probe.Error) {
	configDir, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}

	sharedURLsDataDir := filepath.Join(configDir, globalSharedURLsDataDir)
	return sharedURLsDataDir, nil
}

func isSharedURLsDataDirExists() bool {
	shareDir, err := getSharedURLsDataDir()
	fatalIf(err.Trace(), "Unable to determine share folder.")

	if _, e := os.Stat(shareDir); e != nil {
		return false
	}
	return true
}

func createSharedURLsDataDir() *probe.Error {
	shareDir, err := getSharedURLsDataDir()
	if err != nil {
		return err.Trace()
	}

	if err := os.MkdirAll(shareDir, 0700); err != nil {
		return probe.NewError(err)
	}
	return nil
}

func getSharedURLsDataFile() (string, *probe.Error) {
	shareDir, err := getSharedURLsDataDir()
	if err != nil {
		return "", err.Trace()
	}

	shareFile := filepath.Join(shareDir, "urls.json")
	return shareFile, nil
}

func isSharedURLsDataFileExists() bool {
	shareFile, err := getSharedURLsDataFile()
	fatalIf(err.Trace(), "Unable to determine share filename.")

	if _, e := os.Stat(shareFile); e != nil {
		return false
	}
	return true
}

func createSharedURLsDataFile() *probe.Error {
	if err := saveSharedURLsV2(newSharedURLs()); err != nil {
		return err.Trace()
	}
	return nil
}
