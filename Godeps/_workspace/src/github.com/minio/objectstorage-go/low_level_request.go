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

package objectstorage

import (
	"bytes"
	"encoding/hex"
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
	req    *http.Request
	config *Config
	body   io.ReadSeeker
}

const (
	authHeaderPrefix = "AWS4-HMAC-SHA256"
	iso8601Format    = "20060102T150405Z"
	yyyymmdd         = "20060102"
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
	u, err := url.Parse(op.HTTPServer + op.HTTPPath)
	if err != nil {
		return nil, err
	}

	// get a new HTTP request, for the requested method
	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// set UserAgent
	req.Header.Set("User-Agent", config.UserAgent)

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
	client := &http.Client{}
	if r.config.Transport != nil {
		client.Transport = r.config.Transport
	}
	return client.Do(r.req)
}

// Set - set additional headers if any
func (r *request) Set(key, value string) {
	r.req.Header.Set(key, value)
}

// Get - get header values
func (r *request) Get(key string) string {
	return r.req.Header.Get(key)
}

// SignV4 the request before Do() (version 4.0)
func (r *request) SignV4() {
	t := time.Now().UTC()
	// Add date if not present
	if date := r.Get("Date"); date == "" {
		r.Set("X-Amz-Date", t.Format(iso8601Format))
	}

	hash := func() string {
		switch {
		case r.body == nil:
			return hex.EncodeToString(sum256([]byte{}))
		default:
			sum256Bytes, _ := sum256Reader(r.body)
			return hex.EncodeToString(sum256Bytes)
		}
	}
	bodyDigest := hash()
	r.req.Header.Add("X-Amz-Content-Sha256", bodyDigest)

	canonicalHeaders := func() string {
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

	signedHeaders := func() string {
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

	canonicalRequest := func() string {
		r.req.URL.RawQuery = strings.Replace(r.req.URL.Query().Encode(), "+", "%20", -1)
		canonicalRequest := strings.Join([]string{
			r.req.Method,
			r.req.URL.Path,
			r.req.URL.RawQuery,
			canonicalHeaders(),
			signedHeaders(),
			bodyDigest,
		}, "\n")
		return canonicalRequest
	}

	scope := strings.Join([]string{
		t.Format(yyyymmdd),
		r.config.Region,
		"s3",
		"aws4_request",
	}, "/")

	stringToSign := func() string {
		stringToSign := authHeaderPrefix + "\n" + t.Format(iso8601Format) + "\n"
		stringToSign = stringToSign + scope + "\n"
		stringToSign = stringToSign + hex.EncodeToString(sum256([]byte(canonicalRequest())))
		return stringToSign
	}

	signingKey := func() []byte {
		secret := r.config.SecretAccessKey
		date := sumHMAC([]byte("AWS4"+secret), []byte(t.Format(yyyymmdd)))
		region := sumHMAC(date, []byte(r.config.Region))
		service := sumHMAC(region, []byte("s3"))
		signingKey := sumHMAC(service, []byte("aws4_request"))
		return signingKey
	}

	signature := func() string {
		return hex.EncodeToString(sumHMAC(signingKey(), []byte(stringToSign())))
	}

	parts := []string{authHeaderPrefix + " Credential=" + r.config.AccessKeyID + "/" + scope,
		"SignedHeaders=" + signedHeaders(), "Signature=" + signature(),
	}
	auth := strings.Join(parts, ", ")
	r.Set("Authorization", auth)
}
