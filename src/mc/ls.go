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

package mc

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/minio/mc/src/console"
	"github.com/minio/minio/pkg/probe"
)

// printDate - human friendly formatted date.
const (
	printDate = "2006-01-02 15:04:05 MST"
)

// contentMessage container for content message structure.
type contentMessage struct {
	Status   string    `json:"status"`
	Filetype string    `json:"type"`
	Time     time.Time `json:"lastModified"`
	Size     int64     `json:"size"`
	Key      string    `json:"key"`
}

// String colorized string message.
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

// JSON jsonified content message.
func (c contentMessage) JSON() string {
	c.Status = "success"
	jsonMessageBytes, e := json.Marshal(c)
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// parseContent parse client Content container into printer struct.
func parseContent(c *clientContent) contentMessage {
	content := contentMessage{}
	content.Time = c.Time.Local()

	// guess file type.
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
		// for windows make sure to print in 'windows' specific style.
		case runtime.GOOS == "windows":
			c.URL.Path = strings.Replace(c.URL.Path, "/", "\\", -1)
			c.URL.Path = strings.TrimSuffix(c.URL.Path, "\\")
		default:
			c.URL.Path = strings.TrimSuffix(c.URL.Path, "/")
		}
		if c.Type.IsDir() {
			switch {
			// for windows make sure to print in 'windows' specific style.
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

// doList - list all entities inside a folder.
func doList(clnt Client, isRecursive, isIncomplete bool) *probe.Error {
	prefixPath := clnt.GetURL().Path
	separator := string(clnt.GetURL().Separator)
	if !strings.HasSuffix(prefixPath, separator) {
		prefixPath = prefixPath[:strings.LastIndex(prefixPath, separator)+1]
	}
	for content := range clnt.List(isRecursive, isIncomplete) {
		if content.Err != nil {
			switch content.Err.ToGoError().(type) {
			// handle this specifically for filesystem related errors.
			case BrokenSymlink:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list broken link.")
				continue
			case TooManyLevelsSymlink:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list too many levels link.")
				continue
			case PathNotFound:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
				continue
			case PathInsufficientPermission:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
				continue
			case ObjectOnGlacier:
				errorIf(content.Err.Trace(clnt.GetURL().String()), "")
				continue
			}
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
			continue
		}
		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(content.URL.Path)
		prefixPath = filepath.ToSlash(prefixPath)

		// Trim prefix path from the content path.
		contentURL = strings.TrimPrefix(contentURL, prefixPath)
		content.URL.Path = contentURL
		parsedContent := parseContent(content)
		// Print colorized or jsonized content info.
		printMsg(parsedContent)
	}
	return nil
}
