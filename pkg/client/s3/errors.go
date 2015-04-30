/*
 * Minio Client (C) 2015 Minio, Inc.
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
	"strconv"

	"github.com/clbanning/mxj"
	"github.com/minio-io/minio/pkg/iodine"
)

/// ACL related errors

// InvalidACL - acl invalid
type InvalidACL struct {
	ACL string
}

func (e InvalidACL) Error() string {
	return "Requested ACL is " + e.ACL + " invalid"
}

// InvalidMaxKeys - invalid maxkeys provided
type InvalidMaxKeys struct {
	MaxKeys int
}

func (e InvalidMaxKeys) Error() string {
	return "invalid maxkeys: " + strconv.Itoa(e.MaxKeys)
}

// GenericBucketError - generic bucket operations error
type GenericBucketError struct {
	Bucket string
}

// BucketExists - bucket exists
type BucketExists GenericBucketError

func (e BucketExists) Error() string {
	return "bucket " + e.Bucket + " exists"
}

// InvalidBucketName - bucket name invalid (http://goo.gl/wJlzDz)
type InvalidBucketName GenericBucketError

func (e InvalidBucketName) Error() string {
	return "Invalid bucketname [" + e.Bucket + "], please read http://goo.gl/wJlzDz"
}

// GenericObjectError - generic object operations error
type GenericObjectError struct {
	Bucket string
	Object string
}

// ObjectNotFound - object requested does not exist
type ObjectNotFound GenericObjectError

func (e ObjectNotFound) Error() string {
	return "object " + e.Object + " not found in bucket " + e.Bucket
}

// InvalidObjectName - object requested is invalid
type InvalidObjectName GenericObjectError

func (e InvalidObjectName) Error() string {
	return "object " + e.Object + "at" + e.Bucket + "is invalid"
}

// ObjectExists - object exists
type ObjectExists GenericObjectError

func (e ObjectExists) Error() string {
	return "object " + e.Object + " exists"
}

// GenericError - generic error
type GenericError struct {
	Err error
}

// InvalidAuthorizationKey - invalid authorization key
type InvalidAuthorizationKey GenericError

func (e InvalidAuthorizationKey) Error() string {
	return e.Err.Error()
}

// InvalidQueryURL - generic error
type InvalidQueryURL struct {
	URL string
}

func (e InvalidQueryURL) Error() string {
	return "Invalid query URL: " + e.URL
}

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
	message, _ := e.responseMap.ValuesForKey("Message")
	return fmt.Sprintf("%s", message[0])
}
