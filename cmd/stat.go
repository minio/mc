/*
 * MinIO Client (C) 2017 MinIO, Inc.
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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// contentMessage container for content message structure.
type statMessage struct {
	Status            string            `json:"status"`
	Key               string            `json:"name"`
	Date              time.Time         `json:"lastModified"`
	Size              int64             `json:"size"`
	ETag              string            `json:"etag"`
	Type              string            `json:"type"`
	Expires           time.Time         `json:"expires"`
	EncryptionHeaders map[string]string `json:"encryption,omitempty"`
	Metadata          map[string]string `json:"metadata"`
}

// String colorized string message.
func printStat(stat statMessage) {
	// Format properly for alignment based on maxKey length
	stat.Key = fmt.Sprintf("%-10s: %s", "Name", stat.Key)
	console.Println(console.Colorize("Name", stat.Key))
	console.Println(fmt.Sprintf("%-10s: %s ", "Date", stat.Date.Format(printDate)))
	console.Println(fmt.Sprintf("%-10s: %-6s ", "Size", humanize.IBytes(uint64(stat.Size))))
	if stat.ETag != "" {
		console.Println(fmt.Sprintf("%-10s: %s ", "ETag", stat.ETag))
	}
	console.Println(fmt.Sprintf("%-10s: %s ", "Type", stat.Type))
	if !stat.Expires.IsZero() {
		console.Println(fmt.Sprintf("%-10s: %s ", "Expires", stat.Expires.Format(printDate)))
	}
	var maxKey = 0
	for k := range stat.Metadata {
		if len(k) > maxKey {
			maxKey = len(k)
		}
	}
	if len(stat.Metadata) > 0 {
		console.Println(fmt.Sprintf("%-10s:", "Metadata"))
		for k, v := range stat.Metadata {
			console.Println(fmt.Sprintf("  %-*.*s: %s ", maxKey, maxKey, k, v))
		}
	}
	maxKey = 0
	for k := range stat.EncryptionHeaders {
		if len(k) > maxKey {
			maxKey = len(k)
		}
	}
	if len(stat.EncryptionHeaders) > 0 {
		console.Println(fmt.Sprintf("%-10s:", "Encrypted"))
		for k, v := range stat.EncryptionHeaders {
			console.Println(fmt.Sprintf("  %-*.*s: %s ", maxKey, maxKey, k, v))
		}
	}
	console.Println()
}

// JSON jsonified content message.
func (c statMessage) JSON() string {
	c.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// parseStat parses client Content container into statMessage struct.
func parseStat(c *clientContent) statMessage {
	content := statMessage{}
	content.Date = c.Time.Local()
	// guess file type.
	content.Type = func() string {
		if c.Type.IsDir() {
			return "folder"
		}
		return "file"
	}()
	content.Size = c.Size
	content.Key = getKey(c)
	content.Metadata = c.Metadata
	content.ETag = strings.TrimPrefix(c.ETag, "\"")
	content.ETag = strings.TrimSuffix(content.ETag, "\"")
	content.Expires = c.Expires
	content.EncryptionHeaders = c.EncryptionHeaders
	return content
}

// statURL - simple or recursive listing
func statURL(targetURL string, isIncomplete, isRecursive bool, encKeyDB map[string][]prefixSSEPair) ([]*clientContent, *probe.Error) {
	var stats []*clientContent
	var clnt Client
	clnt, err := newClient(targetURL)
	if err != nil {
		return nil, err
	}

	targetAlias, _, _ := mustExpandAlias(targetURL)

	prefixPath := clnt.GetURL().Path
	separator := string(clnt.GetURL().Separator)
	if !strings.HasSuffix(prefixPath, separator) {
		prefixPath = prefixPath[:strings.LastIndex(prefixPath, separator)+1]
	}
	var cErr error
	for content := range clnt.List(isRecursive, isIncomplete, DirNone) {
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
			cErr = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}
		url := targetAlias + getKey(content)

		if !isRecursive && !strings.HasPrefix(url, targetURL) {
			return nil, errTargetNotFound(targetURL)
		}

		_, stat, err := url2Stat(url, true, encKeyDB)
		if err != nil {
			stat = content
		}
		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(stat.URL.Path)
		prefixPath = filepath.ToSlash(prefixPath)
		// Trim prefix path from the content path.
		contentURL = strings.TrimPrefix(contentURL, prefixPath)
		stat.URL.Path = contentURL
		stats = append(stats, stat)
	}

	return stats, probe.NewError(cErr)
}
