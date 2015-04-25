/*
 * Mini Copy (C) 2015 Minio, Inc.
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

package s3

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
)

func (a *s3Client) loadKeys(cert string, key string) (*TLSConfig, error) {
	certBlock, err := ioutil.ReadFile(cert)
	if err != nil {
		return nil, err
	}
	keyBlock, err := ioutil.ReadFile(key)
	if err != nil {
		return nil, err
	}
	t := &TLSConfig{}
	t.CertPEMBlock = certBlock
	t.KeyPEMBlock = keyBlock
	return t, nil
}

func (a *s3Client) getTLSTransport() (*http.Transport, error) {
	if a.CertPEM == "" || a.KeyPEM == "" {
		return &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
		}, nil
	}

	tlsconfig, err := a.loadKeys(a.CertPEM, a.KeyPEM)
	if err != nil {
		return nil, err
	}
	var cert tls.Certificate
	cert, err = tls.X509KeyPair(tlsconfig.CertPEMBlock, tlsconfig.KeyPEMBlock)
	if err != nil {
		return nil, err
	}

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	return transport, nil
}

func (a *s3Client) signRequest(req *http.Request, host string) {
	if date := req.Header.Get("Date"); date == "" {
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}
	hm := hmac.New(sha1.New, []byte(a.SecretAccessKey))
	ss := a.stringToSign(req, host)
	//fmt.Printf("String to sign: %q (%x)\n", ss, ss)
	io.WriteString(hm, ss)

	authHeader := new(bytes.Buffer)
	fmt.Fprintf(authHeader, "AWS %s:", a.AccessKeyID)
	encoder := base64.NewEncoder(base64.StdEncoding, authHeader)
	encoder.Write(hm.Sum(nil))
	encoder.Close()
	req.Header.Set("Authorization", authHeader.String())
}

// From the Amazon docs:
//
// StringToSign = HTTP-Verb + "\n" +
// 	 Content-MD5 + "\n" +
//	 Content-Type + "\n" +
//	 Date + "\n" +
//	 CanonicalizedAmzHeaders +
//	 CanonicalizedResource;
func (a *s3Client) stringToSign(req *http.Request, host string) string {
	buf := new(bytes.Buffer)
	buf.WriteString(req.Method)
	buf.WriteByte('\n')
	buf.WriteString(req.Header.Get("Content-MD5"))
	buf.WriteByte('\n')
	buf.WriteString(req.Header.Get("Content-Type"))
	buf.WriteByte('\n')
	if req.Header.Get("x-amz-date") == "" {
		buf.WriteString(req.Header.Get("Date"))
	}
	buf.WriteByte('\n')
	a.writeCanonicalizedAmzHeaders(buf, req)
	a.writeCanonicalizedResource(buf, req, host)
	return buf.String()
}

func hasPrefixCaseInsensitive(s, pfx string) bool {
	if len(pfx) > len(s) {
		return false
	}
	shead := s[:len(pfx)]
	if shead == pfx {
		return true
	}
	shead = strings.ToLower(shead)
	return shead == pfx || shead == strings.ToLower(pfx)
}

func (a *s3Client) writeCanonicalizedAmzHeaders(buf *bytes.Buffer, req *http.Request) {
	var amzHeaders []string
	vals := make(map[string][]string)
	for k, vv := range req.Header {
		if hasPrefixCaseInsensitive(k, "x-amz-") {
			lk := strings.ToLower(k)
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
var subResList = []string{
	"acl",
	"location",
	"logging",
	"notification",
	"partNumber",
	"policy",
	"requestPayment",
	"torrent",
	"uploadId",
	"uploads",
	"versionId",
	"versioning",
	"versions",
	"response-content-type",
	"response-content-language",
	"response-expires",
	"response-cache-control",
	"response-content-disposition",
	"response-content-encoding",
	"website",
}

// From the Amazon docs:
//
// CanonicalizedResource = [ "/" + Bucket ] +
// 	  <HTTP-Request-URI, from the protocol name up to the query string> +
// 	  [ sub-resource, if present. For example "?acl", "?location", "?logging", or "?torrent"];
func (a *s3Client) writeCanonicalizedResource(buf *bytes.Buffer, req *http.Request, host string) {
	bucket := a.bucketFromHost(req, host)
	if bucket != "" {
		buf.WriteByte('/')
		buf.WriteString(bucket)
	}
	buf.WriteString(req.URL.Path)
	sort.Strings(subResList)
	if req.URL.RawQuery != "" {
		n := 0
		vals, _ := url.ParseQuery(req.URL.RawQuery)
		for _, subres := range subResList {
			if vv, ok := vals[subres]; ok && len(vv) > 0 {
				n++
				if n == 1 {
					buf.WriteByte('?')
				} else {
					buf.WriteByte('&')
				}
				buf.WriteString(subres)
				if len(vv[0]) > 0 {
					buf.WriteByte('=')
					buf.WriteString(url.QueryEscape(vv[0]))
				}
			}
		}
	}
}

// hasDotSuffix reports whether s ends with "." + suffix.
func hasDotSuffix(s string, suffix string) bool {
	return len(s) >= len(suffix)+1 && strings.HasSuffix(s, suffix) && s[len(s)-len(suffix)-1] == '.'
}

func (a *s3Client) bucketFromHost(req *http.Request, host string) string {
	reqHost := req.Host
	if reqHost == "" {
		host = req.URL.Host
	}

	if reqHost == strings.TrimPrefix(host, "http://") {
		return ""
	}

	if reqHost == strings.TrimPrefix(host, "https://") {
		return ""
	}

	if reqHostSuffix := strings.TrimPrefix(host, "https://"); hasDotSuffix(reqHost, reqHostSuffix) {
		return reqHost[:len(reqHost)-len(reqHostSuffix)-1]
	}

	reqHost, _, _ = net.SplitHostPort(reqHost)
	return reqHost
}
