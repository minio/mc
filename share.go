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
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio/pkg/probe"
	"github.com/minio/minio/pkg/quick"
)

type sharedURLs struct {
	Version string
	URLs    map[string]struct {
		Date    time.Time
		Message shareMessage
	}
}

func createSharedURLsDatadir() *probe.Error {
	shareDir, err := getSharedURLsDatadir()
	if err != nil {
		return err.Trace()
	}

	if err := os.MkdirAll(shareDir, 0700); err != nil {
		return probe.NewError(err)
	}
	return nil
}

func getSharedURLsDatadir() (string, *probe.Error) {
	configDir, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}

	sharedURLsDatadir := filepath.Join(configDir, globalSharedURLsDatadir)
	return sharedURLsDatadir, nil
}

func isSharedURLsDatadirExists() bool {
	shareDir, err := getSharedURLsDatadir()
	fatalIf(err.Trace(), "Unable to determine share folder.")

	if _, e := os.Stat(shareDir); e != nil {
		return false
	}
	return true
}

func getSharedURLsDataFile() (string, *probe.Error) {
	sharedURLsDatadir, err := getSharedURLsDatadir()
	if err != nil {
		return "", err.Trace()
	}

	sharedURLsDataFile := filepath.Join(sharedURLsDatadir, "urls.json")
	return sharedURLsDataFile, nil
}

func loadSharedURLsV1() (*sharedURLs, *probe.Error) {
	sharedURLsDataFile, err := getSharedURLsDataFile()
	if err != nil {
		return nil, err.Trace()
	}
	if _, err := os.Stat(sharedURLsDataFile); err != nil {
		return nil, probe.NewError(err)
	}

	s := &sharedURLs{}
	s.Version = "1.0.0"

	qs, err := quick.New(s)
	if err != nil {
		return nil, err.Trace()
	}
	err = qs.Load(sharedURLsDataFile)
	if err != nil {
		return nil, err.Trace(sharedURLsDataFile)
	}
	s = qs.Data().(*sharedURLs)
	return s, nil
}

func saveSharedURLsV1(s *sharedURLs) *probe.Error {
	qs, err := quick.New(s)
	if err != nil {
		return err.Trace()
	}
	sharedURLsDataFile, err := getSharedURLsDataFile()
	if err != nil {
		return err.Trace()
	}
	return qs.Save(sharedURLsDataFile).Trace(sharedURLsDataFile)
}
