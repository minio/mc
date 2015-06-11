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

package minio

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"strings"
	"unicode/utf8"
)

/* **** SAMPLE ERROR RESPONSE ****
<?xml version="1.0" encoding="UTF-8"?>
<Error>
   <Code>AccessDenied</Code>
   <Message>Access Denied</Message>
   <Resource>/mybucket/myphoto.jpg</Resource>
   <RequestId>F19772218238A85A</RequestId>
   <HostId>GuWkjyviSiGHizehqpmsD1ndz5NClSP19DOT+s2mv7gXGQ8/X1lhbDGiIJEXpGFD</HostId>
</Error>
*/

// ErrorResponse is the type error returned by some API operations.
type ErrorResponse struct {
	XMLName   xml.Name `xml:"Error" json:"-"`
	Code      string
	Message   string
	Resource  string
	RequestID string `xml:"RequestId"`
	HostID    string `xml:"HostId"`
}

// ToErrorResponse returns parsed ErrorResponse struct, if input is nil or not ErrorResponse return value is nil
// this fuction is useful when some one wants to dig deeper into the error structures over the network.
//
// for example:
//
//   import s3 "github.com/minio/minio-go"
//   ...
//   ...
//   ..., err := s3.GetObject(...)
//   if err != nil {
//      resp := s3.ToErrorResponse(err)
//      fmt.Println(resp.ToXML())
//   }
//   ...
//   ...
func ToErrorResponse(err error) *ErrorResponse {
	switch err := err.(type) {
	case ErrorResponse:
		return &err
	default:
		return nil
	}
}

// XML send raw xml marshalled as string
func (e ErrorResponse) ToXML() string {
	b, _ := xml.Marshal(&e)
	return string(b)
}

// JSON send raw json marshalled as string
func (e ErrorResponse) ToJSON() string {
	b, _ := json.Marshal(&e)
	return string(b)
}

// Error formats HTTP error string
func (e ErrorResponse) Error() string {
	return e.Message
}

/// Internal function not exposed

// responseToError returns a new encoded ErrorResponse structure
func (a lowLevelAPI) responseToError(errBody io.Reader) error {
	var respError ErrorResponse
	err := acceptTypeDecoder(errBody, a.config.AcceptType, &respError)
	if err != nil {
		return err
	}
	return respError
}

func invalidBucketToError(bucket string) error {
	if strings.TrimSpace(bucket) == "" || bucket == "" {
		// no resource since bucket is empty string
		errorResponse := ErrorResponse{
			Code:      "InvalidBucketName",
			Message:   "The specified bucket is not valid.",
			RequestID: "minio",
		}
		return errorResponse
	}
	return nil
}

func invalidArgumentToError(arg string) error {
	errorResponse := ErrorResponse{
		Code:      "InvalidArgument",
		Message:   "Invalid Argument",
		RequestID: "minio",
	}
	if strings.TrimSpace(arg) == "" || arg == "" {
		// no resource since arg is empty string
		return errorResponse
	}
	if !utf8.ValidString(arg) {
		// add resource to reply back with invalid string
		errorResponse.Resource = arg
		return errorResponse
	}
	return nil
}
