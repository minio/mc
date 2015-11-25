/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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

package minio

import (
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"
)

// operation - rest operation.
type operation struct {
	HTTPServer string
	HTTPMethod string
	HTTPPath   string
}

// Request - a http request.
type Request struct {
	req     *http.Request
	config  *Config
	expires int64
}

// requestMetadata a http request metadata.
type requestMetadata struct {
	body               io.ReadCloser
	contentType        string
	contentLength      int64
	sha256PayloadBytes []byte
	md5SumPayloadBytes []byte
}

// Do - start the request.
func (r *Request) Do() (resp *http.Response, err error) {
	// if not an anonymous request, calculate relevant signature.
	if !r.config.isAnonymous() {
		if r.config.Signature.isV2() {
			// if signature version '2' requested, use that.
			r.SignV2()
		}
		if r.config.Signature.isV4() || r.config.Signature.isLatest() {
			// Not a presigned request, set behavior to default.
			presign := false
			r.SignV4(presign)
		}
	}
	transport := http.DefaultTransport
	if r.config.Transport != nil {
		transport = r.config.Transport
	}
	// do not use http.Client{}, while it may seem intuitive but the problem seems to be
	// that http.Client{} internally follows redirects and there is no easier way to disable
	// it from outside using a configuration parameter -
	//     this auto redirect causes complications in verifying subsequent errors
	//
	// The best is to use RoundTrip() directly, so the request comes back to the caller where
	// we are going to handle such replies. And indeed that is the right thing to do here.
	//
	return transport.RoundTrip(r.req)
}

// Set - set additional headers if any.
func (r *Request) Set(key, value string) {
	r.req.Header.Set(key, value)
}

// Get - get header values.
func (r *Request) Get(key string) string {
	return r.req.Header.Get(key)
}

// path2BucketAndObject - extract bucket and object names from URL path.
func path2BucketAndObject(path string) (bucketName, objectName string) {
	pathSplits := strings.SplitN(path, "?", 2)
	splits := strings.SplitN(pathSplits[0], separator, 3)
	switch len(splits) {
	case 0, 1:
		bucketName = ""
		objectName = ""
	case 2:
		bucketName = splits[1]
		objectName = ""
	case 3:
		bucketName = splits[1]
		objectName = splits[2]
	}
	return bucketName, objectName
}

// path2Object gives objectName from URL path
func path2Object(path string) (objectName string) {
	_, objectName = path2BucketAndObject(path)
	return
}

// path2Bucket gives bucketName from URL path
func path2Bucket(path string) (bucketName string) {
	bucketName, _ = path2BucketAndObject(path)
	return
}

// path2Query gives query part from URL path
func path2Query(path string) (query string) {
	pathSplits := strings.SplitN(path, "?", 2)
	if len(pathSplits) > 1 {
		query = pathSplits[1]
	}
	return
}

// getURLEncodedPath encode the strings from UTF-8 byte representations to HTML hex escape sequences
//
// This is necessary since regular url.Parse() and url.Encode() functions do not support UTF-8
// non english characters cannot be parsed due to the nature in which url.Encode() is written
//
// This function on the other hand is a direct replacement for url.Encode() technique to support
// pretty much every UTF-8 character.
func getURLEncodedPath(pathName string) string {
	// if object matches reserved string, no need to encode them
	reservedNames := regexp.MustCompile("^[a-zA-Z0-9-_.~/]+$")
	if reservedNames.MatchString(pathName) {
		return pathName
	}
	var encodedPathname string
	for _, s := range pathName {
		if 'A' <= s && s <= 'Z' || 'a' <= s && s <= 'z' || '0' <= s && s <= '9' { // §2.3 Unreserved characters (mark)
			encodedPathname = encodedPathname + string(s)
			continue
		}
		switch s {
		case '-', '_', '.', '~', '/': // §2.3 Unreserved characters (mark)
			encodedPathname = encodedPathname + string(s)
			continue
		default:
			len := utf8.RuneLen(s)
			if len < 0 {
				// if utf8 cannot convert return the same string as is
				return pathName
			}
			u := make([]byte, len)
			utf8.EncodeRune(u, s)
			for _, r := range u {
				hex := hex.EncodeToString([]byte{r})
				encodedPathname = encodedPathname + "%" + strings.ToUpper(hex)
			}
		}
	}
	return encodedPathname
}

// getRequetURL - get a properly encoded request URL.
func (op *operation) getRequestURL(config Config) (url string) {
	// parse URL for the combination of HTTPServer + HTTPPath
	url = op.HTTPServer + separator
	if !config.isVirtualHostedStyle {
		url += path2Bucket(op.HTTPPath)
	}
	objectName := getURLEncodedPath(path2Object(op.HTTPPath))
	queryPath := path2Query(op.HTTPPath)
	if objectName == "" && queryPath != "" {
		url += "?" + queryPath
		return
	}
	if objectName != "" && queryPath == "" {
		if strings.HasSuffix(url, separator) {
			url += objectName
		} else {
			url += separator + objectName
		}
		return
	}
	if objectName != "" && queryPath != "" {
		if strings.HasSuffix(url, separator) {
			url += objectName + "?" + queryPath
		} else {
			url += separator + objectName + "?" + queryPath
		}
	}
	return
}

func newPresignedRequest(op *operation, config *Config, expires int64) (*Request, error) {
	// if no method default to POST
	method := op.HTTPMethod
	if method == "" {
		method = "POST"
	}

	u := op.getRequestURL(*config)

	// get a new HTTP request, for the requested method
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}

	// set UserAgent
	req.Header.Set("User-Agent", config.userAgent)

	// save for subsequent use
	r := new(Request)
	r.config = config
	r.expires = expires
	r.req = req

	return r, nil
}

// newRequest - instantiate a new request
func newRequest(op *operation, config *Config, metadata requestMetadata) (*Request, error) {
	// if no method default to POST
	method := op.HTTPMethod
	if method == "" {
		method = "POST"
	}

	u := op.getRequestURL(*config)
	// get a new HTTP request, for the requested method
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}

	// set UserAgent
	req.Header.Set("User-Agent", config.userAgent)

	// add body
	switch {
	case metadata.body == nil:
		req.Body = nil
	default:
		req.Body = metadata.body
	}

	// save for subsequent use
	r := new(Request)
	r.config = config
	r.req = req

	// Set contentType for the request.
	if metadata.contentType != "" {
		r.Set("Content-Type", metadata.contentType)
	}

	// set incoming content-length.
	if metadata.contentLength > 0 {
		r.req.ContentLength = metadata.contentLength
	}

	// set sha256 sum for signature calculation only with signature version '4'.
	if r.config.Signature.isV4() || r.config.Signature.isLatest() {
		r.Set("X-Amz-Content-Sha256", hex.EncodeToString(sum256([]byte{})))
		if metadata.sha256PayloadBytes != nil {
			r.Set("X-Amz-Content-Sha256", hex.EncodeToString(metadata.sha256PayloadBytes))
		}
	}
	// set md5Sum for in transit corruption detection.
	if metadata.md5SumPayloadBytes != nil {
		r.Set("Content-MD5", base64.StdEncoding.EncodeToString(metadata.md5SumPayloadBytes))
	}
	return r, nil
}
