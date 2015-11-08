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
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio-xl/pkg/quick"
)

func shareDataDirSetup() {
	if !isSharedURLsDataDirExists() {
		shareDir, err := getSharedURLsDataDir()
		fatalIf(err.Trace(), "Unable to get shared URL data folder.")

		fatalIf(createSharedURLsDataDir().Trace(), "Unable to create shared URL data folder ‘"+shareDir+"’.")
	}
	if !isSharedURLsDataFileExists() {
		shareFile, err := getSharedURLsDataFile()
		fatalIf(err.Trace(), "Unable to get shared URL data file")

		fatalIf(createSharedURLsDataFile().Trace(), "Unable to create shared URL data file ‘"+shareFile+"’.")
	}
}

// migrateSharedURLs migrate to newest version sequentially
func migrateSharedURLs() {
	migrateSharedURLsV1ToV2()
	migrateSharedURLsV2ToV3()
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
				entry := struct {
					Date    time.Time
					Message ShareMessageV2
				}{
					Date: value.Date,
					Message: ShareMessageV2{
						Expiry: value.Message.Expiry,
						URL:    value.Message.URL,
						Key:    value.Message.Key,
					},
				}
				sURLsV2.URLs = append(sURLsV2.URLs, entry)
			}
			err = saveSharedURLsV2(sURLsV2)
			fatalIf(err.Trace(), "Unable to save new shared url version ‘1.1.0’.")
		default:
			fatalIf(err.Trace(), "Unable to load shared url version ‘1.1.0’.")
		}
	}
}

func migrateSharedURLsV2ToV3() {
	if !isSharedURLsDataFileExists() {
		return
	}
	conffile, err := getSharedURLsDataFile()
	if err != nil {
		return
	}
	v3, err := quick.CheckVersion(conffile, "3")
	if err != nil {
		fatalIf(err.Trace(), "Unable to check version on share list file")
	}
	if v3 {
		return
	}

	// try to load V2 if possible
	sURLsV2, err := loadSharedURLsV2()
	fatalIf(err.Trace(), "Unable to load shared url version ‘1.1.0’.")
	if sURLsV2.Version != "1.1.0" {
		fatalIf(errDummy().Trace(), "Invalid version loaded ‘"+sURLsV2.Version+"’.")
	}
	sURLsV3 := newSharedURLsV3()
	for _, value := range sURLsV2.URLs {
		entry := struct {
			Date    time.Time
			Message ShareMessageV3
		}{
			Date: value.Date,
			Message: ShareMessageV3{
				Expiry:      value.Message.Expiry,
				DownloadURL: value.Message.URL,
				Key:         value.Message.Key,
			},
		}
		sURLsV3.URLs = append(sURLsV3.URLs, entry)
	}
	err = saveSharedURLsV3(sURLsV3)
	fatalIf(err.Trace(), "Unable to save new shared url version ‘1.2.0’.")
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
	if err := saveSharedURLsV3(newSharedURLsV3()); err != nil {
		return err.Trace()
	}
	return nil
}

// this code is necessary since, share only operates on cloud storage URLs not filesystem
func isObjectKeyPresent(url string) bool {
	u := client.NewURL(url)
	path := u.Path
	match, _ := filepath.Match("*.s3*.amazonaws.com", u.Host)
	switch {
	case match == true:
		hostSplits := strings.SplitN(u.Host, ".", 2)
		path = string(u.Separator) + hostSplits[0] + u.Path
	}
	pathSplits := strings.SplitN(path, "?", 2)
	splits := strings.SplitN(pathSplits[0], string(u.Separator), 3)
	switch len(splits) {
	case 0, 1:
		return false
	case 2:
		return false
	case 3:
		if splits[2] == "" {
			return false
		}
		return true
	}
	return false
}

func setSharePalette(style string) {
	console.SetCustomPalette(map[string]*color.Color{
		"Share":   color.New(color.FgGreen, color.Bold),
		"Expires": color.New(color.FgRed, color.Bold),
		"URL":     color.New(color.FgCyan, color.Bold),
		"File":    color.New(color.FgRed, color.Bold),
	})
	if style == "light" {
		console.SetCustomPalette(map[string]*color.Color{
			"Share":   color.New(color.FgWhite, color.Bold),
			"Expires": color.New(color.FgWhite, color.Bold),
			"URL":     color.New(color.FgWhite, color.Bold),
			"File":    color.New(color.FgWhite, color.Bold),
		})
		return
	}
	/// Add more styles here

	if style == "nocolor" {
		// All coloring options exhausted, setting nocolor safely
		console.SetNoColor()
	}
}
