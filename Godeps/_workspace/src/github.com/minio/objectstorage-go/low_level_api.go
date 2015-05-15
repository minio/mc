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
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type lowLevelAPI struct {
	config *Config
}

// putBucketRequest wrapper creates a new PutBucket request
func (a *lowLevelAPI) putBucketRequest(bucket, acl, location string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket,
	}
	var createBucketConfigBuffer *bytes.Reader
	// If location is set use it and create proper bucket configuration
	switch {
	case location != "":
		createBucketConfig := new(createBucketConfiguration)
		createBucketConfig.Location = location
		createBucketConfigBytes, err := xml.Marshal(createBucketConfig)
		if err != nil {
			return nil, err
		}
		createBucketConfigBuffer = bytes.NewReader(createBucketConfigBytes)
	}
	var r *request
	var err error
	switch {
	case createBucketConfigBuffer == nil:
		r, err = newRequest(op, a.config, nil)
		if err != nil {
			return nil, err
		}
	default:
		r, err = newRequest(op, a.config, createBucketConfigBuffer)
		if err != nil {
			return nil, err
		}
		r.req.ContentLength = int64(createBucketConfigBuffer.Len())
	}
	// by default bucket is private
	switch {
	case acl != "":
		r.Set("x-amz-acl", acl)
	default:
		r.Set("x-amz-acl", "private")
	}

	return r, nil
}

/// Bucket Write Operations

// putBucket create a new bucket
//
// Requires valid AWS Access Key ID to authenticate requests
// Anonymous requests are never allowed to create buckets
//
// optional arguments are acl and location - by default all buckets are created
// with ``private`` acl and location set to US Standard if one wishes to set
// different ACLs and Location one can set them properly.
//
// ACL valid values
// ------------------
// private - owner gets full access [DEFAULT]
// public-read - owner gets full access, others get read access
// public-read-write - owner gets full access, others get full access too
// ------------------
//
// Location valid values
// ------------------
// [ us-west-1 | us-west-2 | eu-west-1 | eu-central-1 | ap-southeast-1 | ap-northeast-1 | ap-southeast-2 | sa-east-1 ]
// Default - US standard
func (a *lowLevelAPI) putBucket(bucket, acl, location string) error {
	req, err := a.putBucketRequest(bucket, acl, location)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return responseToError(resp)
		}
	}
	return resp.Body.Close()
}

// putBucketRequestACL wrapper creates a new putBucketACL request
func (a *lowLevelAPI) putBucketRequestACL(bucket, acl string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "?acl",
	}
	req, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	req.Set("x-amz-acl", acl)
	return req, nil
}

// putBucketACL set the permissions on an existing bucket using access control lists (ACL)
func (a *lowLevelAPI) putBucketACL(bucket, acl string) error {
	req, err := a.putBucketRequestACL(bucket, acl)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return responseToError(resp)
		}
	}
	return resp.Body.Close()
}

// getBucketLocationRequest wrapper creates a new getBucketLocation request
func (a *lowLevelAPI) getBucketLocationRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + "?location",
	}
	req, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// getBucketLocation uses location subresource to return a bucket's region
func (a *lowLevelAPI) getBucketLocation(bucket string) (string, error) {
	req, err := a.getBucketLocationRequest(bucket)
	if err != nil {
		return "", err
	}
	resp, err := req.Do()
	if err != nil {
		return "", err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return "", responseToError(resp)
		}
	}
	var locationConstraint string
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(&locationConstraint)
	if err != nil {
		return "", err
	}
	return locationConstraint, resp.Body.Close()
}

// listObjectsRequest wrapper creates a new ListObjects request
func (a *lowLevelAPI) listObjectsRequest(bucket, marker, prefix, delimiter string, maxkeys int) (*request, error) {
	// resourceQuery - get resources properly escaped and lined up before using them in http request
	resourceQuery := func() string {
		switch {
		case marker != "":
			marker = fmt.Sprintf("&marker=%s", url.QueryEscape(marker))
			fallthrough
		case prefix != "":
			prefix = fmt.Sprintf("&prefix=%s", url.QueryEscape(prefix))
			fallthrough
		case delimiter != "":
			delimiter = fmt.Sprintf("&delimiter=%s", url.QueryEscape(delimiter))
		}
		return fmt.Sprintf("?max-keys=%d", maxkeys) + marker + prefix + delimiter
	}
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + resourceQuery(),
	}
	r, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	return r, nil
}

/// Bucket Read Operations

// listObjects - (List Objects) - List some or all (up to 1000) of the objects in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the objects in a bucket.
// request paramters :-
// ---------
// ?marker - Specifies the key to start with when listing objects in a bucket.
// ?delimiter - A delimiter is a character you use to group keys.
// ?prefix - Limits the response to keys that begin with the specified prefix.
// ?max-keys - Sets the maximum number of keys returned in the response body.
func (a *lowLevelAPI) listObjects(bucket, marker, prefix, delimiter string, maxkeys int) (*listBucketResult, error) {
	req, err := a.listObjectsRequest(bucket, marker, prefix, delimiter, maxkeys)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	listBucketResult := new(listBucketResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listBucketResult)
	if err != nil {
		return nil, err
	}

	// close body while returning, along with any error
	return listBucketResult, resp.Body.Close()
}

