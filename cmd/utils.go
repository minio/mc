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
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-ieproxy"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"

	jwtgo "github.com/golang-jwt/jwt/v4"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

func isErrIgnored(err *probe.Error) (ignored bool) {
	// For all non critical errors we can continue for the remaining files.
	switch e := err.ToGoError().(type) {
	// Handle these specifically for filesystem related errors.
	case BrokenSymlink, TooManyLevelsSymlink, PathNotFound:
		ignored = true
	// Handle these specifically for object storage related errors.
	case BucketNameEmpty, ObjectMissing, ObjectAlreadyExists:
		ignored = true
	case ObjectAlreadyExistsAsDirectory, BucketDoesNotExist, BucketInvalid:
		ignored = true
	case minio.ErrorResponse:
		ignored = strings.Contains(e.Error(), "The specified key does not exist")
	default:
		ignored = false
	}
	return ignored
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyz01234569"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// UTCNow - returns current UTC time.
func UTCNow() time.Time {
	return time.Now().UTC()
}

// randString generates random names and prepends them with a known prefix.
func randString(n int, src rand.Source, prefix string) string {
	if n == 0 {
		return prefix
	}
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	x := n / 2
	if x == 0 {
		x = 1
	}
	return prefix + string(b[0:x])
}

// printTLSCertInfo prints some fields of the certificates received from the server.
// Fields will be inspected by the user, so they must be conscise and useful
func printTLSCertInfo(t *tls.ConnectionState) {
	if globalDebug {
		for _, cert := range t.PeerCertificates {
			console.Debugln("TLS Certificate found: ")
			if len(cert.Issuer.Country) > 0 {
				console.Debugln(" >> Country: " + cert.Issuer.Country[0])
			}
			if len(cert.Issuer.Organization) > 0 {
				console.Debugln(" >> Organization: " + cert.Issuer.Organization[0])
			}
			console.Debugln(" >> Expires: " + cert.NotAfter.String())
		}
	}
}

// splitStr splits a string into n parts, empty strings are added
// if we are not able to reach n elements
func splitStr(path, sep string, n int) []string {
	splits := strings.SplitN(path, sep, n)
	// Add empty strings if we found elements less than nr
	for i := n - len(splits); i > 0; i-- {
		splits = append(splits, "")
	}
	return splits
}

// NewS3Config simply creates a new Config struct using the passed
// parameters.
func NewS3Config(alias, urlStr string, aliasCfg *aliasConfigV10) *Config {
	// We have a valid alias and hostConfig. We populate the
	// credentials from the match found in the config file.
	s3Config := new(Config)

	s3Config.AppName = filepath.Base(os.Args[0])
	s3Config.AppVersion = ReleaseTag
	s3Config.Debug = globalDebug
	s3Config.Insecure = globalInsecure
	s3Config.ConnReadDeadline = globalConnReadDeadline
	s3Config.ConnWriteDeadline = globalConnWriteDeadline
	s3Config.UploadLimit = int64(globalLimitUpload)
	s3Config.DownloadLimit = int64(globalLimitDownload)

	s3Config.HostURL = urlStr
	s3Config.Alias = alias
	if aliasCfg != nil {
		s3Config.AccessKey = aliasCfg.AccessKey
		s3Config.SecretKey = aliasCfg.SecretKey
		s3Config.SessionToken = aliasCfg.SessionToken
		s3Config.Signature = aliasCfg.API
		s3Config.Lookup = getLookupType(aliasCfg.Path)
	}
	return s3Config
}

// lineTrunc - truncates a string to the given maximum length by
// adding ellipsis in the middle
func lineTrunc(content string, maxLen int) string {
	runes := []rune(content)
	rlen := len(runes)
	if rlen <= maxLen {
		return content
	}
	halfLen := maxLen / 2
	fstPart := string(runes[0:halfLen])
	sndPart := string(runes[rlen-halfLen:])
	return fstPart + "…" + sndPart
}

// isOlder returns true if the passed object is older than olderRef
func isOlder(ti time.Time, olderRef string) bool {
	if olderRef == "" {
		return false
	}
	objectAge := time.Since(ti)
	olderThan, e := ParseDuration(olderRef)
	if e != nil {
		for _, format := range rewindSupportedFormat {
			if t, e2 := time.Parse(format, olderRef); e2 == nil {
				olderThan = Duration(time.Since(t))
				e = nil
				break
			}
		}
	}
	fatalIf(probe.NewError(e), "Unable to parse olderThan=`"+olderRef+"`. Supply relative '7d6h2m' or absolute '"+printDate+"'.")
	return objectAge < time.Duration(olderThan)
}

// isNewer returns true if the passed object is newer than newerRef
func isNewer(ti time.Time, newerRef string) bool {
	if newerRef == "" {
		return false
	}

	objectAge := time.Since(ti)
	newerThan, e := ParseDuration(newerRef)
	if e != nil {
		for _, format := range rewindSupportedFormat {
			if t, e2 := time.Parse(format, newerRef); e2 == nil {
				newerThan = Duration(time.Since(t))
				e = nil
				break
			}
		}
	}
	fatalIf(probe.NewError(e), "Unable to parse newerThan=`"+newerRef+"`. Supply relative '7d6h2m' or absolute '"+printDate+"'.")
	return objectAge >= time.Duration(newerThan)
}

// getLookupType returns the minio.BucketLookupType for lookup
// option entered on the command line
func getLookupType(l string) minio.BucketLookupType {
	l = strings.ToLower(l)
	switch l {
	case "off":
		return minio.BucketLookupDNS
	case "on":
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

// Return true if target url is a part of a source url such as:
// alias/bucket/ and alias/bucket/dir/, however
func isURLContains(srcURL, tgtURL, sep string) bool {
	// Add a separator to source url if not found
	if !strings.HasSuffix(srcURL, sep) {
		srcURL += sep
	}
	if !strings.HasSuffix(tgtURL, sep) {
		tgtURL += sep
	}
	// Check if we are going to copy a directory into itself
	if strings.HasPrefix(tgtURL, srcURL) {
		return true
	}
	return false
}

// ErrInvalidFileSystemAttribute reflects invalid fily system attribute
var ErrInvalidFileSystemAttribute = errors.New("Error in parsing file system attribute")

func parseAtimeMtime(attr map[string]string) (atime, mtime time.Time, err *probe.Error) {
	if val, ok := attr["atime"]; ok {
		vals := strings.SplitN(val, "#", 2)
		atim, e := strconv.ParseInt(vals[0], 10, 64)
		if e != nil {
			return atime, mtime, probe.NewError(e)
		}
		var atimnsec int64
		if len(vals) == 2 {
			atimnsec, e = strconv.ParseInt(vals[1], 10, 64)
			if e != nil {
				return atime, mtime, probe.NewError(e)
			}
		}
		atime = time.Unix(atim, atimnsec)
	}

	if val, ok := attr["mtime"]; ok {
		vals := strings.SplitN(val, "#", 2)
		mtim, e := strconv.ParseInt(vals[0], 10, 64)
		if e != nil {
			return atime, mtime, probe.NewError(e)
		}
		var mtimnsec int64
		if len(vals) == 2 {
			mtimnsec, e = strconv.ParseInt(vals[1], 10, 64)
			if e != nil {
				return atime, mtime, probe.NewError(e)
			}
		}
		mtime = time.Unix(mtim, mtimnsec)
	}
	return atime, mtime, nil
}

// Returns a map by parsing the value of X-Amz-Meta-Mc-Attrs/X-Amz-Meta-s3Cmd-Attrs
func parseAttribute(meta map[string]string) (map[string]string, error) {
	attribute := make(map[string]string)
	if meta == nil {
		return attribute, nil
	}

	parseAttrs := func(attrs string) error {
		var err error
		param := strings.Split(attrs, "/")
		for _, val := range param {
			attr := strings.TrimSpace(val)
			if attr == "" {
				err = ErrInvalidFileSystemAttribute
			} else {
				attrVal := strings.Split(attr, ":")
				if len(attrVal) == 2 {
					attribute[strings.TrimSpace(attrVal[0])] = strings.TrimSpace(attrVal[1])
				} else if len(attrVal) == 1 {
					attribute[attrVal[0]] = ""
				} else {
					err = ErrInvalidFileSystemAttribute
				}
			}
		}
		return err
	}

	if attrs, ok := meta[metadataKey]; ok {
		if err := parseAttrs(attrs); err != nil {
			return attribute, err
		}
	}

	if attrs, ok := meta[metadataKeyS3Cmd]; ok {
		if err := parseAttrs(attrs); err != nil {
			return attribute, err
		}
	}

	return attribute, nil
}

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var reAnsi = regexp.MustCompile(ansi)

func centerText(s string, w int) string {
	var sb strings.Builder
	textWithoutColor := reAnsi.ReplaceAllString(s, "")
	length := len(textWithoutColor)
	padding := float64(w-length) / 2
	fmt.Fprintf(&sb, "%s", bytes.Repeat([]byte{' '}, int(math.Ceil(padding))))
	fmt.Fprintf(&sb, "%s", s)
	fmt.Fprintf(&sb, "%s", bytes.Repeat([]byte{' '}, int(math.Floor(padding))))
	return sb.String()
}

func getClient(aliasURL string) *madmin.AdminClient {
	client, err := newAdminClient(aliasURL)
	fatalIf(err, "Unable to initialize admin connection.")
	return client
}

func httpClient(reqTimeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: reqTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			Proxy: ieproxy.GetProxyFunc(),
			TLSClientConfig: &tls.Config{
				RootCAs:            globalRootCAs,
				InsecureSkipVerify: globalInsecure,
				// Can't use SSLv3 because of POODLE and BEAST
				// Can't use TLSv1.0 because of POODLE and BEAST using CBC cipher
				// Can't use TLSv1.1 because of RC4 cipher usage
				MinVersion: tls.VersionTLS12,
			},
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 10 * time.Second,
		},
	}
}

func getPrometheusToken(hostConfig *aliasConfigV10) (string, error) {
	jwt := jwtgo.NewWithClaims(jwtgo.SigningMethodHS512, jwtgo.RegisteredClaims{
		ExpiresAt: jwtgo.NewNumericDate(UTCNow().Add(defaultPrometheusJWTExpiry)),
		Subject:   hostConfig.AccessKey,
		Issuer:    "prometheus",
	})

	token, e := jwt.SignedString([]byte(hostConfig.SecretKey))
	if e != nil {
		return "", e
	}
	return token, nil
}

// conservativeFileName returns a conservative file name
func conservativeFileName(s string) string {
	return strings.Trim(strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case strings.ContainsAny(string(r), "+-_%()[]!@"):
			return r
		default:
			return '_'
		}
	}, s), "_")
}
