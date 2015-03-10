// Original license //
// ---------------- //

/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// All other modifications and improvements //
// ---------------------------------------- //

/*
 * Mini Object Storage, (C) 2015 Minio, Inc.
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
	"strings"

	"encoding/hex"
	"encoding/xml"
	"net/http"
)

// Error is the type returned by some API operations.
type Error struct {
	Op     string
	Code   int         // HTTP status code
	Body   []byte      // response body
	Header http.Header // response headers

	// UsedEndpoint and AmazonCode are the XML response's Endpoint and
	// Code fields, respectively.
	UseEndpoint string // if a temporary redirect (wrong endpoint)
	AmazonCode  string
}

// xmlError is the Error response from Amazon.
type xmlError struct {
	XMLName           xml.Name `xml:"Error"`
	Code              string
	Message           string
	RequestID         string
	Bucket            string
	Endpoint          string
	StringToSignBytes string
}

func (e *Error) Error() string {
	if bytes.Contains(e.Body, []byte("<Error>")) {
		return fmt.Sprintf("s3.%s: status %d: %s", e.Op, e.Code, e.Body)
	}
	return fmt.Sprintf("s3.%s: status %d", e.Op, e.Code)
}

func (e *Error) parseXML() {
	var xe xmlError
	_ = xml.NewDecoder(bytes.NewReader(e.Body)).Decode(&xe)
	e.AmazonCode = xe.Code
	if xe.Code == "TemporaryRedirect" {
		e.UseEndpoint = xe.Endpoint
	}
	if xe.Code == "SignatureDoesNotMatch" {
		want, _ := hex.DecodeString(strings.Replace(xe.StringToSignBytes, " ", "", -1))
		fmt.Printf("S3 SignatureDoesNotMatch. StringToSign should be %d bytes: %q (%x)", len(want), want, want)
	}

}
