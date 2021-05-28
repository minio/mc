// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/notification"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/console"
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
	Expiration        time.Time         `json:"expiration"`
	ExpirationRuleID  string            `json:"expirationRuleID"`
	ReplicationStatus string            `json:"replicationStatus"`
	Metadata          map[string]string `json:"metadata"`
	VersionID         string            `json:"versionID,omitempty"`
	DeleteMarker      bool              `json:"deleteMarker,omitempty"`
	singleObject      bool
}

func (stat statMessage) String() (msg string) {
	var msgBuilder strings.Builder
	// Format properly for alignment based on maxKey leng
	stat.Key = fmt.Sprintf("%-10s: %s", "Name", stat.Key)
	msgBuilder.WriteString(console.Colorize("Name", stat.Key) + "\n")
	msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "Date", stat.Date.Format(printDate)) + "\n")
	msgBuilder.WriteString(fmt.Sprintf("%-10s: %-6s ", "Size", humanize.IBytes(uint64(stat.Size))) + "\n")
	if stat.ETag != "" {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "ETag", stat.ETag) + "\n")
	}
	if stat.VersionID != "" {
		versionIDField := stat.VersionID
		if stat.DeleteMarker {
			versionIDField += " (delete-marker)"
		}
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "VersionID", versionIDField) + "\n")
	}
	msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "Type", stat.Type) + "\n")
	if !stat.Expires.IsZero() {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "Expires", stat.Expires.Format(printDate)) + "\n")
	}
	if !stat.Expiration.IsZero() {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s (lifecycle-rule-id: %s) ", "Expiration",
			stat.Expiration.Local().Format(printDate), stat.ExpirationRuleID) + "\n")
	}
	var maxKeyMetadata = 0
	var maxKeyEncrypted = 0
	for k := range stat.Metadata {
		// Skip encryption headers, we print them later.
		if !strings.HasPrefix(strings.ToLower(k), serverEncryptionKeyPrefix) {
			if len(k) > maxKeyMetadata {
				maxKeyMetadata = len(k)
			}
		} else if strings.HasPrefix(strings.ToLower(k), serverEncryptionKeyPrefix) {
			if len(k) > maxKeyEncrypted {
				maxKeyEncrypted = len(k)
			}
		}
	}
	if maxKeyMetadata > 0 {
		msgBuilder.WriteString(fmt.Sprintf("%-10s:", "Metadata") + "\n")
		for k, v := range stat.Metadata {
			// Skip encryption headers, we print them later.
			if !strings.HasPrefix(strings.ToLower(k), serverEncryptionKeyPrefix) {
				msgBuilder.WriteString(fmt.Sprintf("  %-*.*s: %s ", maxKeyMetadata, maxKeyMetadata, k, v) + "\n")
			}
		}
	}

	if maxKeyEncrypted > 0 {
		msgBuilder.WriteString(fmt.Sprintf("%-10s:", "Encrypted") + "\n")
		for k, v := range stat.Metadata {
			if strings.HasPrefix(strings.ToLower(k), serverEncryptionKeyPrefix) {
				msgBuilder.WriteString(fmt.Sprintf("  %-*.*s: %s ", maxKeyEncrypted, maxKeyEncrypted, k, v) + "\n")
			}
		}
	}
	if stat.ReplicationStatus != "" {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "Replication Status", stat.ReplicationStatus))
	}

	return msgBuilder.String()
}

// JSON jsonified content message.
func (stat statMessage) JSON() string {
	stat.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(stat, "", " ")
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
	content.VersionID = c.VersionID
	content.Key = getKey(c)
	content.Metadata = c.Metadata
	content.ETag = strings.TrimPrefix(c.ETag, "\"")
	content.ETag = strings.TrimSuffix(content.ETag, "\"")
	content.Expires = c.Expires
	content.Expiration = c.Expiration
	content.ExpirationRuleID = c.ExpirationRuleID
	content.ReplicationStatus = c.ReplicationStatus
	return content
}

// Return standardized URL to be used to compare later.
func getStandardizedURL(targetURL string) string {
	return filepath.FromSlash(targetURL)
}

