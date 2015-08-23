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
	"github.com/minio/minio/pkg/probe"
)

/// ls - related internal functions

const (
	printDate = "2006-01-02 15:04:05 MST"
)

// parseContent parse client Content container into printer struct
func parseContent(c *client.Content) ContentMessage {
	content := ContentMessage{}
	content.Time = c.Time.Local().Format(printDate)

	// guess file type
	content.Filetype = func() string {
		if c.Type.IsDir() {
			return "folder"
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
func doList(clnt client.Client, recursive bool) *probe.Error {
	var err *probe.Error
	for contentCh := range clnt.List(recursive) {
		if contentCh.Err != nil {
			switch contentCh.Err.ToGoError().(type) {
			// handle this specifically for filesystem
			case client.ISBrokenSymlink:
				errorIf(contentCh.Err.Trace(), "Unable to list broken link.")
				continue
			}
			if os.IsNotExist(contentCh.Err.ToGoError()) || os.IsPermission(contentCh.Err.ToGoError()) {
				errorIf(contentCh.Err.Trace(), "Unable to list.")
				continue
			}
			err = contentCh.Err.Trace()
			break
		}
		console.Print(parseContent(contentCh.Content))
	}
	if err != nil {
		return err.Trace()
	}
	return nil
}
