/*
 * MinIO Client (C) 2017-2019 MinIO, Inc.
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
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

// contentMessage container for content message structure.
type statMessage struct {
	Status           string            `json:"status"`
	Key              string            `json:"name"`
	Date             time.Time         `json:"lastModified"`
	Size             int64             `json:"size"`
	ETag             string            `json:"etag"`
	Type             string            `json:"type"`
	Expires          time.Time         `json:"expires"`
	Expiration       time.Time         `json:"expiration"`
	ExpirationRuleID string            `json:"expirationRuleID"`
	Metadata         map[string]string `json:"metadata"`
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
	if !stat.Expiration.IsZero() {
		console.Println(fmt.Sprintf("%-10s: %s (lifecycle-rule-id: %s) ", "Expiration", stat.Expiration.Local().Format(printDate), stat.ExpirationRuleID))
	}
	var maxKey = 0
	for k := range stat.Metadata {
		// Skip encryption headers, we print them later.
		if !strings.HasPrefix(strings.ToLower(k), serverEncryptionKeyPrefix) {
			if len(k) > maxKey {
				maxKey = len(k)
			}
		}
	}
	if maxKey > 0 {
		console.Println(fmt.Sprintf("%-10s:", "Metadata"))
		for k, v := range stat.Metadata {
			// Skip encryption headers, we print them later.
			if !strings.HasPrefix(strings.ToLower(k), serverEncryptionKeyPrefix) {
				console.Println(fmt.Sprintf("  %-*.*s: %s ", maxKey, maxKey, k, v))
			}
		}
	}

	maxKey = 0
	for k := range stat.Metadata {
		if strings.HasPrefix(strings.ToLower(k), serverEncryptionKeyPrefix) {
			if len(k) > maxKey {
				maxKey = len(k)
			}
		}
	}
	if maxKey > 0 {
		console.Println(fmt.Sprintf("%-10s:", "Encrypted"))
		for k, v := range stat.Metadata {
			if strings.HasPrefix(strings.ToLower(k), serverEncryptionKeyPrefix) {
				console.Println(fmt.Sprintf("  %-*.*s: %s ", maxKey, maxKey, k, v))
			}
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
func parseStat(c *ClientContent) statMessage {
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
	content.Expiration = c.Expiration
	content.ExpirationRuleID = c.ExpirationRuleID
	return content
}

// Return standardized URL to be used to compare later.
func getStandardizedURL(targetURL string) string {
	return filepath.FromSlash(targetURL)
}

// statURL - uses combination of GET listing and HEAD to fetch information of one or more objects
// HEAD can fail with 400 with an SSE-C encrypted object but we still return information gathered
// from GET listing.
func statURL(ctx context.Context, targetURL, versionID string, isIncomplete, isRecursive bool, encKeyDB map[string][]prefixSSEPair) ([]*ClientContent, *probe.Error) {
	var stats []*ClientContent
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
	lstOptions := ListOptions{isRecursive: isRecursive, isIncomplete: isIncomplete, showDir: DirNone}
	if versionID != "" {
		lstOptions.withOlderVersions = true
		lstOptions.withDeleteMarkers = true
	}
	var cErr error
	for content := range clnt.List(ctx, lstOptions) {
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

		url := targetAlias + getKey(content)
		standardizedURL := getStandardizedURL(targetURL)

		if !isRecursive && !strings.HasPrefix(url, standardizedURL) {
			return nil, errTargetNotFound(targetURL).Trace(url, standardizedURL)
		}

		if versionID != "" {
			if versionID != content.VersionID {
				continue
			}
		}

		_, stat, err := url2Stat(ctx, url, versionID, true, encKeyDB, time.Time{})
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
