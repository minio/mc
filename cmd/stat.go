// Copyright (c) 2015-2024 MinIO, Inc.
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

	"github.com/dustin/go-humanize"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/notification"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/v3/console"
)

// contentMessage container for content message structure.
type statMessage struct {
	Status            string             `json:"status"`
	Key               string             `json:"name"`
	Date              time.Time          `json:"lastModified"`
	Size              int64              `json:"size"`
	ETag              string             `json:"etag"`
	Type              string             `json:"type,omitempty"`
	Expires           *time.Time         `json:"expires,omitempty"`
	Expiration        *time.Time         `json:"expiration,omitempty"`
	ExpirationRuleID  string             `json:"expirationRuleID,omitempty"`
	ReplicationStatus string             `json:"replicationStatus,omitempty"`
	Metadata          map[string]string  `json:"metadata,omitempty"`
	VersionID         string             `json:"versionID,omitempty"`
	DeleteMarker      bool               `json:"deleteMarker,omitempty"`
	Restore           *minio.RestoreInfo `json:"restore,omitempty"`
	Checksum          map[string]string  `json:"checksum,omitempty"`
}

func (stat statMessage) String() (msg string) {
	var msgBuilder strings.Builder
	// Format properly for alignment based on maxKey leng
	stat.Key = fmt.Sprintf("%-10s: %s", "Name", stat.Key)
	msgBuilder.WriteString(console.Colorize("Name", stat.Key) + "\n")
	if !stat.Date.IsZero() && !stat.Date.Equal(timeSentinel) {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "Date", stat.Date.Format(printDate)) + "\n")
	}
	if stat.Type != "folder" {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %-6s ", "Size", humanize.IBytes(uint64(stat.Size))) + "\n")
	}

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
	if stat.Expires != nil && !stat.Expires.IsZero() && !stat.Expires.Equal(timeSentinel) {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "Expires", stat.Expires.Format(printDate)) + "\n")
	}
	if stat.Expiration != nil && !stat.Expiration.IsZero() && !stat.Expiration.Equal(timeSentinel) {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s (lifecycle-rule-id: %s) ", "Expiration",
			stat.Expiration.Local().Format(printDate), stat.ExpirationRuleID) + "\n")
	}
	if len(stat.Checksum) > 0 {
		cs := strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%v", stat.Checksum), "map["), "]")
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %v", "Checksum", cs) + "\n")
	}
	if stat.Restore != nil {
		msgBuilder.WriteString(fmt.Sprintf("%-10s:", "Restore") + "\n")
		if !stat.Restore.ExpiryTime.IsZero() && !stat.Restore.ExpiryTime.Equal(timeSentinel) {
			msgBuilder.WriteString(fmt.Sprintf("  %-10s: %s", "ExpiryTime",
				stat.Restore.ExpiryTime.Local().Format(printDate)) + "\n")
		}
		msgBuilder.WriteString(fmt.Sprintf("  %-10s: %t", "Ongoing",
			stat.Restore.OngoingRestore) + "\n")
	}
	maxKeyMetadata := 0
	maxKeyEncrypted := 0
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

	if maxKeyEncrypted > 0 {
		// Handle various AWS S3 headers, behaviors etc.
		var found bool
		if enabled, ok := stat.Metadata["X-Amz-Server-Side-Encryption-Bucket-Key-Enabled"]; ok {
			if enabled == "true" {
				msgBuilder.WriteString(fmt.Sprintf("%-10s: SSE-%s\n", "Encryption", "KMS"))
			}
			// we still need to treat this as 'true' because X-Amz-Server-Side-Encryption-Bucket-Key-Enabled
			// can be set to 'false' by the server to indicate there is no SSE enabled on the object
			// we shouldn't be printing `unknown` in that scenario.
			found = true
		} else if keyID, ok := stat.Metadata["X-Amz-Server-Side-Encryption-Aws-Kms-Key-Id"]; ok {
			msgBuilder.WriteString(fmt.Sprintf("%-10s: SSE-%s (%s)\n", "Encryption", "KMS", keyID))
			found = true
		} else if _, ok := stat.Metadata["X-Amz-Server-Side-Encryption-Customer-Key-Md5"]; ok {
			msgBuilder.WriteString(fmt.Sprintf("%-10s: SSE-%s\n", "Encryption", "C"))
			found = true
		} else if algo, ok := stat.Metadata["X-Amz-Server-Side-Encryption"]; ok && algo == "AES256" {
			msgBuilder.WriteString(fmt.Sprintf("%-10s: SSE-%s\n", "Encryption", "S3"))
			found = true
		}
		if !found {
			// encryption headers are present but not something we recognize, check `mc stat --debug`
			// to obtain more information and understand if we are missing something.
			msgBuilder.WriteString(fmt.Sprintf("%-10s: SSE-%s\n", "Encryption", "Unknown"))
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

	if stat.ReplicationStatus != "" {
		msgBuilder.WriteString(fmt.Sprintf("%-10s: %s ", "Replication Status", stat.ReplicationStatus))
	}

	msgBuilder.WriteString("\n")

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
	if !c.Expires.IsZero() {
		content.Expires = &c.Expires
	}
	if !c.Expiration.IsZero() {
		content.Expiration = &c.Expiration
	}
	content.ExpirationRuleID = c.ExpirationRuleID
	content.ReplicationStatus = c.ReplicationStatus
	content.Restore = c.Restore
	content.Checksum = c.Checksum
	return content
}

// Return standardized URL to be used to compare later.
func getStandardizedURL(targetURL string) string {
	return filepath.FromSlash(targetURL)
}

// statURL - uses combination of GET listing and HEAD to fetch information of one or more objects
// HEAD can fail with 400 with an SSE-C encrypted object but we still return information gathered
// from GET listing.
func statURL(ctx context.Context, targetURL, versionID string, timeRef time.Time, includeOlderVersions, isIncomplete, isRecursive, headOnly bool, encKeyDB map[string][]prefixSSEPair) *probe.Error {
	clnt, err := newClient(targetURL)
	if err != nil {
		return err
	}

	targetAlias, _, _ := mustExpandAlias(targetURL)
	separator := string(clnt.GetURL().Separator)
	prefixPath := clnt.GetURL().Path
	hasTrailingSlash := strings.HasSuffix(prefixPath, separator)

	if !hasTrailingSlash {
		prefixPath = prefixPath[:strings.LastIndex(prefixPath, separator)+1]
	}

	if headOnly || versionID != "" {
		url := getStandardizedURL(targetURL)

		_, stat, err := url2Stat(ctx, url2StatOptions{
			urlStr: url, versionID: versionID,
			fileAttr: true, encKeyDB: encKeyDB,
			timeRef: timeRef, isZip: false,
			ignoreBucketExistsCheck: false,
			headOnly:                headOnly,
		})
		if err != nil {
			return err
		}

		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(stat.URL.Path)

		// Trim prefix path from the content path.
		stat.URL.Path = strings.TrimPrefix(contentURL, filepath.ToSlash(prefixPath))

		printMsg(parseStat(stat))
		return nil
	}

	// if stat is on a bucket and non-recursive mode, serve the bucket metadata
	if !isRecursive && !hasTrailingSlash {
		bstat, err := clnt.GetBucketInfo(ctx)
		if err == nil {
			// Convert any os specific delimiters to "/".
			contentURL := filepath.ToSlash(bstat.URL.Path)
			prefixPath = filepath.ToSlash(prefixPath)
			// Trim prefix path from the content path.
			contentURL = strings.TrimPrefix(contentURL, prefixPath)
			bstat.URL.Path = contentURL

			if bstat.Date.IsZero() || bstat.Date.Equal(timeSentinel) {
				bstat.Date = time.Now()
			}

			var bu madmin.BucketUsageInfo

			adminClient, _ := newAdminClient(targetURL)
			if adminClient != nil {
				// Create a new MinIO Admin Client
				duinfo, e := adminClient.DataUsageInfo(ctx)
				if e == nil {
					bu = duinfo.BucketsUsage[bstat.Key]
				}
			}

			if prefixPath != "/" {
				bstat.Prefix = true
			}

			printMsg(bucketInfoMessage{
				Status:     "success",
				BucketInfo: bstat,
				Usage:      bu,
			})

			return nil
		}
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

	var e error
	var found int
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
			e = exitStatus(globalErrorExitStatus) // Set the exit status.
			continue
		}
		found++

		if content.StorageClass == s3StorageClassGlacier {
			continue
		}

		url := getStandardizedURL(targetAlias + getKey(content))
		standardizedURL := getStandardizedURL(targetURL)

		if !isRecursive && !strings.HasPrefix(filepath.FromSlash(url), standardizedURL) && !filepath.IsAbs(url) {
			return errTargetNotFound(targetURL).Trace(url, standardizedURL)
		}

		if versionID != "" {
			if versionID != content.VersionID {
				continue
			}
		}
		_, stat, err := url2Stat(ctx, url2StatOptions{
			urlStr: url, versionID: content.VersionID,
			fileAttr: true, encKeyDB: encKeyDB,
			timeRef: timeRef, isZip: false,
			ignoreBucketExistsCheck: false,
		})
		if err != nil {
			return err.Trace(url)
		}

		// Convert any os specific delimiters to "/".
		contentURL := filepath.ToSlash(stat.URL.Path)
		prefixPath = filepath.ToSlash(prefixPath)
		// Trim prefix path from the content path.
		contentURL = strings.TrimPrefix(contentURL, prefixPath)
		stat.URL.Path = contentURL

		printMsg(parseStat(stat))
	}

	if found <= 0 {
		return probe.NewError(ObjectMissing{timeRef})
	}

	return probe.NewError(e)
}

