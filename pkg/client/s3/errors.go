/*
 * Mini Copy, (C) 2015 Minio, Inc.
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
	"fmt"
	"net/http"

	"github.com/clbanning/mxj"
	"github.com/minio-io/minio/pkg/iodine"
)

/* **** SAMPLE ERROR RESPONSE ****
s3.ListBucket: status 403:
<?xml version="1.0" encoding="UTF-8"?>
<Error>
   <Code>AccessDenied</Code>
   <Message>Access Denied</Message>
   <Resource>/mybucket/myphoto.jpg</Resource>
   <RequestId>F19772218238A85A</RequestId>
   <HostId>GuWkjyviSiGHizehqpmsD1ndz5NClSP19DOT+s2mv7gXGQ8/X1lhbDGiIJEXpGFD</HostId>
</Error>
*/

// Error is the type returned by some API operations.
type Error struct {
	response    *http.Response // response headers
	responseMap mxj.Map        // Keys: Code, Message, Resource, RequestId, HostId
}

// NewError returns a new initialized S3.Error structure
func NewError(res *http.Response) error {
	var err error
	s3Err := new(Error)
	s3Err.response = res
	s3Err.responseMap, err = mxj.NewMapXmlReader(res.Body)
	if err != nil {
		return iodine.New(err, nil)
	}
	return s3Err
}

// Error formats HTTP error string
func (e *Error) Error() string {
	return fmt.Sprintf("%s", e.response.Status)
}
