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
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
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
	expires string
}

const (
	authHeader    = "AWS4-HMAC-SHA256"
	iso8601Format = "20060102T150405Z"
	yyyymmdd      = "20060102"
)

///
/// Excerpts from @lsegal - https://github.com/aws/aws-sdk-js/issues/659#issuecomment-120477258
///
///  User-Agent:
///
///      This is ignored from signing because signing this causes problems with generating pre-signed URLs
///      (that are executed by other agents) or when customers pass requests through proxies, which may
///      modify the user-agent.
///
///  Content-Length:
///
///      This is ignored from signing because generating a pre-signed URL should not provide a content-length
///      constraint, specifically when vending a S3 pre-signed PUT URL. The corollary to this is that when
///      sending regular requests (non-pre-signed), the signature contains a checksum of the body, which
///      implicitly validates the payload length (since changing the number of bytes would change the checksum)
///      and therefore this header is not valuable in the signature.
///
///  Content-Type:
///
///      Signing this header causes quite a number of problems in browser environments, where browsers
///      like to modify and normalize the content-type header in different ways. There is more information
///      on this in https://github.com/aws/aws-sdk-js/issues/244. Avoiding this field simplifies logic
///      and reduces the possibility of future bugs
///
///  Authorization:
///
///      Is skipped for obvious reasons
///
var ignoredHeaders = map[string]bool{
	"Authorization":  true,
	"Content-Type":   true,
	"Content-Length": true,
	"User-Agent":     true,
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

func newPresignedRequest(op *operation, config *Config, expires string) (*request, error) {
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
		r.SignV4()
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

func (r *request) SetQuery(key, value string) {
	r.req.URL.Query().Set(key, value)
}

func (r *request) AddQuery(key, value string) {
	r.req.URL.Query().Add(key, value)
}

// Set - set additional headers if any
func (r *request) Set(key, value string) {
	r.req.Header.Set(key, value)
}

// Get - get header values
func (r *request) Get(key string) string {
	return r.req.Header.Get(key)
}

// getHashedPayload get the hexadecimal value of the SHA256 hash of the request payload
func (r *request) getHashedPayload() string {
	hash := func() string {
		switch {
		case r.expires != "":
			return "UNSIGNED-PAYLOAD"
		case r.body == nil:
			return hex.EncodeToString(sum256([]byte{}))
		default:
			sum256Bytes, _ := sum256Reader(r.body)
			return hex.EncodeToString(sum256Bytes)
		}
	}
	hashedPayload := hash()
	r.req.Header.Set("X-Amz-Content-Sha256", hashedPayload)
	if hashedPayload == "UNSIGNED-PAYLOAD" {
		r.req.Header.Set("X-Amz-Content-Sha256", "")
	}
	return hashedPayload
}

// getCanonicalHeaders generate a list of request headers with their values
func (r *request) getCanonicalHeaders() string {
	var headers []string
	vals := make(map[string][]string)
	for k, vv := range r.req.Header {
		if _, ok := ignoredHeaders[http.CanonicalHeaderKey(k)]; ok {
			continue // ignored header
		}
		headers = append(headers, strings.ToLower(k))
		vals[strings.ToLower(k)] = vv
	}
	headers = append(headers, "host")
	sort.Strings(headers)

	var buf bytes.Buffer
	for _, k := range headers {
		buf.WriteString(k)
		buf.WriteByte(':')
		switch {
		case k == "host":
			buf.WriteString(r.req.URL.Host)
			fallthrough
		default:
			for idx, v := range vals[k] {
				if idx > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(v)
			}
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

// getSignedHeaders generate a string i.e alphabetically sorted, semicolon-separated list of lowercase request header names
func (r *request) getSignedHeaders() string {
	var headers []string
	for k := range r.req.Header {
		if _, ok := ignoredHeaders[http.CanonicalHeaderKey(k)]; ok {
			continue // ignored header
		}
		headers = append(headers, strings.ToLower(k))
	}
	headers = append(headers, "host")
	sort.Strings(headers)
	return strings.Join(headers, ";")
}

// getCanonicalRequest generate a canonical request of style
//
// canonicalRequest =
//  <HTTPMethod>\n
//  <CanonicalURI>\n
//  <CanonicalQueryString>\n
//  <CanonicalHeaders>\n
//  <SignedHeaders>\n
//  <HashedPayload>
//
func (r *request) getCanonicalRequest(hashedPayload string) string {
	r.req.URL.RawQuery = strings.Replace(r.req.URL.Query().Encode(), "+", "%20", -1)

	// get path URI from Opaque
	uri := strings.Join(strings.Split(r.req.URL.Opaque, "/")[3:], "/")

	canonicalRequest := strings.Join([]string{
		r.req.Method,
		"/" + uri,
		r.req.URL.RawQuery,
		r.getCanonicalHeaders(),
		r.getSignedHeaders(),
		hashedPayload,
	}, "\n")
	return canonicalRequest
}

// getScope generate a string of a specific date, an AWS region, and a service
func (r *request) getScope(t time.Time) string {
	scope := strings.Join([]string{
		t.Format(yyyymmdd),
		r.config.Region,
		"s3",
		"aws4_request",
	}, "/")
	return scope
}

// getStringToSign a string based on selected query values
func (r *request) getStringToSign(canonicalRequest string, t time.Time) string {
	stringToSign := authHeader + "\n" + t.Format(iso8601Format) + "\n"
	stringToSign = stringToSign + r.getScope(t) + "\n"
	stringToSign = stringToSign + hex.EncodeToString(sum256([]byte(canonicalRequest)))
	return stringToSign
}

// getSigningKey hmac seed to calculate final signature
func (r *request) getSigningKey(t time.Time) []byte {
	secret := r.config.SecretAccessKey
	date := sumHMAC([]byte("AWS4"+secret), []byte(t.Format(yyyymmdd)))
	region := sumHMAC(date, []byte(r.config.Region))
	service := sumHMAC(region, []byte("s3"))
	signingKey := sumHMAC(service, []byte("aws4_request"))
	return signingKey
}

// getSignature final signature in hexadecimal form
func (r *request) getSignature(signingKey []byte, stringToSign string) string {
	return hex.EncodeToString(sumHMAC(signingKey, []byte(stringToSign)))
}

// Presign the request, in accordance with - http://docs.aws.amazon.com/AmazonS3/latest/API/sigv4-query-string-auth.html
func (r *request) PreSignV4() (string, error) {
	if r.config.AccessKeyID == "" && r.config.SecretAccessKey == "" {
		return "", errors.New("presign requires accesskey and secretkey")
	}
	r.SignV4()
	return r.req.URL.String(), nil
}

// SignV4 the request before Do(), in accordance with - http://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html
func (r *request) SignV4() {
	query := r.req.URL.Query()
	if r.expires != "" {
		query.Set("X-Amz-Algorithm", authHeader)
	}
	t := time.Now().UTC()
	// Add date if not present
	if r.expires != "" {
		query.Set("X-Amz-Date", t.Format(iso8601Format))
		query.Set("X-Amz-Expires", r.expires)
	} else {
		r.Set("X-Amz-Date", t.Format(iso8601Format))
	}

	hashedPayload := r.getHashedPayload()
	signedHeaders := r.getSignedHeaders()
	if r.expires != "" {
		query.Set("X-Amz-SignedHeaders", signedHeaders)
	}
	scope := r.getScope(t)
	if r.expires != "" {
		query.Set("X-Amz-Credential", r.config.AccessKeyID+"/"+scope)
		r.req.URL.RawQuery = query.Encode()
	}
	canonicalRequest := r.getCanonicalRequest(hashedPayload)
	stringToSign := r.getStringToSign(canonicalRequest, t)
	signingKey := r.getSigningKey(t)
	signature := r.getSignature(signingKey, stringToSign)

	if r.expires != "" {
		r.req.URL.RawQuery += "&X-Amz-Signature=" + signature
	} else {
		// final Authorization header
		parts := []string{
			authHeader + " Credential=" + r.config.AccessKeyID + "/" + scope,
			"SignedHeaders=" + signedHeaders,
			"Signature=" + signature,
		}
		auth := strings.Join(parts, ", ")
		r.Set("Authorization", auth)
	}
}
