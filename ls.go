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
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

/// ls - related internal functions
const (
	printDate = "2006-01-02 15:04:05 MST"
)

// contentMessage container for content message structure.
type contentMessage struct {
	Filetype string    `json:"type"`
	Time     time.Time `json:"lastModified"`
	Size     int64     `json:"size"`
	Key      string    `json:"key"`
}

// String colorized string message
func (c contentMessage) String() string {
	message := console.Colorize("Time", fmt.Sprintf("[%s] ", c.Time.Format(printDate)))
	message = message + console.Colorize("Size", fmt.Sprintf("%6s ", humanize.IBytes(uint64(c.Size))))
	message = func() string {
		if c.Filetype == "folder" {
			return message + console.Colorize("Dir", fmt.Sprintf("%s", c.Key))
		}
		return message + console.Colorize("File", fmt.Sprintf("%s", c.Key))
	}()
	return message
}

// JSON jsonified content message
func (c contentMessage) JSON() string {
	jsonMessageBytes, e := json.Marshal(c)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// parseContent parse client Content container into printer struct.
func parseContent(c *client.Content) contentMessage {
	content := contentMessage{}
	content.Time = c.Time.Local()

	// guess file type
	content.Filetype = func() string {
		if c.Type.IsDir() {
			return "folder"
		}
		return "file"
	}()

	content.Size = c.Size
	// Convert OS Type to match console file printing style.
	content.Key = func() string {
		switch {
		case runtime.GOOS == "windows":
			c.URL.Path = strings.Replace(c.URL.Path, "/", "\\", -1)
			c.URL.Path = strings.TrimSuffix(c.URL.Path, "\\")
		default:
			c.URL.Path = strings.TrimSuffix(c.URL.Path, "/")
		}
		if c.Type.IsDir() {
			switch {
			case runtime.GOOS == "windows":
				return fmt.Sprintf("%s\\", c.URL.Path)
			default:
				return fmt.Sprintf("%s/", c.URL.Path)
			}
		}
		return c.URL.Path
	}()
	return content
}

// trimContent to fancify the output for directories
func trimContent(parentContent, childContent *client.Content, recursive bool) *client.Content {
	if recursive {
		// If recursive remove the unnecessary parentContent prefix. '/', in the beginning
		trimmedContent := new(client.Content)
		trimmedContent = childContent
		if strings.HasSuffix(parentContent.URL.Path, string(parentContent.URL.Separator)) {
			trimmedContent.URL.Path = strings.TrimPrefix(trimmedContent.URL.Path, parentContent.URL.Path)
		}
		if strings.Index(trimmedContent.URL.Path, string(trimmedContent.URL.Separator)) == 0 {
			if len(trimmedContent.URL.Path) > 0 {
				trimmedContent.URL.Path = trimmedContent.URL.Path[1:]
			}
		}
		return trimmedContent
	}
	// If parentContent is a directory, use it to trim the sub-folders
	if parentContent.Type.IsDir() {
		// Allocate a new client.Content for trimmed output
		trimmedContent := new(client.Content)
		trimmedContent = childContent
		if parentContent.URL.Path == string(parentContent.URL.Separator) {
			trimmedContent.URL.Path = strings.TrimPrefix(trimmedContent.URL.Path, parentContent.URL.Path)
			return trimmedContent
		}
		// If the beginning of the trimPrefix is a URL.Separator ignore it.
		trimmedContent.URL.Path = strings.TrimPrefix(trimmedContent.URL.Path, string(trimmedContent.URL.Separator))
		trimPrefixContentPath := parentContent.URL.Path[:strings.LastIndex(parentContent.URL.Path,
			string(parentContent.URL.Separator))+1]
		// If the beginning of the trimPrefix is a URL.Separator ignore it.
		trimPrefixContentPath = strings.TrimPrefix(trimPrefixContentPath, string(parentContent.URL.Separator))
		trimmedContent.URL.Path = strings.TrimPrefix(trimmedContent.URL.Path, trimPrefixContentPath)
		return trimmedContent
	}
	// if the target is a file, no more trimming needed return back as is.
	return childContent
}

// doList - list all entities inside a folder.
func doList(clnt client.Client, isRecursive, isIncomplete bool) *probe.Error {
	_, parentContent, err := url2Stat(clnt.GetURL().String())
	if err != nil {
		return err.Trace(clnt.GetURL().String())
	}

	for contentCh := range clnt.List(isRecursive, isIncomplete) {
		if contentCh.Err != nil {
			switch contentCh.Err.ToGoError().(type) {
			// handle this specifically for filesystem
			case client.BrokenSymlink:
				errorIf(contentCh.Err.Trace(), "Unable to list broken link.")
				continue
			case client.TooManyLevelsSymlink:
				errorIf(contentCh.Err.Trace(), "Unable to list too many levels link.")
				continue
			}
			if os.IsNotExist(contentCh.Err.ToGoError()) || os.IsPermission(contentCh.Err.ToGoError()) {
				if contentCh.Content != nil {
					if contentCh.Content.Type.IsDir() {
						if contentCh.Content.Type&os.ModeSymlink == os.ModeSymlink {
							errorIf(contentCh.Err.Trace(), "Unable to list broken folder link.")
							continue
						}
						errorIf(contentCh.Err.Trace(), "Unable to list folder.")
					}
				} else {
					errorIf(contentCh.Err.Trace(), "Unable to list.")
					continue
				}
			}
			err = contentCh.Err.Trace()
			break
		}
		trimmedContent := trimContent(parentContent, contentCh.Content, isRecursive)
		parsedContent := parseContent(trimmedContent)
		printMsg(parsedContent)
	}
	if err != nil {
		return err.Trace()
	}
	return nil
}
