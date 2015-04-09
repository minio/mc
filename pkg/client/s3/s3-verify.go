/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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
	"errors"
	"net/http"
	"strings"

	"github.com/minio-io/minio/pkg/iodine"
)

// Only generic HTTP S3 fields are validated using this mechanism. S3
// command specific checks should be added to their own appropriate
// files.
type s3Verify struct {
}

/*
Example AWS S3 Request / Response
=================================
  $ mc --debug ls https://zek.s3.amazonaws.com
  GET /?max-keys=1000 HTTP/1.1
  Host: zek.s3.amazonaws.com
  User-Agent: Minio Client
  Authorization: AWS **PASSWORD**STRIPPED**
  Date: Thu, 12 Mar 2015 23:04:23 GMT
  Accept-Encoding: gzip

  HTTP/1.1 200 OK
  Transfer-Encoding: chunked
  Content-Type: application/xml
  Date: Thu, 12 Mar 2015 23:04:24 GMT
  Server: AmazonS3
  X-Amz-Id-2: NFc3FTxx4dD+z348qOv532aIaq4VufRnZEnq2kGOloTKhHYhBBZ6qe8mcmZcR+L3Us4N5j0WNtk=
  X-Amz-Request-Id: 5E9A737908F886D0

  2015-03-11 01:28:03 -0700 PDT    1.10 KB color.go
  2015-03-08 03:31:50 -0700 PDT   11.09 KB apache/license.txt
*/

// HTTP S3 request validator.
func (t s3Verify) Request(req *http.Request) error {
	if req.Header.Get("Authorization") == "" {
		return iodine.New(errors.New("Client request header has authorization key"), nil)
	}
	if !strings.HasPrefix(req.Header.Get("Authorization"), "AWS") {
		return iodine.New(errors.New("Client request header has malformed authorization key"), nil)
	}
	return nil
}

// HTTP S3 response validator.
func (t s3Verify) Response(res *http.Response) error {
	return nil
}
