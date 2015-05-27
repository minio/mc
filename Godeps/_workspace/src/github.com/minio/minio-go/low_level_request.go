/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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
	"encoding/hex"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
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
	req    *http.Request
	config *Config
	body   io.ReadSeeker
}

const (
	authHeader    = "AWS4-HMAC-SHA256"
	iso8601Format = "20060102T150405Z"
	yyyymmdd      = "20060102"
)

var ignoredHeaders = map[string]bool{
	"Authorization":  true,
	"Content-Type":   true,
	"Content-Length": true,
	"User-Agent":     true,
}

// newRequest - instantiate a new request
func newRequest(op *operation, config *Config, body io.ReadSeeker) (*request, error) {
	// if no method default to POST
	method := op.HTTPMethod
	if method == "" {
		method = "POST"
	}

	// parse URL for the combination of HTTPServer + HTTPPath
	u := op.HTTPServer + op.HTTPPath

	// get a new HTTP request, for the requested method
	req, err := http.NewRequest(method, u, nil)
	if err != nil {
		return nil, err
	}

	// if userAgent empty set it
	if config.userAgent == "" {
		config.userAgent = LibraryName + "/" + LibraryVersion + " (" + runtime.GOOS + ", " + runtime.GOARCH + ") "
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
		case r.body == nil:
			return hex.EncodeToString(sum256([]byte{}))
		default:
			sum256Bytes, _ := sum256Reader(r.body)
			return hex.EncodeToString(sum256Bytes)
		}
	}
	hashedPayload := hash()
	r.req.Header.Add("X-Amz-Content-Sha256", hashedPayload)
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
	encodedPath, _ := urlEncodeName(r.req.URL.Path)
	// convert any space strings back to "+"
	encodedPath = strings.Replace(encodedPath, "+", "%20", -1)
	canonicalRequest := strings.Join([]string{
		r.req.Method,
		encodedPath,
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

// SignV4 the request before Do(), in accordance with - http://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html
func (r *request) SignV4() {
	t := time.Now().UTC()
	// Add date if not present
	if date := r.Get("Date"); date == "" {
		r.Set("X-Amz-Date", t.Format(iso8601Format))
	}

	hashedPayload := r.getHashedPayload()
	signedHeaders := r.getSignedHeaders()
	canonicalRequest := r.getCanonicalRequest(hashedPayload)
	scope := r.getScope(t)
	stringToSign := r.getStringToSign(canonicalRequest, t)
	signingKey := r.getSigningKey(t)
	signature := r.getSignature(signingKey, stringToSign)

	// final Authorization header
	parts := []string{
		authHeader + " Credential=" + r.config.AccessKeyID + "/" + scope,
		"SignedHeaders=" + signedHeaders,
		"Signature=" + signature,
	}
	auth := strings.Join(parts, ", ")
	r.Set("Authorization", auth)
}
