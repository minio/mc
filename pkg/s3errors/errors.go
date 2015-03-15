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

//Package s3errors implements client side error handling for S3 protocol
package s3errors

import (
	"fmt"

	"io/ioutil"
	"net/http"

	"github.com/clbanning/x2j"
)

/* **** SAMPLE ERROR RESPONSE ****
s3.ListBucket: status 403:
<?xml version="1.0" encoding="UTF-8"?>
<Error>
	<Code>AccessDenied</Code>
	<Message>Access Denied</Message>
	<RequestId>F19772218238A85A</RequestId>
	<HostId>GuWkjyviSiGHizehqpmsD1ndz5NClSP19DOT+s2mv7gXGQ8/X1lhbDGiIJEXpGFD</HostId>
</Error>
*/

// Error is the type returned by some API operations.
type Error struct {
	res    *http.Response         // response headers
	resMsg map[string]interface{} //Keys: Status, Message, RequestID, Bucket, Endpoint
}

// New returns a new initialized S3.Error structure
func New(res *http.Response) error {
	var s3Err Error
	s3Err.res = res

	xmlBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	s3Err.resMsg, err = x2j.DocToMap(string(xmlBody))
	if err != nil {
		return err
	}
	return s3Err
}

// Error formats HTTP error string
func (e Error) Error() string {
	/*
		if bytes.Contains(e.Body, []byte("<Error>")) {
			return fmt.Sprintf("s3.%s: status %d: %s", e.Op, e.Code, e.Body)
		}
		return fmt.Sprintf("%s: status %d", e.Op, e.Code)
	*/
	return fmt.Sprintf("%s", e.res.Status)
}
