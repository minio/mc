/*
 * MinIO Client (C) 2015 MinIO, Inc.
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

package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
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
	ETag     string    `json:"etag"`
	URL      string    `json:"url,omitempty"`

	VersionID      string `json:"versionId,omitempty"`
	IsDeleteMarker bool   `json:"isDeleteMarker,omitempty"`
}

// String colorized string message.
func (c contentMessage) String() string {
	message := console.Colorize("Time", fmt.Sprintf("[%s] ", c.Time.Format(printDate)))
	message += console.Colorize("Size", fmt.Sprintf("%7s ", strings.Join(strings.Fields(humanize.IBytes(uint64(c.Size))), "")))
	if c.Filetype == "folder" {
		return message + console.Colorize("Dir", c.Key)
	}
	if c.VersionID != "" {
		message += " [" + c.VersionID + "] "
	}
	if c.IsDeleteMarker {
		message += console.Colorize("File", fmt.Sprintf("\033[9m%s\033[0m", c.Key))
	} else {
		message += console.Colorize("File", c.Key)
	}
	return message
}

// JSON jsonified content message.
func (c contentMessage) JSON() string {
	c.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// parseContent parse client Content container into printer struct.
func parseContent(c *ClientContent) contentMessage {
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
	md5sum := strings.TrimPrefix(c.ETag, "\"")
	md5sum = strings.TrimSuffix(md5sum, "\"")
	content.ETag = md5sum
	// Convert OS Type to match console file printing style.
	content.Key = getKey(c)
	content.VersionID = c.VersionID
	content.IsDeleteMarker = c.IsDeleteMarker
	return content
}

// get content key
func getKey(c *ClientContent) string {
	sep := "/"

	// for windows make sure to print in 'windows' specific style.
	if runtime.GOOS == "windows" {
		c.URL.Path = strings.Replace(c.URL.Path, "/", "\\", -1)
		sep = "\\"
	}

	if c.Type.IsDir() && !strings.HasSuffix(c.URL.Path, sep) {
		return fmt.Sprintf("%s%s", c.URL.Path, sep)
	}
	return c.URL.Path
}

// doList - list all entities inside a folder.
func doList(ctx context.Context, clnt Client, isRecursive, isIncomplete bool, timeRef time.Time, withOlderVersions bool) error {
	prefixPath := clnt.GetURL().Path
	separator := string(clnt.GetURL().Separator)
	if !strings.HasSuffix(prefixPath, separator) {
		prefixPath = prefixPath[:strings.LastIndex(prefixPath, separator)+1]
	}

	var cErr error
	for content := range clnt.List(ctx, ListOptions{
		isRecursive:       isRecursive,
		isIncomplete:      isIncomplete,
		timeRef:           timeRef,
		withOlderVersions: withOlderVersions,
		withDeleteMarkers: true,
		showDir:           DirNone,
	}) {
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
			}
			errorIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
			cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}

		if content.StorageClass == s3StorageClassGlacier {
			continue
		}

		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(content.URL.Path)
		prefixPath = filepath.ToSlash(prefixPath)

		// Trim prefix of current working dir
		prefixPath = strings.TrimPrefix(prefixPath, "."+separator)
		// Trim prefix path from the content path.
		contentURL = strings.TrimPrefix(contentURL, prefixPath)
		content.URL.Path = contentURL
		parsedContent := parseContent(content)
		// URL is empty by default
		// Set it to either relative dir (host) or public url (remote)
		parsedContent.URL = clnt.GetURL().String()
		// Print colorized or jsonized content info.
		printMsg(parsedContent)
	}
	return cErr
}
