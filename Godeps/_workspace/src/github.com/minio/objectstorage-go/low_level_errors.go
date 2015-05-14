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
	"encoding/xml"
	"net/http"
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

// errorResponse is the type returned by some API operations.
type errorResponse struct {
	Code      string
	Message   string
	Resource  string
	RequestID string
	HostID    string
}

// responseToError returns a new encoded ErrorResponse structure
func responseToError(res *http.Response) error {
	var respError errorResponse
	decoder := xml.NewDecoder(res.Body)
	err := decoder.Decode(&respError)
	if err != nil {
		return err
	}
	return respError
}

// Error formats HTTP error string
func (e errorResponse) Error() string {
	return e.Message
}
