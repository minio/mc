// Copyright (c) 2015-2022 MinIO, Inc.
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
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/mimedb"
)

// ClientURL url client url structure
type ClientURL struct {
	Type            ClientURLType
	Scheme          string
	Host            string
	Path            string
	SchemeSeparator string
	Separator       rune
}

// ClientURLType - enum of different url types
type ClientURLType int

// url2StatOptions - convert url to stat options
type url2StatOptions struct {
	urlStr, versionID       string
	fileAttr                bool
	encKeyDB                map[string][]prefixSSEPair
	timeRef                 time.Time
	isZip                   bool
	headOnly                bool
	ignoreBucketExistsCheck bool
}

// enum types
const (
	objectStorage = iota // MinIO and S3 compatible cloud storage
	fileSystem           // POSIX compatible file systems
)

// Maybe rawurl is of the form scheme:path. (Scheme must be [a-zA-Z][a-zA-Z0-9+-.]*)
// If so, return scheme, path; else return "", rawurl.
func getScheme(rawurl string) (scheme, path string) {
	urlSplits := strings.Split(rawurl, "://")
	if len(urlSplits) == 2 {
		scheme, uri := urlSplits[0], "//"+urlSplits[1]
		// ignore numbers in scheme
		validScheme := regexp.MustCompile("^[a-zA-Z]+$")
		if uri != "" {
			if validScheme.MatchString(scheme) {
				return scheme, uri
			}
		}
	}
	return "", rawurl
}

// Assuming s is of the form [s delimiter s].
// If so, return s, [delimiter]s or return s, s if cutdelimiter == true
// If no delimiter found return s, "".
func splitSpecial(s, delimiter string, cutdelimiter bool) (string, string) {
	i := strings.Index(s, delimiter)
	if i < 0 {
		// if delimiter not found return as is.
		return s, ""
	}
	// if delimiter should be removed, remove it.
	if cutdelimiter {
		return s[0:i], s[i+len(delimiter):]
	}
	// return split strings with delimiter
	return s[0:i], s[i:]
}

// getHost - extract host from authority string, we do not support ftp style username@ yet.
func getHost(authority string) (host string) {
	i := strings.LastIndex(authority, "@")
	if i >= 0 {
		// TODO support, username@password style userinfo, useful for ftp support.
		return
	}
	return authority
}

// newClientURL returns an abstracted URL for filesystems and object storage.
func newClientURL(urlStr string) *ClientURL {
	scheme, rest := getScheme(urlStr)
	if strings.HasPrefix(rest, "//") {
		// if rest has '//' prefix, skip them
		var authority string
		authority, rest = splitSpecial(rest[2:], "/", false)
		if rest == "" {
			rest = "/"
		}
		host := getHost(authority)
		if host != "" && (scheme == "http" || scheme == "https") {
			return &ClientURL{
				Scheme:          scheme,
				Type:            objectStorage,
				Host:            host,
				Path:            rest,
				SchemeSeparator: "://",
				Separator:       '/',
			}
		}
	}
	return &ClientURL{
		Type:      fileSystem,
		Path:      rest,
		Separator: filepath.Separator,
	}
}

// joinURLs join two input urls and returns a url
func joinURLs(url1, url2 *ClientURL) *ClientURL {
	var url1Path, url2Path string
	url1Path = filepath.ToSlash(url1.Path)
	url2Path = filepath.ToSlash(url2.Path)
	if strings.HasSuffix(url1Path, "/") {
		url1.Path = url1Path + strings.TrimPrefix(url2Path, "/")
	} else {
		url1.Path = url1Path + "/" + strings.TrimPrefix(url2Path, "/")
	}
	return url1
}

// Clone the url into a new object.
func (u ClientURL) Clone() ClientURL {
	return ClientURL{
		Type:            u.Type,
		Scheme:          u.Scheme,
		Host:            u.Host,
		Path:            u.Path,
		SchemeSeparator: u.SchemeSeparator,
		Separator:       u.Separator,
	}
}

// String convert URL into its canonical form.
func (u ClientURL) String() string {
	var buf bytes.Buffer
	// if fileSystem no translation needed, return as is.
	if u.Type == fileSystem {
		return u.Path
	}
	// if objectStorage convert from any non standard paths to a supported URL path style.
	if u.Type == objectStorage {
		buf.WriteString(u.Scheme)
		buf.WriteByte(':')
		buf.WriteString("//")
		if h := u.Host; h != "" {
			buf.WriteString(h)
		}
		switch runtime.GOOS {
		case "windows":
			if u.Path != "" && u.Path[0] != '\\' && u.Host != "" && u.Path[0] != '/' {
				buf.WriteByte('/')
			}
			buf.WriteString(strings.ReplaceAll(u.Path, "\\", "/"))
		default:
			if u.Path != "" && u.Path[0] != '/' && u.Host != "" {
				buf.WriteByte('/')
			}
			buf.WriteString(u.Path)
		}
	}
	return buf.String()
}

