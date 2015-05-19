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

package client

import (
	"bytes"
	"errors"
	"runtime"
	"strings"

	"github.com/minio/minio/pkg/iodine"
)

// URL client url structure
type URL struct {
	Type   URLType
	Scheme string
	Host   string
	Path   string
}

// URLType - enum of different url types
type URLType int

// enum types
const (
	Object     = iota // Minio and S3 compatible object storage
	Filesystem        // POSIX compatible file systems
)

// Maybe rawurl is of the form scheme:path. (Scheme must be [a-zA-Z][a-zA-Z0-9+-.]*)
// If so, return scheme, path; else return "", rawurl.
func getScheme(rawurl string) (scheme, path string, err error) {
	for i := 0; i < len(rawurl); i++ {
		c := rawurl[i]
		switch {
		// valid characters, do nothing
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// invalid characters, return raw url
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return "", rawurl, nil
			}
		// check if the scheme delimiter is a first character, return missing protocol scheme
		case c == ':':
			if i == 0 {
				return "", "", iodine.New(errors.New("missing protocol scheme"), nil)
			}
			// if not separate them properly
			return rawurl[0:i], rawurl[i+1:], nil
		default:
			// we have encountered an unexpected character, so there is no valid scheme
			return "", rawurl, nil
		}
	}
	return "", rawurl, nil
}

// Maybe s is of the form s d s.  If so, return s, ds (or s, s if cutd == true).
// If not, return s, "".
func split(s string, d string, cutd bool) (string, string) {
	i := strings.Index(s, d)
	if i < 0 {
		return s, ""
	}
	if cutd {
		return s[0:i], s[i+len(d):]
	}
	return s[0:i], s[i:]
}

func getHost(rest string) (host string) {
	i := strings.LastIndex(rest, "@")
	if i < 0 {
		host = rest
		return
	}
	return
}

// Parse url parse
func Parse(urlStr string) (*URL, error) {
	scheme, rest, err := getScheme(urlStr)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	rest, _ = split(rest, "?", true)
	if strings.HasPrefix(rest, "//") {
		// if rest has '//' prefix, skip them
		authority, rest := split(rest[2:], "/", false)
		host := getHost(authority)
		if host != "" && (scheme == "http" || scheme == "https") {
			return &URL{
				Scheme: scheme,
				Type:   Object,
				Host:   host,
				Path:   rest,
			}, nil
		}
	}
	return &URL{
		Type: Filesystem,
		Path: rest,
	}, nil
}

func (u *URL) String() string {
	var buf bytes.Buffer
	if u.Scheme != "" {
		buf.WriteString(u.Scheme)
		buf.WriteByte(':')
	}
	if u.Scheme != "" || u.Host != "" {
		buf.WriteString("//")
		if h := u.Host; h != "" {
			buf.WriteString(h)
		}
	}
	switch runtime.GOOS {
	case "windows":
		if u.Path != "" && u.Path[0] != '\\' && u.Host != "" {
			buf.WriteByte('/')
		}
		buf.WriteString(strings.Replace(u.Path, "\\", "/", -1))
	default:
		if u.Path != "" && u.Path[0] != '/' && u.Host != "" {
			buf.WriteByte('/')
		}
		buf.WriteString(u.Path)
	}

	return buf.String()
}
