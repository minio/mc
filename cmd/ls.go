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
	"sort"
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
	VersionOrd     int    `json:"versionOrdinal,omitempty"`
	VersionIndex   int    `json:"versionIndex,omitempty"`
	IsDeleteMarker bool   `json:"isDeleteMarker,omitempty"`
}

// String colorized string message.
func (c contentMessage) String() string {
	message := console.Colorize("Time", fmt.Sprintf("[%s]", c.Time.Format(printDate)))
	message += console.Colorize("Size", fmt.Sprintf("%7s", strings.Join(strings.Fields(humanize.IBytes(uint64(c.Size))), "")))
	fileDesc := ""

	if c.VersionID != "" {
		fileDesc += console.Colorize("VersionID", " "+c.VersionID) + console.Colorize("VersionOrd", fmt.Sprintf(" v%d", c.VersionOrd))
		if c.IsDeleteMarker {
			fileDesc += console.Colorize("DEL", " DEL")
		} else {
			fileDesc += console.Colorize("PUT", " PUT")
		}
	}

	fileDesc += " " + c.Key

	if c.Filetype == "folder" {
		message += console.Colorize("Dir", fileDesc)
	} else {
		message += console.Colorize("File", fileDesc)
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

// Use OS separator and adds a trailing separator if it is a dir
func getOSDependantKey(path string, isDir bool) string {
	sep := "/"

	// for windows make sure to print in 'windows' specific style.
	if runtime.GOOS == "windows" {
		path = strings.Replace(path, "/", "\\", -1)
		sep = "\\"
	}

	if isDir && !strings.HasSuffix(path, sep) {
		return fmt.Sprintf("%s%s", path, sep)
	}
	return path
}

// get content key
func getKey(c *ClientContent) string {
	return getOSDependantKey(c.URL.Path, c.Type.IsDir())
}

// Generate printable listing from a list of sorted client
// contents, the latest created content comes first.
func generateContentMessages(clntURL ClientURL, ctnts []*ClientContent, printAllVersions bool) (msgs []contentMessage) {
	prefixPath := clntURL.Path
	prefixPath = filepath.ToSlash(prefixPath)
	if !strings.HasSuffix(prefixPath, "/") {
		prefixPath = prefixPath[:strings.LastIndex(prefixPath, "/")+1]
	}
	prefixPath = strings.TrimPrefix(prefixPath, "./")

	nrVersions := len(ctnts)

	for i, c := range ctnts {
		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(c.URL.Path)
		// Trim prefix path from the content path.
		c.URL.Path = strings.TrimPrefix(contentURL, prefixPath)

		contentMsg := contentMessage{}
		contentMsg.Time = c.Time.Local()

		// guess file type.
		contentMsg.Filetype = func() string {
			if c.Type.IsDir() {
				return "folder"
			}
			return "file"
		}()

		contentMsg.Size = c.Size
		md5sum := strings.TrimPrefix(c.ETag, "\"")
		md5sum = strings.TrimSuffix(md5sum, "\"")
		contentMsg.ETag = md5sum
		// Convert OS Type to match console file printing style.
		contentMsg.Key = getKey(c)
		contentMsg.VersionID = c.VersionID
		contentMsg.IsDeleteMarker = c.IsDeleteMarker
		contentMsg.VersionOrd = nrVersions - i
		// URL is empty by default
		// Set it to either relative dir (host) or public url (remote)
		contentMsg.URL = clntURL.String()

		msgs = append(msgs, contentMsg)

		if !printAllVersions {
			break
		}
	}
	return
}

func sortObjectVersions(ctntVersions []*ClientContent) {
	// Sort versions
	sort.Slice(ctntVersions, func(i, j int) bool {
		if ctntVersions[i].IsLatest {
			return true
		}
		if ctntVersions[j].IsLatest {
			return false
		}
		return ctntVersions[i].Time.After(ctntVersions[j].Time)
	})
}

// Pretty print the list of versions belonging to one object
func printObjectVersions(clntURL ClientURL, ctntVersions []*ClientContent, printAllVersions bool) {
	sortObjectVersions(ctntVersions)
	msgs := generateContentMessages(clntURL, ctntVersions, printAllVersions)
	for _, msg := range msgs {
		printMsg(msg)
	}
}

// doList - list all entities inside a folder.
func doList(ctx context.Context, clnt Client, isRecursive, isIncomplete bool, timeRef time.Time, withOlderVersions bool) error {

	var (
		lastPath          string
		perObjectVersions []*ClientContent
		cErr              error
	)

	for content := range clnt.List(ctx, ListOptions{
		IsRecursive:       isRecursive,
		IsIncomplete:      isIncomplete,
		TimeRef:           timeRef,
		WithOlderVersions: withOlderVersions || !timeRef.IsZero(),
		WithDeleteMarkers: true,
		ShowDir:           DirNone,
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

		if lastPath != content.URL.Path {
			// Print any object in the current list before reinitializing it
			printObjectVersions(clnt.GetURL(), perObjectVersions, withOlderVersions)
			lastPath = content.URL.Path
			perObjectVersions = []*ClientContent{}
		}

		perObjectVersions = append(perObjectVersions, content)
	}

	printObjectVersions(clnt.GetURL(), perObjectVersions, withOlderVersions)
	return cErr
}
