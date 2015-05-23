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
	"runtime"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

/// ls - related internal functions

// iso8601 date
const (
	printDate = "2006-01-02 15:04:05 MST"
)

// printContent prints content meta-data
func printContent(c *client.Content) {
	content := console.Content{}
	content.Filetype = func() string {
		if c.Type.IsDir() {
			return "inode/directory"
		}
		if c.Type.IsRegular() {
			return "application/octet-stream"
		}
		return "application/octet-stream"
	}()
	content.Size = humanize.IBytes(uint64(c.Size))
	if runtime.GOOS == "windows" {
		content.Name = strings.Replace(c.Name, "/", "\\", -1)
		content.Name = strings.TrimSuffix(content.Name, "\\")
	}
	content.Name = strings.TrimSuffix(content.Name, "/")
	content.Time = c.Time.Local().Format(printDate)
	console.ContentInfo(content)
}

// doList - list all entities inside a folder
func doList(clnt client.Client, targetURL string, recursive bool) error {
	var err error
	for contentCh := range clnt.List(recursive) {
		if contentCh.Err != nil {
			err = contentCh.Err
			break
		}
		contentName := contentCh.Content.Name
		if recursive {
			// this special handling is necessary since we are sending back absolute paths with in ListRecursive()
			// a user would not wish to see a URL just for recursive and not for regular List()
			//
			// To be consistent we have to filter them out
			contentName = strings.TrimPrefix(contentName, strings.TrimSuffix(targetURL, "/")+"/")
		}
		contentCh.Content.Name = contentName
		printContent(contentCh.Content)
	}
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	return nil
}