// statURL - uses combination of GET listing and HEAD to fetch information of one or more objects
// HEAD can fail with 400 with an SSE-C encrypted object but we still return information gathered
// from GET listing.
func statURL(ctx context.Context, targetURL, versionID string, timeRef time.Time, includeOlderVersions, isIncomplete, isRecursive bool, encKeyDB map[string][]prefixSSEPair) ([]*ClientContent, []*BucketInfo, *probe.Error) {
	var stats []*ClientContent
	var bucketStats []*BucketInfo
	var clnt Client
	clnt, err := newClient(targetURL)
	if err != nil {
		return nil, nil, err
	}

	targetAlias, _, _ := mustExpandAlias(targetURL)

	prefixPath := clnt.GetURL().Path
	separator := string(clnt.GetURL().Separator)
	if !strings.HasSuffix(prefixPath, separator) {
		prefixPath = prefixPath[:strings.LastIndex(prefixPath, separator)+1]
	}
	lstOptions := ListOptions{Recursive: isRecursive, Incomplete: isIncomplete, ShowDir: DirNone}
	switch {
	case versionID != "":
		lstOptions.WithOlderVersions = true
		lstOptions.WithDeleteMarkers = true
	case !timeRef.IsZero(), includeOlderVersions:
		lstOptions.WithOlderVersions = includeOlderVersions
		lstOptions.WithDeleteMarkers = true
		lstOptions.TimeRef = timeRef
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
			return nil, nil, errTargetNotFound(targetURL).Trace(url, standardizedURL)
		}

		if versionID != "" {
			if versionID != content.VersionID {
				continue
			}
		}
		clnt, stat, err := url2Stat(ctx, url, content.VersionID, true, encKeyDB, timeRef)
		if err != nil {
			continue
		}
		// if stat is on a bucket and non-recursive mode, serve the bucket metadata
		if clnt != nil && !isRecursive && stat.Type.IsDir() {
			bstat, err := clnt.GetBucketInfo(ctx)
			if err == nil {
				// Convert any os specific delimiters to "/".
				contentURL := filepath.ToSlash(bstat.URL.Path)
				prefixPath = filepath.ToSlash(prefixPath)
				// Trim prefix path from the content path.
				contentURL = strings.TrimPrefix(contentURL, prefixPath)
				bstat.URL.Path = contentURL
				bucketStats = append(bucketStats, &bstat)
				continue
			}
		}

		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(stat.URL.Path)
		prefixPath = filepath.ToSlash(prefixPath)
		// Trim prefix path from the content path.
		contentURL = strings.TrimPrefix(contentURL, prefixPath)
		stat.URL.Path = contentURL
		stats = append(stats, stat)
	}

	return stats, bucketStats, probe.NewError(cErr)
}

// BucketInfo holds info about a bucket
type BucketInfo struct {
	URL        ClientURL   `json:"-"`
	Key        string      `json:"name"`
	Date       time.Time   `json:"lastModified"`
	Size       int64       `json:"size"`
	Type       os.FileMode `json:"-"`
	Versioning struct {
		Status    string `json:"status"`
		MFADelete string `json:"MFADelete"`
	} `json:"Versioning,omitempty"`
	Encryption struct {
		Algorithm string `json:"algorithm,omitempty"`
		KeyID     string `json:"keyId,omitempty"`
	} `json:"Encryption,omitempty"`
	Locking struct {
		Enabled  string              `json:"enabled"`
		Mode     minio.RetentionMode `json:"mode"`
		Validity string              `json:"validity"`
	} `json:"ObjectLock,omitempty"`
	Replication struct {
		Enabled bool               `json:"enabled"`
		Config  replication.Config `json:"config,omitempty"`
	} `json:"Replication"`
	Policy struct {
		Type string `json:"type"`
		Text string `json:"policy,omitempty"`
	} `json:"Policy,omitempty"`
	Location string            `json:"location"`
	Tagging  map[string]string `json:"tagging,omitempty"`
	ILM      struct {
		Config *lifecycle.Configuration `json:"config,omitempty"`
	} `json:"ilm,omitempty"`
	Notification struct {
		Config notification.Configuration `json:"config,omitempty"`
	} `json:"notification,omitempty"`
}