// BucketInfo holds info about a bucket
type BucketInfo struct {
	URL        ClientURL   `json:"-"`
	Key        string      `json:"name"`
	Date       time.Time   `json:"lastModified"`
	Size       int64       `json:"size"`
	Type       os.FileMode `json:"-"`
	Prefix     bool        `json:"-"`
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
	Status string `json:"status"`
	BucketInfo
	Usage madmin.BucketUsageInfo
}

func (v bucketInfoMessage) JSON() string {
	v.Status = "success"
	v.Key = getKey(&ClientContent{URL: v.URL, Type: v.Type})
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", " ")
	// Disable escaping special chars to display XML tags correctly
	enc.SetEscapeHTML(false)

	fatalIf(probe.NewError(enc.Encode(v)), "Unable to marshal into JSON.")
	return buf.String()
}

func countDigits(num uint64) (count uint) {
	for num > 0 {
		num /= 10
		count++
	}
	return
}

func (v bucketInfoMessage) String() string {
	var b strings.Builder

	keyStr := getKey(&ClientContent{URL: v.URL, Type: v.Type})
	keyStr = strings.TrimSuffix(keyStr, slashSeperator)
	key := fmt.Sprintf("%-10s: %s", "Name", keyStr)
	b.WriteString(console.Colorize("Title", key) + "\n")
	if !v.Date.IsZero() && !v.Date.Equal(timeSentinel) {
		b.WriteString(fmt.Sprintf("%-10s: %s ", "Date", v.Date.Format(printDate)) + "\n")
	}
	b.WriteString(fmt.Sprintf("%-10s: %-6s \n", "Size", "N/A"))

	fType := func() string {
		if v.Prefix {
			return "prefix"
		}
		if v.Type.IsDir() {
			return "folder"
		}
		return "file"
	}()
	b.WriteString(fmt.Sprintf("%-10s: %s \n", "Type", fType))
	fmt.Fprintf(&b, "\n")

	if !v.Prefix {
		fmt.Fprint(&b, console.Colorize("Title", "Properties:\n"))
		fmt.Fprint(&b, prettyPrintBucketMetadata(v.BucketInfo))
		fmt.Fprintf(&b, "\n")
	}

	fmt.Fprint(&b, console.Colorize("Title", "Usage:\n"))

	fmt.Fprintf(&b, "%16s: %s\n", "Total size", console.Colorize("Count", humanize.IBytes(v.Usage.Size)))
	fmt.Fprintf(&b, "%16s: %s\n", "Objects count", console.Colorize("Count", humanize.Comma(int64(v.Usage.ObjectsCount))))
	fmt.Fprintf(&b, "%16s: %s\n", "Versions count", console.Colorize("Count", humanize.Comma(int64(v.Usage.VersionsCount))))
	fmt.Fprintf(&b, "\n")

	if len(v.Usage.ObjectSizesHistogram) > 0 {
		fmt.Fprint(&b, console.Colorize("Title", "Object sizes histogram:\n"))

		var maxDigits uint
		for _, val := range v.Usage.ObjectSizesHistogram {
			if d := countDigits(val); d > maxDigits {
				maxDigits = d
			}
		}

		var sortedTags []string
		for k := range v.Usage.ObjectSizesHistogram {
			sortedTags = append(sortedTags, k)
		}
		sort.Strings(sortedTags)
		for _, tagName := range sortedTags {
			val, ok := v.Usage.ObjectSizesHistogram[tagName]
			if ok {
				fmt.Fprintf(&b, "   %*d object(s) %s\n", maxDigits, val, tagName)
			}
		}
	}

	return b.String()
}