func (a *lowLevelAPI) headBucketRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "HEAD",
		HTTPPath:   "/" + bucket,
	}
	return newRequest(op, a.config, nil)
}

// headBucket - useful to determine if a bucket exists and you have permission to access it.
func (a *lowLevelAPI) headBucket(bucket string) error {
	req, err := a.headBucketRequest(bucket)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			// Head has no response body, handle it
			return fmt.Errorf("%s", resp.Status)
		}
	}
	return resp.Body.Close()
}

// deleteBucketRequest wrapper creates a new DeleteBucket request
func (a *lowLevelAPI) deleteBucketRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket,
	}
	return newRequest(op, a.config, nil)
}

// deleteBucket - deletes the bucket named in the URI
// NOTE: -
//  All objects (including all object versions and delete markers)
//  in the bucket must be deleted before successfully attempting this request
func (a *lowLevelAPI) deleteBucket(bucket string) error {
	req, err := a.deleteBucketRequest(bucket)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

/// Object Read/Write/Stat Operations

// putObjectRequest wrapper creates a new PutObject request
func (a *lowLevelAPI) putObjectRequest(bucket, object string, size int64, body io.ReadSeeker) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	md5SumBytes, err := sumMD5Reader(body, size)
	if err != nil {
		return nil, err
	}
	r, err := newRequest(op, a.config, body)
	if err != nil {
		return nil, err
	}
	// set Content-MD5 as base64 encoded md5
	r.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5SumBytes))
	r.req.ContentLength = size
	return r, nil
}

// putObject - add an object to a bucket
//
// You must have WRITE permissions on a bucket to add an object to it.
func (a *lowLevelAPI) putObject(bucket, object string, size int64, body io.ReadSeeker) (*ObjectMetadata, error) {
	req, err := a.putObjectRequest(bucket, object, size, body)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	metadata := new(ObjectMetadata)
	metadata.ETag = strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	return metadata, resp.Body.Close()
}

// getObjectRequest wrapper creates a new GetObject request
func (a *lowLevelAPI) getObjectRequest(bucket, object string, offset, length uint64) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	r, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	// TODO - fix this to support full - http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html
	switch {
	case length > 0:
		r.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	default:
		r.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	return r, nil
}

// getObject - retrieve object from Object Storage
//
// Additionally it also takes range arguments to download the specified range bytes of an object.
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a *lowLevelAPI) getObject(bucket, object string, offset, length uint64) (io.ReadCloser, *ObjectMetadata, error) {
	req, err := a.getObjectRequest(bucket, object, offset, length)
	if err != nil {
		return nil, nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, nil, err
	}
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusOK:
		case http.StatusPartialContent:
		default:
			return nil, nil, responseToError(resp)
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	if md5sum == "" {
		return nil, nil, errors.New("missing ETag")
	}
	date, err := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
	if err != nil {
		return nil, nil, err
	}
	objectmetadata := new(ObjectMetadata)
	objectmetadata.ETag = md5sum
	objectmetadata.Key = object
	objectmetadata.Size = resp.ContentLength
	objectmetadata.LastModified = date

	// do not close body here, caller will close
	return resp.Body, objectmetadata, nil
}

// headObjectRequest wrapper creates a new HeadObject request
func (a *lowLevelAPI) headObjectRequest(bucket, object string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "HEAD",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	return newRequest(op, a.config, nil)
}

// headObject - retrieves metadata from an object without returning the object itself
func (a *lowLevelAPI) headObject(bucket, object string) (*ObjectMetadata, error) {
	req, err := a.headObjectRequest(bucket, object)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	if md5sum == "" {
		return nil, errors.New("missing ETag")
	}
	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return nil, err
	}
	date, err := time.Parse(time.RFC1123, resp.Header.Get("Last-Modified"))
	if err != nil {
		return nil, err
	}
	objectmetadata := new(ObjectMetadata)
	objectmetadata.ETag = md5sum
	objectmetadata.Key = object
	objectmetadata.Size = size
	objectmetadata.LastModified = date
	return objectmetadata, nil
}

// deleteObjectRequest wrapper creates a new DeleteObject request
func (a *lowLevelAPI) deleteObjectRequest(bucket, object string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket + "/" + object,
	}
	return newRequest(op, a.config, nil)
}

// deleteObject removes the object
func (a *lowLevelAPI) deleteObject(bucket, object string) error {
	req, err := a.deleteObjectRequest(bucket, object)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

/// Service Operations

// listBucketRequest wrapper creates a new ListBuckets request
func (a *lowLevelAPI) listBucketsRequest() (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "GET",
		HTTPPath:   "/",
	}
	return newRequest(op, a.config, nil)
}

// listBuckets list of all buckets owned by the authenticated sender of the request
func (a *lowLevelAPI) listBuckets() (*listAllMyBucketsResult, error) {
	req, err := a.listBucketsRequest()
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	listAllMyBucketsResult := new(listAllMyBucketsResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listAllMyBucketsResult)
	if err != nil {
		return nil, err
	}
	return listAllMyBucketsResult, resp.Body.Close()
}
