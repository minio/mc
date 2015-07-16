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
	"os"
	"runtime"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
)

/// ls - related internal functions

const (
	printDate = "2006-01-02 15:04:05 MST"
)

// parseContent parse client Content container into printer struct
func parseContent(c *client.Content) Content {
	content := Content{}
	content.Time = c.Time.Local().Format(printDate)

	// guess file type
	content.Filetype = func() string {
		if c.Type.IsDir() {
			return "directory"
		}
		return "file"
	}()

	content.Size = humanize.IBytes(uint64(c.Size))

	// Convert OS Type to match console file printing style
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
func doList(clnt client.Client, recursive bool) error {
	var err error
	for contentCh := range clnt.List(recursive) {
		if contentCh.Err != nil {
			switch err := ToError(contentCh.Err).(type) {
			// handle this specifically for filesystem
			case client.ISBrokenSymlink:
				console.Errorf(err.Error())
				continue
			}
			if os.IsNotExist(ToError(contentCh.Err)) || os.IsPermission(ToError(contentCh.Err)) {
				console.Errorf(contentCh.Err.Error())
				continue
			}
			err = contentCh.Err
			break
		}
		console.Prints(parseContent(contentCh.Content))
	}
	if err != nil {
		return NewIodine(err, map[string]string{"Target": clnt.URL().String()})
	}
	return nil
}