// urlJoinPath Join a path to existing URL.
func urlJoinPath(url1, url2 string) string {
	u1 := newClientURL(url1)
	u2 := newClientURL(url2)
	return joinURLs(u1, u2).String()
}

// url2Stat returns stat info for URL - supports bucket, object and a prefixe with or without a trailing slash
func url2Stat(ctx context.Context, opts url2StatOptions) (client Client, content *ClientContent, err *probe.Error) {
	client, err = newClient(opts.urlStr)
	if err != nil {
		return nil, nil, err.Trace(opts.urlStr)
	}
	alias, _ := url2Alias(opts.urlStr)
	sse := getSSE(opts.urlStr, opts.encKeyDB[alias])

	content, err = client.Stat(ctx, StatOptions{
		preserve:           opts.fileAttr,
		sse:                sse,
		timeRef:            opts.timeRef,
		versionID:          opts.versionID,
		isZip:              opts.isZip,
		ignoreBucketExists: opts.ignoreBucketExistsCheck,
		headOnly:           opts.headOnly,
	})
	if err != nil {
		return nil, nil, err.Trace(opts.urlStr)
	}
	return client, content, nil
}

// firstURL2Stat returns the stat info of the first object having the specified prefix
func firstURL2Stat(ctx context.Context, prefix string, timeRef time.Time, isZip bool) (client Client, content *ClientContent, err *probe.Error) {
	client, err = newClient(prefix)
	if err != nil {
		return nil, nil, err.Trace(prefix)
	}
	content = <-client.List(ctx, ListOptions{Recursive: true, TimeRef: timeRef, Count: 1, ListZip: isZip})
	if content == nil {
		return nil, nil, probe.NewError(ObjectMissing{timeRef: timeRef}).Trace(prefix)
	}
	if content.Err != nil {
		return nil, nil, content.Err.Trace(prefix)
	}
	return client, content, nil
}

// url2Alias separates alias and path from the URL. Aliased URL is of
// the form alias/path/to/blah.
func url2Alias(aliasedURL string) (alias, path string) {
	// Save aliased url.
	urlStr := aliasedURL

	// Convert '/' on windows to filepath.Separator.
	urlStr = filepath.FromSlash(urlStr)

	if runtime.GOOS == "windows" {
		// Remove '/' prefix before alias if any to support '\\home' alias
		// style under Windows
		urlStr = strings.TrimPrefix(urlStr, string(filepath.Separator))
	}

	// Remove everything after alias (i.e. after '/').
	urlParts := strings.SplitN(urlStr, string(filepath.Separator), 2)
	if len(urlParts) == 2 {
		// Convert windows style path separator to Unix style.
		return urlParts[0], urlParts[1]
	}
	return urlParts[0], ""
}

// guessURLContentType - guess content-type of the URL.
// on failure just return 'application/octet-stream'.
func guessURLContentType(urlStr string) string {
	url := newClientURL(urlStr)
	contentType := mimedb.TypeByExtension(filepath.Ext(url.Path))
	return contentType
}

// urlParts - split URL into parts.
func urlParts(urlStr string) []string {
	// Convert '/' on windows to filepath.Separator.
	urlStr = filepath.FromSlash(urlStr)

	if runtime.GOOS == "windows" {
		// Remove '/' prefix before alias if any to support '\\home' alias
		// style under Windows
		urlStr = strings.TrimPrefix(urlStr, string(filepath.Separator))
	}

	// Remove everything after alias (i.e. after '/').
	return strings.Split(urlStr, string(filepath.Separator))
}

// isURLPrefix - check if source and destination be subdirectories of each other
func isURLPrefix(src string, dest string) bool {
	srcURLParts := urlParts(src)
	dstURLParts := urlParts(dest)
	minIndex := min(len(srcURLParts), len(dstURLParts))
	isPrefix := true
	for i := range minIndex {
		// if one of the URLs ends with '/' and other does not
		if (i == minIndex-1) && (dstURLParts[i] == "" || srcURLParts[i] == "" || dstURLParts[i] == "*" || srcURLParts[i] == "*") {
			continue
		}
		if srcURLParts[i] != dstURLParts[i] {
			isPrefix = false
			break
		}
	}
	return isPrefix
}
