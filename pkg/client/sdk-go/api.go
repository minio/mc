/*
 * Minimalist Object Storage SDK (C) 2015 Minio, Inc.
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

import "io"

type API struct {
	*Config
}

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

type Config struct {
	*Credentials
	ForcePathStyle bool
	SSLEnabled     bool
	Region         string
	Endpoint       string
}

func New(config *Config) *API {
	return &API{config}
}

// PutBucket - create a new bucket
//
// Requires valid AWS Access Key ID to authenticate requests
// Anonymous requests are never allowed to create buckets
func (a *API) PutBucket(bucket string) error {
}

// PutBucketACL - set the permissions on an existing bucket using access control lists (ACL)
//
// Currently supported are
//    - "private"
//    - "public-read"
//    - "public-read-write"
func (a *API) PutBucketACL(bucket, acl string) error {
}

// Put - add an object to a bucket
//
// You must have WRITE permissions on a bucket to add an object to it.
func (a *API) Put(bucket, object string, body io.ReadCloser) error {
}

// Get - retrieve object from Object Storage
//
// Additionally it also takes range arguments to download the specified range bytes of an object.
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a *API) Get(bucket, object string, offset, length int64) error {
}

// GetBucket - (List Objects) - List some or all (up to 1000) of the objects in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the objects in a bucket.
// request paramters :-
// ---------
// ?delimiter - A delimiter is a character you use to group keys.
// ?marker - Specifies the key to start with when listing objects in a bucket.
// ?max-keys - Sets the maximum number of keys returned in the response body.
// ?prefix - Limits the response to keys that begin with the specified prefix.
func (a *API) GetBucket(bucket string) error {
}

// GetService - (List Buckets) - list of all buckets owned by the authenticated sender of the request
func (a *API) GetService() error {

}