// Tags returns stringified tag list.
func (i BucketInfo) Tags() string {
	keys := []string{}
	for key := range i.Tagging {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	strs := []string{}
	for _, key := range keys {
		strs = append(
			strs,
			fmt.Sprintf("%v:%v", console.Colorize("Key", key), console.Colorize("Value", i.Tagging[key])),
		)
	}

	return strings.Join(strs, ", ")
}

type bucketInfoMessage struct {
	Op       string
	URL      string     `json:"url"`
	Status   string     `json:"status"`
	Metadata BucketInfo `json:"metadata"`
}

func (v bucketInfoMessage) JSON() string {
	v.Status = "success"
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", " ")
	// Disable escaping special chars to display XML tags correctly
	enc.SetEscapeHTML(false)

	fatalIf(probe.NewError(enc.Encode(v)), "Unable to marshal into JSON.")
	return buf.String()

}

func (v bucketInfoMessage) String() string {
	var b strings.Builder
	info := v.Metadata

	keyStr := getKey(&ClientContent{URL: v.Metadata.URL, Type: v.Metadata.Type})
	key := fmt.Sprintf("%-10s: %s", "Name", keyStr)
	fmt.Fprintln(&b, console.Colorize("Name", key))
	fmt.Fprintf(&b, fmt.Sprintf("%-10s: %-6s \n", "Size", humanize.IBytes(uint64(v.Metadata.Size))))
	fType := func() string {
		if v.Metadata.Type.IsDir() {
			return "folder"
		}
		return "file"
	}()
	fmt.Fprintf(&b, fmt.Sprintf("%-10s: %s \n", "Type", fType))
	fmt.Fprintf(&b, fmt.Sprintf("%-10s:\n", "Metadata"))
	placeHolder := ""
	if info.Encryption.Algorithm != "" {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "Encryption: ")
		fmt.Fprintf(&b, console.Colorize("Key", "\n\tAlgorithm: "))
		fmt.Fprintf(&b, console.Colorize("Value", info.Encryption.Algorithm))
		fmt.Fprintf(&b, console.Colorize("Key", "\n\tKey ID: "))
		fmt.Fprintf(&b, console.Colorize("Value", info.Encryption.KeyID))
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%2s%s", placeHolder, "Versioning: ")
	if info.Versioning.Status == "" {
		fmt.Fprintf(&b, console.Colorize("Unset", "Un-versioned"))
	} else {
		fmt.Fprintf(&b, console.Colorize("Set", info.Versioning.Status))
	}
	fmt.Fprintln(&b)

	if info.Locking.Mode != "" {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "LockConfiguration: ")
		fmt.Fprintf(&b, "%4s%s", placeHolder, "RetentionMode: ")
		fmt.Fprintf(&b, console.Colorize("Value", info.Locking.Mode))
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "%4s%s", placeHolder, "Retention Until Date: ")
		fmt.Fprintf(&b, console.Colorize("Value", info.Locking.Validity))
		fmt.Fprintln(&b)
	}
	if len(info.Notification.Config.TopicConfigs) > 0 {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "Notification: ")
		fmt.Fprintf(&b, console.Colorize("Set", "Set"))
		fmt.Fprintln(&b)
	}
	if info.Replication.Enabled {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "Replication: ")
		fmt.Fprintf(&b, console.Colorize("Set", "Enabled"))
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%2s%s", placeHolder, "Location: ")
	fmt.Fprintf(&b, console.Colorize("Generic", info.Location))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%2s%s", placeHolder, "Policy: ")
	if info.Policy.Type == "none" {
		fmt.Fprintf(&b, console.Colorize("UnSet", info.Policy.Type))
	} else {
		fmt.Fprintf(&b, console.Colorize("Set", info.Policy.Type))
	}
	fmt.Fprintln(&b)
	if info.Tags() != "" {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "Tagging: ")
		fmt.Fprintf(&b, console.Colorize("Generic", info.Tags()))
		fmt.Fprintln(&b)
	}
	if info.ILM.Config != nil {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "ILM: ")
		fmt.Fprintf(&b, console.Colorize("Set", "Set"))
		fmt.Fprintln(&b)
	}

	return b.String()
}
