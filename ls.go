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
	"fmt"
	"path/filepath"
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

// parseContent parse client Content container into printer struct
func parseContent(c *client.Content) Content {
	content := Content{}
	content.Time = c.Time.Local().Format(printDate)
	content.Filetype = func() string {
		if c.Type.IsDir() {
			return "inode/directory"
		}
		return "application/octet-stream"
	}()
	content.Size = humanize.IBytes(uint64(c.Size))
	content.Name = func() string {
		switch {
		case runtime.GOOS == "windows":
			c.Name = strings.Replace(c.Name, "/", "\\", -1)
			c.Name = strings.TrimSuffix(c.Name, "\\")
		default:
			c.Name = strings.TrimSuffix(c.Name, "/")
		}
		if c.Type.IsDir() {
			switch {
			case runtime.GOOS == "windows":
				return fmt.Sprintf("%s\\", c.Name)
			default:
				return fmt.Sprintf("%s/", c.Name)
			}
		}
		return c.Name
	}()
	return content
}

// doList - list all entities inside a folder
func doList(clnt client.Client, targetURL string, recursive bool, recursivePrefix string) error {
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
			contentName = strings.TrimPrefix(contentName,
				strings.TrimSuffix(targetURL, string(filepath.Separator))+string(filepath.Separator))
			if recursivePrefix != "" {
				contentName = recursivePrefix + string(filepath.Separator) + contentName
			}
		}
		contentCh.Content.Name = contentName
		console.Println(parseContent(contentCh.Content))
	}
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	return nil
}
