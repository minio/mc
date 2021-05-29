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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-ieproxy"
	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"maze.io/x/duration"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
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

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// newRandomID generates a random id of regular lower case and uppercase english characters.
func newRandomID(n int) string {
	rand.Seed(UTCNow().UnixNano())
	sid := make([]rune, n)
	for i := range sid {
		sid[i] = letters[rand.Intn(len(letters))]
	}
	return string(sid)
}

// randString generates random names and prepends them with a known prefix.
func randString(n int, src rand.Source, prefix string) string {
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
	return prefix + string(b[0:30-len(prefix)])
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
func NewS3Config(urlStr string, aliasCfg *aliasConfigV10) *Config {
	// We have a valid alias and hostConfig. We populate the
	// credentials from the match found in the config file.
	s3Config := new(Config)

	s3Config.AppName = filepath.Base(os.Args[0])
	s3Config.AppVersion = ReleaseTag
	s3Config.Debug = globalDebug
	s3Config.Insecure = globalInsecure

	s3Config.HostURL = urlStr
	if aliasCfg != nil {
		s3Config.AccessKey = aliasCfg.AccessKey
		s3Config.SecretKey = aliasCfg.SecretKey
		s3Config.SessionToken = aliasCfg.SessionToken
		s3Config.Signature = aliasCfg.API
	}
	s3Config.Lookup = getLookupType(aliasCfg.Path)
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
	olderThan, e := duration.ParseDuration(olderRef)
	fatalIf(probe.NewError(e), "Unable to parse olderThan=`"+olderRef+"`.")
	return objectAge < time.Duration(olderThan)
}

// isNewer returns true if the passed object is newer than newerRef
func isNewer(ti time.Time, newerRef string) bool {
	if newerRef == "" {
		return false
	}

	objectAge := time.Since(ti)
	newerThan, e := duration.ParseDuration(newerRef)
	fatalIf(probe.NewError(e), "Unable to parse newerThan=`"+newerRef+"`.")
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

// struct representing object prefix and sse keys association.
type prefixSSEPair struct {
	Prefix string
	SSE    encrypt.ServerSide
}

// parse and validate encryption keys entered on command line
func parseAndValidateEncryptionKeys(sseKeys string, sse string) (encMap map[string][]prefixSSEPair, err *probe.Error) {
	encMap, err = parseEncryptionKeys(sseKeys)
	if err != nil {
		return nil, err
	}
	if sse != "" {
		for _, prefix := range strings.Split(sse, ",") {
			alias, _ := url2Alias(prefix)
			encMap[alias] = append(encMap[alias], prefixSSEPair{
				Prefix: prefix,
				SSE:    encrypt.NewSSE(),
			})
		}
	}
	for alias, ps := range encMap {
		if hostCfg := mustGetHostConfig(alias); hostCfg == nil {
			for _, p := range ps {
				return nil, probe.NewError(errors.New("SSE prefix " + p.Prefix + " has invalid alias"))
			}
		}
	}
	return encMap, nil
}

// parse list of comma separated alias/prefix=sse key values entered on command line and
// construct a map of alias to prefix and sse pairs.
func parseEncryptionKeys(sseKeys string) (encMap map[string][]prefixSSEPair, err *probe.Error) {
	encMap = make(map[string][]prefixSSEPair)
	if sseKeys == "" {
		return
	}
	prefix := ""
	index := 0 // start index of prefix
	vs := 0    // start index of sse-c key
	sseKeyLen := 32
	delim := 1
	k := len(sseKeys)
	for index < k {
		i := strings.Index(sseKeys[index:], "=")
		if i == -1 {
			return nil, probe.NewError(errors.New("SSE-C prefix should be of the form prefix1=key1,... "))
		}
		prefix = sseKeys[index : index+i]
		alias, _ := url2Alias(prefix)
		vs = i + 1 + index
		if vs+32 > k {
			return nil, probe.NewError(errors.New("SSE-C key should be 32 bytes long"))
		}
		if (vs+sseKeyLen < k) && sseKeys[vs+sseKeyLen] != ',' {
			return nil, probe.NewError(errors.New("SSE-C prefix=secret should be delimited by , and secret should be 32 bytes long"))
		}
		sseKey := sseKeys[vs : vs+sseKeyLen]
		if _, ok := encMap[alias]; !ok {
			encMap[alias] = make([]prefixSSEPair, 0)
		}
		sse, e := encrypt.NewSSEC([]byte(sseKey))
		if e != nil {
			return nil, probe.NewError(e)
		}
		encMap[alias] = append(encMap[alias], prefixSSEPair{
			Prefix: prefix,
			SSE:    sse,
		})
		// advance index sseKeyLen + delim bytes for the next key start
		index = vs + sseKeyLen + delim
	}

	// Sort encryption keys in descending order of prefix length
	for _, encKeys := range encMap {
		sort.Sort(byPrefixLength(encKeys))
	}

	// Success.
	return encMap, nil
}

// byPrefixLength implements sort.Interface.
type byPrefixLength []prefixSSEPair

func (p byPrefixLength) Len() int { return len(p) }
func (p byPrefixLength) Less(i, j int) bool {
	return len(p[i].Prefix) > len(p[j].Prefix)
}
func (p byPrefixLength) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

// get SSE Key if object prefix matches with given resource.
func getSSE(resource string, encKeys []prefixSSEPair) encrypt.ServerSide {
	for _, k := range encKeys {
		if strings.HasPrefix(resource, k.Prefix) {
			return k.SSE
		}
	}
	return nil
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
		atime = time.Unix(int64(atim), int64(atimnsec))
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
		mtime = time.Unix(int64(mtim), int64(mtimnsec))
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

// Returns true if "s3" is entirely in sub-domain and false otherwise.
// true for s3.amazonaws.com, false for ams3.digitaloceanspaces.com, 192.168.1.12
func matchS3InHost(urlHost string) bool {
	if strings.Contains(urlHost, ":") {
		if host, _, err := net.SplitHostPort(urlHost); err == nil {
			urlHost = host
		}
	}
	fqdnParts := strings.Split(urlHost, ".")
	for _, fqdn := range fqdnParts {
		if fqdn == "s3" {
			return true
		}
	}
	return false
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

func getAliasAndBucket(ctx *cli.Context) (string, string) {
	args := ctx.Args()
	aliasedURL := args.Get(0)
	aliasedURL = filepath.Clean(aliasedURL)
	return url2Alias(aliasedURL)
}

func getClient(aliasURL string) *madmin.AdminClient {
	client, err := newAdminClient(aliasURL)
	fatalIf(err, "Unable to initialize admin connection.")
	return client
}

func httpClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: ieproxy.GetProxyFunc(),
			// need to close connection after usage.
			DisableKeepAlives: true,
		},
	}
}