// Pretty print bucket configuration - used by stat and admin bucket info as well
func prettyPrintBucketMetadata(info BucketInfo) string {
	var b strings.Builder
	placeHolder := ""
	if info.Encryption.Algorithm != "" {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "Encryption: ")
		if info.Encryption.Algorithm == "aws:kms" {
			fmt.Fprint(&b, console.Colorize("Key", "\n\tKey Type: "))
			fmt.Fprint(&b, console.Colorize("Value", "SSE-KMS"))
			fmt.Fprint(&b, console.Colorize("Key", "\n\tKey ID: "))
			fmt.Fprint(&b, console.Colorize("Value", info.Encryption.KeyID))
		} else {
			fmt.Fprint(&b, console.Colorize("Key", "\n\tKey Type: "))
			fmt.Fprint(&b, console.Colorize("Value", strings.ToUpper(info.Encryption.Algorithm)))
		}
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%2s%s", placeHolder, "Versioning: ")
	if info.Versioning.Status == "" {
		fmt.Fprint(&b, console.Colorize("Unset", "Un-versioned"))
	} else {
		fmt.Fprint(&b, console.Colorize("Set", info.Versioning.Status))
	}
	fmt.Fprintln(&b)

	if info.Locking.Mode != "" {
		fmt.Fprintf(&b, "%2s%s\n", placeHolder, "LockConfiguration: ")
		fmt.Fprintf(&b, "%4s%s", placeHolder, "RetentionMode: ")
		fmt.Fprint(&b, console.Colorize("Value", info.Locking.Mode))
		fmt.Fprintln(&b)
		fmt.Fprintf(&b, "%4s%s", placeHolder, "Retention Until Date: ")
		fmt.Fprint(&b, console.Colorize("Value", info.Locking.Validity))
		fmt.Fprintln(&b)
	}
	if len(info.Notification.Config.TopicConfigs) > 0 {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "Notification: ")
		fmt.Fprint(&b, console.Colorize("Set", "Set"))
		fmt.Fprintln(&b)
	}
	if info.Replication.Enabled {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "Replication: ")
		fmt.Fprint(&b, console.Colorize("Set", "Enabled"))
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%2s%s", placeHolder, "Location: ")
	fmt.Fprint(&b, console.Colorize("Generic", info.Location))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%2s%s", placeHolder, "Anonymous: ")
	if info.Policy.Type == "none" {
		fmt.Fprint(&b, console.Colorize("UnSet", "Disabled"))
	} else {
		fmt.Fprint(&b, console.Colorize("Set", "Enabled"))
	}
	fmt.Fprintln(&b)
	if info.Tags() != "" {
		fmt.Fprintf(&b, "%2s%s", placeHolder, "Tagging: ")
		fmt.Fprint(&b, console.Colorize("Generic", info.Tags()))
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%2s%s", placeHolder, "ILM: ")
	if info.ILM.Config != nil {
		fmt.Fprint(&b, console.Colorize("Set", "Enabled"))
	} else {
		fmt.Fprint(&b, console.Colorize("UnSet", "Disabled"))
	}
	fmt.Fprintln(&b)

	return b.String()
}
