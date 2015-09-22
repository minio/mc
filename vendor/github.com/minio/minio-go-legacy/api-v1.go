/*
 * Minio Go Library for Amazon S3 Legacy v2 Signature Compatible Cloud Storage (C) 2015 Minio, Inc.
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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	separator = "/"
)

// apiV1 container to hold unexported internal functions
type apiV1 struct {
	config *Config
}

// closeResp close non nil response with any response Body
func closeResp(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// putBucketRequest wrapper creates a new putBucket request
func (a apiV1) putBucketRequest(bucket, acl, location string) (*request, error) {
	var r *request
	var err error
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   separator + bucket,
	}
	var createBucketConfigBuffer *bytes.Reader
	// If location is set use it and create proper bucket configuration
	switch {
	case location != "":
		createBucketConfig := new(createBucketConfiguration)
		createBucketConfig.Location = location
		var createBucketConfigBytes []byte
		switch {
		case a.config.AcceptType == "application/xml":
			createBucketConfigBytes, err = xml.Marshal(createBucketConfig)
		case a.config.AcceptType == "application/json":
			createBucketConfigBytes, err = json.Marshal(createBucketConfig)
		default:
			createBucketConfigBytes, err = xml.Marshal(createBucketConfig)
		}
		if err != nil {
			return nil, err
		}
		createBucketConfigBuffer = bytes.NewReader(createBucketConfigBytes)
	}
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
// authenticated-read - owner gets full access, authenticated users get read access
// ------------------
//
// Location valid values
// ------------------
// [ us-west-1 | us-west-2 | eu-west-1 | eu-central-1 | ap-southeast-1 | ap-northeast-1 | ap-southeast-2 | sa-east-1 ]
//
// Default - US standard
func (a apiV1) putBucket(bucket, acl, location string) error {
	req, err := a.putBucketRequest(bucket, acl, location)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	return nil
}

// putBucketRequestACL wrapper creates a new putBucketACL request
func (a apiV1) putBucketACLRequest(bucket, acl string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   separator + bucket + "?acl",
	}
	req, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	req.Set("x-amz-acl", acl)
	return req, nil
}

// putBucketACL set the permissions on an existing bucket using Canned ACL's
func (a apiV1) putBucketACL(bucket, acl string) error {
	req, err := a.putBucketACLRequest(bucket, acl)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	return nil
}

// getBucketACLRequest wrapper creates a new getBucketACL request
func (a apiV1) getBucketACLRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + "?acl",
	}
	req, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// getBucketACL get the acl information on an existing bucket
func (a apiV1) getBucketACL(bucket string) (accessControlPolicy, error) {
	req, err := a.getBucketACLRequest(bucket)
	if err != nil {
		return accessControlPolicy{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return accessControlPolicy{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return accessControlPolicy{}, BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	policy := accessControlPolicy{}
	err = acceptTypeDecoder(resp.Body, a.config.AcceptType, &policy)
	if err != nil {
		return accessControlPolicy{}, err
	}
	if policy.AccessControlList.Grant == nil {
		errorResponse := ErrorResponse{
			Code:      "InternalError",
			Message:   "Access control Grant list is empty, please report this at https://github.com/minio/minio-go-legacy/issues",
			Resource:  separator + bucket,
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
		}
		return accessControlPolicy{}, errorResponse
	}
	return policy, nil
}

// getBucketLocationRequest wrapper creates a new getBucketLocation request
func (a apiV1) getBucketLocationRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + "?location",
	}
	req, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// getBucketLocation uses location subresource to return a bucket's region
func (a apiV1) getBucketLocation(bucket string) (string, error) {
	req, err := a.getBucketLocationRequest(bucket)
	if err != nil {
		return "", err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return "", err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return "", BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	var locationConstraint string
	err = acceptTypeDecoder(resp.Body, a.config.AcceptType, &locationConstraint)
	if err != nil {
		return "", err
	}
	return locationConstraint, nil
}

// listObjectsRequest wrapper creates a new listObjects request
func (a apiV1) listObjectsRequest(bucket, marker, prefix, delimiter string, maxkeys int) (*request, error) {
	// resourceQuery - get resources properly escaped and lined up before using them in http request
	resourceQuery := func() (*string, error) {
		var err error
		switch {
		case marker != "":
			marker, err = urlEncodeName(marker)
			if err != nil {
				return nil, err
			}
			marker = fmt.Sprintf("&marker=%s", marker)
			fallthrough
		case prefix != "":
			prefix, err = urlEncodeName(prefix)
			if err != nil {
				return nil, err
			}
			prefix = fmt.Sprintf("&prefix=%s", prefix)
			fallthrough
		case delimiter != "":
			delimiter, err = urlEncodeName(delimiter)
			if err != nil {
				return nil, err
			}
			delimiter = fmt.Sprintf("&delimiter=%s", delimiter)
		}
		query := fmt.Sprintf("?max-keys=%d", maxkeys) + marker + prefix + delimiter
		return &query, nil
	}
	query, err := resourceQuery()
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + *query,
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
func (a apiV1) listObjects(bucket, marker, prefix, delimiter string, maxkeys int) (listBucketResult, error) {
	if err := invalidBucketError(bucket); err != nil {
		return listBucketResult{}, err
	}
	req, err := a.listObjectsRequest(bucket, marker, prefix, delimiter, maxkeys)
	if err != nil {
		return listBucketResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return listBucketResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return listBucketResult{}, BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	listBucketResult := listBucketResult{}
	err = acceptTypeDecoder(resp.Body, a.config.AcceptType, &listBucketResult)
	if err != nil {
		return listBucketResult, err
	}
	// close body while returning, along with any error
	return listBucketResult, nil
}

// headBucketRequest wrapper creates a new headBucket request
func (a apiV1) headBucketRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "HEAD",
		HTTPPath:   separator + bucket,
	}
	return newRequest(op, a.config, nil)
}

// headBucket useful to determine if a bucket exists and you have permission to access it.
func (a apiV1) headBucket(bucket string) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	req, err := a.headBucketRequest(bucket)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			// Head has no response body, handle it
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:      "NoSuchBucket",
					Message:   "The specified bucket does not exist.",
					Resource:  separator + bucket,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:      "AccessDenied",
					Message:   "Access Denied",
					Resource:  separator + bucket,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:      resp.Status,
					Message:   resp.Status,
					Resource:  separator + bucket,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			}
			return errorResponse
		}
	}
	return nil
}

// deleteBucketRequest wrapper creates a new deleteBucket request
func (a apiV1) deleteBucketRequest(bucket string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   separator + bucket,
	}
	return newRequest(op, a.config, nil)
}

// deleteBucket deletes the bucket named in the URI
//
// NOTE: -
//  All objects (including all object versions and delete markers)
//  in the bucket must be deleted before successfully attempting this request
func (a apiV1) deleteBucket(bucket string) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	req, err := a.deleteBucketRequest(bucket)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:      "NoSuchBucket",
					Message:   "The specified bucket does not exist.",
					Resource:  separator + bucket,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:      "AccessDenied",
					Message:   "Access Denied",
					Resource:  separator + bucket,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:      resp.Status,
					Message:   resp.Status,
					Resource:  separator + bucket,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			}
			return errorResponse
		}
	}
	return nil
}

/// Object Read/Write/Stat Operations

func (a apiV1) putObjectUnAuthenticatedRequest(bucket, object, contentType string, size int64, body io.Reader) (*request, error) {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   separator + bucket + separator + encodedObject,
	}
	r, err := newUnauthenticatedRequest(op, a.config, body)
	if err != nil {
		return nil, err
	}
	// Content-MD5 is not set consciously
	r.Set("Content-Type", contentType)
	r.req.ContentLength = size
	return r, nil
}

// putObjectUnAuthenticated - add an object to a bucket
// NOTE: You must have WRITE permissions on a bucket to add an object to it.
func (a apiV1) putObjectUnAuthenticated(bucket, object, contentType string, size int64, body io.Reader) (ObjectStat, error) {
	req, err := a.putObjectUnAuthenticatedRequest(bucket, object, contentType, size, body)
	if err != nil {
		return ObjectStat{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return ObjectStat{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ObjectStat{}, BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	var metadata ObjectStat
	metadata.ETag = strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	if metadata.ETag == "" {
		return ObjectStat{}, ErrorResponse{
			Code:      "InternalError",
			Message:   "Missing Etag, please report this issue at https://github.com/minio/minio-go-legacy/issues",
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
		}
	}
	return metadata, nil
}

// putObjectRequest wrapper creates a new PutObject request
func (a apiV1) putObjectRequest(bucket, object, contentType string, md5SumBytes []byte, size int64, body io.ReadSeeker) (*request, error) {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   separator + bucket + separator + encodedObject,
	}
	r, err := newRequest(op, a.config, body)
	if err != nil {
		return nil, err
	}
	// set Content-MD5 as base64 encoded md5
	r.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5SumBytes))
	r.Set("Content-Type", contentType)
	r.req.ContentLength = size
	return r, nil
}

// putObject - add an object to a bucket
// NOTE: You must have WRITE permissions on a bucket to add an object to it.
func (a apiV1) putObject(bucket, object, contentType string, md5SumBytes []byte, size int64, body io.ReadSeeker) (ObjectStat, error) {
	req, err := a.putObjectRequest(bucket, object, contentType, md5SumBytes, size, body)
	if err != nil {
		return ObjectStat{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return ObjectStat{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return ObjectStat{}, BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	var metadata ObjectStat
	metadata.ETag = strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	if metadata.ETag == "" {
		return ObjectStat{}, ErrorResponse{
			Code:      "InternalError",
			Message:   "Missing Etag, please report this issue at https://github.com/minio/minio-go-legacy/issues",
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
		}
	}
	return metadata, nil
}

func (a apiV1) presignedGetObjectRequest(bucket, object string, expires, offset, length int64) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + separator + encodedObject,
	}
	r, err := newPresignedRequest(op, a.config, expires)
	if err != nil {
		return nil, err
	}
	switch {
	case length > 0 && offset > 0:
		r.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	case offset > 0 && length == 0:
		r.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	case length > 0 && offset == 0:
		r.Set("Range", fmt.Sprintf("bytes=-%d", length))
	}
	return r, nil
}

func (a apiV1) presignedGetObject(bucket, object string, expires, offset, length int64) (string, error) {
	if err := invalidArgumentError(object); err != nil {
		return "", err
	}
	req, err := a.presignedGetObjectRequest(bucket, object, expires, offset, length)
	if err != nil {
		return "", err
	}
	return req.PreSignV2()
}

// getObjectRequest wrapper creates a new getObject request
func (a apiV1) getObjectRequest(bucket, object string, offset, length int64) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + separator + encodedObject,
	}
	r, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	switch {
	case length > 0 && offset > 0:
		r.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	case offset > 0 && length == 0:
		r.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	case length > 0 && offset == 0:
		r.Set("Range", fmt.Sprintf("bytes=-%d", length))
	}
	return r, nil
}

// getObject - retrieve object from Object Storage
//
// Additionally this function also takes range arguments to download the specified
// range bytes of an object. Setting offset and length = 0 will download the full object.
//
// For more information about the HTTP Range header, go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a apiV1) getObject(bucket, object string, offset, length int64) (io.ReadCloser, ObjectStat, error) {
	if err := invalidArgumentError(object); err != nil {
		return nil, ObjectStat{}, err
	}
	req, err := a.getObjectRequest(bucket, object, offset, length)
	if err != nil {
		return nil, ObjectStat{}, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, ObjectStat{}, err
	}
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusOK:
		case http.StatusPartialContent:
		default:
			return nil, ObjectStat{}, BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	if md5sum == "" {
		return nil, ObjectStat{}, ErrorResponse{
			Code:      "InternalError",
			Message:   "Missing Etag, please report this issue at https://github.com/minio/minio-go-legacy/issues",
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
		}
	}
	date, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return nil, ObjectStat{}, ErrorResponse{
			Code:      "InternalError",
			Message:   "Content-Length not recognized, please report this issue at https://github.com/minio/minio-go-legacy/issues",
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
		}
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	var objectstat ObjectStat
	objectstat.ETag = md5sum
	objectstat.Key = object
	objectstat.Size = resp.ContentLength
	objectstat.LastModified = date
	objectstat.ContentType = contentType

	// do not close body here, caller will close
	return resp.Body, objectstat, nil
}

// deleteObjectRequest wrapper creates a new deleteObject request
func (a apiV1) deleteObjectRequest(bucket, object string) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   separator + bucket + separator + encodedObject,
	}
	return newRequest(op, a.config, nil)
}

// deleteObject deletes a given object from a bucket
func (a apiV1) deleteObject(bucket, object string) error {
	if err := invalidBucketError(bucket); err != nil {
		return err
	}
	if err := invalidArgumentError(object); err != nil {
		return err
	}
	req, err := a.deleteObjectRequest(bucket, object)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:      "NoSuchKey",
					Message:   "The specified key does not exist.",
					Resource:  separator + bucket + separator + object,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:      "AccessDenied",
					Message:   "Access Denied",
					Resource:  separator + bucket + separator + object,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:      resp.Status,
					Message:   resp.Status,
					Resource:  separator + bucket + separator + object,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			}
			return errorResponse
		}
	}
	return nil
}

// headObjectRequest wrapper creates a new headObject request
func (a apiV1) headObjectRequest(bucket, object string) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "HEAD",
		HTTPPath:   separator + bucket + separator + encodedObject,
	}
	return newRequest(op, a.config, nil)
}

// headObject retrieves metadata from an object without returning the object itself
func (a apiV1) headObject(bucket, object string) (ObjectStat, error) {
	if err := invalidBucketError(bucket); err != nil {
		return ObjectStat{}, err
	}
	if err := invalidArgumentError(object); err != nil {
		return ObjectStat{}, err
	}
	req, err := a.headObjectRequest(bucket, object)
	if err != nil {
		return ObjectStat{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return ObjectStat{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:      "NoSuchKey",
					Message:   "The specified key does not exist.",
					Resource:  separator + bucket + separator + object,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:      "AccessDenied",
					Message:   "Access Denied",
					Resource:  separator + bucket + separator + object,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:      resp.Status,
					Message:   resp.Status,
					Resource:  separator + bucket + separator + object,
					RequestID: resp.Header.Get("x-amz-request-id"),
					HostID:    resp.Header.Get("x-amz-id-2"),
				}

			}
			return ObjectStat{}, errorResponse
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	if md5sum == "" {
		return ObjectStat{}, ErrorResponse{
			Code:      "InternalError",
			Message:   "Missing Etag, please report this issue at https://github.com/minio/minio-go-legacy/issues",
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
		}
	}
	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return ObjectStat{}, ErrorResponse{
			Code:      "InternalError",
			Message:   "Content-Length not recognized, please report this issue at https://github.com/minio/minio-go-legacy/issues",
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
		}
	}
	date, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return ObjectStat{}, ErrorResponse{
			Code:      "InternalError",
			Message:   "Last-Modified time format not recognized, please report this issue at https://github.com/minio/minio-go-legacy/issues",
			RequestID: resp.Header.Get("x-amz-request-id"),
			HostID:    resp.Header.Get("x-amz-id-2"),
		}
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	var objectstat ObjectStat
	objectstat.ETag = md5sum
	objectstat.Key = object
	objectstat.Size = size
	objectstat.LastModified = date
	objectstat.ContentType = contentType
	return objectstat, nil
}

/// Service Operations

// listBucketRequest wrapper creates a new listBuckets request
func (a apiV1) listBucketsRequest() (*request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator,
	}
	return newRequest(op, a.config, nil)
}

// listBuckets list of all buckets owned by the authenticated sender of the request
func (a apiV1) listBuckets() (listAllMyBucketsResult, error) {
	req, err := a.listBucketsRequest()
	if err != nil {
		return listAllMyBucketsResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return listAllMyBucketsResult{}, err
	}
	if resp != nil {
		// for un-authenticated requests, amazon sends a redirect handle it
		if resp.StatusCode == http.StatusTemporaryRedirect {
			return listAllMyBucketsResult{}, ErrorResponse{
				Code:      "AccessDenied",
				Message:   "Anonymous access is forbidden for this operation",
				RequestID: resp.Header.Get("x-amz-request-id"),
				HostID:    resp.Header.Get("x-amz-id-2"),
			}
		}
		if resp.StatusCode != http.StatusOK {
			return listAllMyBucketsResult{}, BodyToErrorResponse(resp.Body, a.config.AcceptType)
		}
	}
	listAllMyBucketsResult := listAllMyBucketsResult{}
	err = acceptTypeDecoder(resp.Body, a.config.AcceptType, &listAllMyBucketsResult)
	if err != nil {
		return listAllMyBucketsResult, err
	}
	return listAllMyBucketsResult, nil
}
