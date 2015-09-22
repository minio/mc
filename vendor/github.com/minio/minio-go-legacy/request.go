/*
 * Minio Go Library for Amazon S3 Legacy v2 Signature Compatible Cloud Storage (C) 2015 Minio, Inc.
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
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// operation - rest operation
type operation struct {
	HTTPServer string
	HTTPMethod string
	HTTPPath   string
}

// request - a http request
type request struct {
	req     *http.Request
	config  *Config
	body    io.ReadSeeker
	expires int64
}

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
	pathSplits := strings.SplitN(path, "?", 2)
	splits := strings.SplitN(pathSplits[0], separator, 3)
	switch len(splits) {
	case 0, 1:
		fallthrough
	case 2:
		objectName = ""
	case 3:
		objectName = splits[2]
	}
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

func (op *operation) getRequestURL(config Config) (url string) {
	// parse URL for the combination of HTTPServer + HTTPPath
	if config.Region != "milkyway" {
		// if virtual style hosts, bucket name is not needed to be part of path
		if config.isVirtualStyle {
			url = op.HTTPServer + separator + path2Object(op.HTTPPath)
			query := path2Query(op.HTTPPath)
			// verify if there is a query string to
			if query != "" {
				url = url + "?" + query
			}
		} else {
			url = op.HTTPServer + op.HTTPPath
		}
	} else {
		url = op.HTTPServer + op.HTTPPath
	}
	return
}

func httpNewRequest(method, urlStr string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	// make sure to encode properly, url.Parse in golang is buggy and creates erroneous encoding
	uEncoded := u
	bucketName, objectName := path2BucketAndObject(uEncoded.Path)
	if objectName != "" {
		encodedObjectName, err := urlEncodeName(objectName)
		if err != nil {
			return nil, err
		}
		uEncoded.Opaque = "//" + uEncoded.Host + separator + bucketName + separator + encodedObjectName
	} else {
		uEncoded.Opaque = "//" + uEncoded.Host + separator + bucketName
	}
	rc, ok := body.(io.ReadCloser)
	if !ok && body != nil {
		rc = ioutil.NopCloser(body)
	}
	req := &http.Request{
		Method:     method,
		URL:        uEncoded,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       rc,
		Host:       uEncoded.Host,
	}
	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
		}
	}
	return req, nil
}

func newPresignedRequest(op *operation, config *Config, expires int64) (*request, error) {
	// if no method default to POST
	method := op.HTTPMethod
	if method == "" {
		method = "POST"
	}

	u := op.getRequestURL(*config)

	// get a new HTTP request, for the requested method
	req, err := httpNewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}

	// set UserAgent
	req.Header.Set("User-Agent", config.userAgent)

	// set Accept header for response encoding style, if available
	if config.AcceptType != "" {
		req.Header.Set("Accept", config.AcceptType)
	}

	// save for subsequent use
	r := new(request)
	r.config = config
	r.expires = expires
	r.req = req
	r.body = nil

	return r, nil
}

// newUnauthenticatedRequest - instantiate a new unauthenticated request
func newUnauthenticatedRequest(op *operation, config *Config, body io.Reader) (*request, error) {
	// if no method default to POST
	method := op.HTTPMethod
	if method == "" {
		method = "POST"
	}

	u := op.getRequestURL(*config)

	// get a new HTTP request, for the requested method
	req, err := httpNewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}

	// set UserAgent
	req.Header.Set("User-Agent", config.userAgent)

	// set Accept header for response encoding style, if available
	if config.AcceptType != "" {
		req.Header.Set("Accept", config.AcceptType)
	}

	// add body
	switch {
	case body == nil:
		req.Body = nil
	default:
		req.Body = ioutil.NopCloser(body)
	}

	// save for subsequent use
	r := new(request)
	r.req = req
	r.config = config

	return r, nil
}

// newRequest - instantiate a new request
func newRequest(op *operation, config *Config, body io.ReadSeeker) (*request, error) {
	// if no method default to POST
	method := op.HTTPMethod
	if method == "" {
		method = "POST"
	}

	u := op.getRequestURL(*config)

	// get a new HTTP request, for the requested method
	req, err := httpNewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}

	// set UserAgent
	req.Header.Set("User-Agent", config.userAgent)

	// set Accept header for response encoding style, if available
	if config.AcceptType != "" {
		req.Header.Set("Accept", config.AcceptType)
	}

	// add body
	switch {
	case body == nil:
		req.Body = nil
	default:
		req.Body = ioutil.NopCloser(body)
	}

	// save for subsequent use
	r := new(request)
	r.config = config
	r.req = req
	r.body = body

	return r, nil
}

// Do - start the request
func (r *request) Do() (resp *http.Response, err error) {
	if r.config.AccessKeyID != "" && r.config.SecretAccessKey != "" {
		r.SignV2()
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

// Set - set additional headers if any
func (r *request) Set(key, value string) {
	r.req.Header.Set(key, value)
}

// Get - get header values
func (r *request) Get(key string) string {
	return r.req.Header.Get(key)
}

// https://${S3_BUCKET}.s3.amazonaws.com/${S3_OBJECT}?AWSAccessKeyId=${S3_ACCESS_KEY}&Expires=${TIMESTAMP}&Signature=${SIGNATURE}
func (r *request) PreSignV2() (string, error) {
	if r.config.AccessKeyID == "" || r.config.SecretAccessKey == "" {
		return "", errors.New("presign requires accesskey and secretkey")
	}
	// Add date if not present
	d := time.Now().UTC()
	if date := r.Get("Date"); date == "" {
		r.Set("Date", d.Format(http.TimeFormat))
	}
	epochExpires := d.Unix() + r.expires
	signText := fmt.Sprintf("GET\n\n\n%d\n%s", epochExpires, r.req.URL.Path)
	hm := hmac.New(sha1.New, []byte(r.config.SecretAccessKey))
	hm.Write([]byte(signText))

	query := r.req.URL.Query()
	query.Set("AWSAccessKeyId", r.config.AccessKeyID)
	query.Set("Expires", strconv.FormatInt(epochExpires, 10))
	query.Set("Signature", base64.StdEncoding.EncodeToString(hm.Sum(nil)))
	r.req.URL.RawQuery = query.Encode()

	return r.req.URL.String(), nil
}

// Authorization = "AWS" + " " + AWSAccessKeyId + ":" + Signature;
// Signature = Base64( HMAC-SHA1( YourSecretAccessKeyID, UTF-8-Encoding-Of( StringToSign ) ) );
//
// StringToSign = HTTP-Verb + "\n" +
//  	Content-MD5 + "\n" +
//  	Content-Type + "\n" +
//  	Date + "\n" +
//  	CanonicalizedAmzHeaders +
//  	CanonicalizedResource;
//
// CanonicalizedResource = [ "/" + Bucket ] +
//  	<HTTP-Request-URI, from the protocol name up to the query string> +
//  	[ subresource, if present. For example "?acl", "?location", "?logging", or "?torrent"];
//
// CanonicalizedAmzHeaders = <described below>

// SignV2 the request before Do() (version 2.0)
func (r *request) SignV2() {
	// Add date if not present
	if date := r.Get("Date"); date == "" {
		r.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}
	// Calculate HMAC for secretAccessKey
	hm := hmac.New(sha1.New, []byte(r.config.SecretAccessKey))
	hm.Write([]byte(r.getStringToSign()))

	// prepare auth header
	authHeader := new(bytes.Buffer)
	authHeader.WriteString(fmt.Sprintf("AWS %s:", r.config.AccessKeyID))
	encoder := base64.NewEncoder(base64.StdEncoding, authHeader)
	encoder.Write(hm.Sum(nil))
	encoder.Close()

	// Set Authorization header
	r.req.Header.Set("Authorization", authHeader.String())
}

// From the Amazon docs:
//
// StringToSign = HTTP-Verb + "\n" +
// 	 Content-MD5 + "\n" +
//	 Content-Type + "\n" +
//	 Date + "\n" +
//	 CanonicalizedAmzHeaders +
//	 CanonicalizedResource;
func (r *request) getStringToSign() string {
	buf := new(bytes.Buffer)
	// write standard headers
	r.writeDefaultHeaders(buf)
	// write canonicalized AMZ headers if any
	r.writeCanonicalizedAmzHeaders(buf)
	// write canonicalized Query resources if any
	r.writeCanonicalizedResource(buf)
	return buf.String()
}

func (r *request) writeDefaultHeaders(buf *bytes.Buffer) {
	buf.WriteString(r.req.Method)
	buf.WriteByte('\n')
	buf.WriteString(r.req.Header.Get("Content-MD5"))
	buf.WriteByte('\n')
	buf.WriteString(r.req.Header.Get("Content-Type"))
	buf.WriteByte('\n')
	buf.WriteString(r.req.Header.Get("Date"))
	buf.WriteByte('\n')
}

func (r *request) writeCanonicalizedAmzHeaders(buf *bytes.Buffer) {
	var amzHeaders []string
	vals := make(map[string][]string)
	for k, vv := range r.req.Header {
		// all the AMZ headers go lower
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "x-amz") {
			amzHeaders = append(amzHeaders, lk)
			vals[lk] = vv
		}
	}
	sort.Strings(amzHeaders)
	for _, k := range amzHeaders {
		buf.WriteString(k)
		buf.WriteByte(':')
		for idx, v := range vals[k] {
			if idx > 0 {
				buf.WriteByte(',')
			}
			if strings.Contains(v, "\n") {
				// TODO: "Unfold" long headers that
				// span multiple lines (as allowed by
				// RFC 2616, section 4.2) by replacing
				// the folding white-space (including
				// new-line) by a single space.
				buf.WriteString(v)
			} else {
				buf.WriteString(v)
			}
		}
		buf.WriteByte('\n')
	}
}

// Must be sorted:
var resourceList = []string{
	"acl",
	"location",
	"logging",
	"notification",
	"partNumber",
	"policy",
	"response-content-type",
	"response-content-language",
	"response-expires",
	"response-cache-control",
	"response-content-disposition",
	"response-content-encoding",
	"requestPayment",
	"torrent",
	"uploadId",
	"uploads",
	"versionId",
	"versioning",
	"versions",
	"website",
}

// From the Amazon docs:
//
// CanonicalizedResource = [ "/" + Bucket ] +
// 	  <HTTP-Request-URI, from the protocol name up to the query string> +
// 	  [ sub-resource, if present. For example "?acl", "?location", "?logging", or "?torrent"];
func (r *request) writeCanonicalizedResource(buf *bytes.Buffer) error {
	requestURL := r.req.URL
	encodedURLPath, err := urlEncodeName(requestURL.Path)
	if err != nil {
		return err
	}
	buf.WriteString(encodedURLPath)
	sort.Strings(resourceList)
	if requestURL.RawQuery != "" {
		var n int
		vals, _ := url.ParseQuery(requestURL.RawQuery)
		// loop through all the supported resourceList
		for _, resource := range resourceList {
			if vv, ok := vals[resource]; ok && len(vv) > 0 {
				n++
				// first element
				switch n {
				case 1:
					buf.WriteByte('?')
				// the rest
				default:
					buf.WriteByte('&')
				}
				buf.WriteString(resource)
				// request parameters
				if len(vv[0]) > 0 {
					buf.WriteByte('=')
					buf.WriteString(url.QueryEscape(vv[0]))
				}
			}
		}
	}
	return nil
}
