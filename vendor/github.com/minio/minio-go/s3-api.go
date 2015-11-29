/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	separator = "/"
)

// s3API container to hold unexported internal functions.
type s3API struct {
	config *Config
}

// closeResp close non nil response with any response Body.
func closeResp(resp *http.Response) {
	// Callers should close resp.Body when done reading from it.
	// If resp.Body is not closed, the Client's underlying RoundTripper
	// (typically Transport) may not be able to re-use a persistent TCP
	// connection to the server for a subsequent "keep-alive" request.
	if resp != nil && resp.Body != nil {
		// Drain any remaining Body and then close the connection.
		// Without this closing connection would disallow re-using
		// the same connection for future uses.
		//  - http://stackoverflow.com/a/17961593/4465767
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
}

// putBucketRequest wrapper creates a new putBucket request.
func (a s3API) putBucketRequest(bucket, acl, location string) (*Request, error) {
	var r *Request
	var err error
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   separator + bucket,
	}
	var createBucketConfigBuffer *bytes.Buffer
	// If location is set use it and create proper bucket configuration.
	switch {
	case location != "":
		createBucketConfig := new(createBucketConfiguration)
		createBucketConfig.Location = location
		createBucketConfigBytes, err := xml.Marshal(createBucketConfig)
		if err != nil {
			return nil, err
		}
		createBucketConfigBuffer = bytes.NewBuffer(createBucketConfigBytes)
	}
	switch {
	case createBucketConfigBuffer == nil:
		r, err = newRequest(op, a.config, requestMetadata{})
		if err != nil {
			return nil, err
		}
	default:
		rmetadata := requestMetadata{
			body:               ioutil.NopCloser(createBucketConfigBuffer),
			contentLength:      int64(createBucketConfigBuffer.Len()),
			sha256PayloadBytes: sum256(createBucketConfigBuffer.Bytes()),
		}
		r, err = newRequest(op, a.config, rmetadata)
		if err != nil {
			return nil, err
		}
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

// putBucket create a new bucket.
//
// Requires valid AWS Access Key ID to authenticate requests.
// Anonymous requests are never allowed to create buckets.
//
// optional arguments are acl and location - by default all buckets are created
// with ``private`` acl and location set to US Standard if one wishes to set
// different ACLs and Location one can set them properly.
//
// ACL valid values
// ------------------
// private - owner gets full access [DEFAULT].
// public-read - owner gets full access, others get read access.
// public-read-write - owner gets full access, others get full access too.
// authenticated-read - owner gets full access, authenticated users get read access.
// ------------------
//
// Location valid values.
// ------------------
// [ us-west-1 | us-west-2 | eu-west-1 | eu-central-1 | ap-southeast-1 | ap-northeast-1 | ap-southeast-2 | sa-east-1 ]
// Default - US standard
func (a s3API) putBucket(bucket, acl, location string) error {
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
			return BodyToErrorResponse(resp.Body)
		}
	}
	return nil
}

// putBucketRequestACL wrapper creates a new putBucketACL request.
func (a s3API) putBucketACLRequest(bucket, acl string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   separator + bucket + "?acl",
	}
	req, err := newRequest(op, a.config, requestMetadata{})
	if err != nil {
		return nil, err
	}
	req.Set("x-amz-acl", acl)
	return req, nil
}

// putBucketACL set the permissions on an existing bucket using Canned ACL's.
func (a s3API) putBucketACL(bucket, acl string) error {
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
			if resp.StatusCode == http.StatusMovedPermanently {
				region, _ := a.getBucketLocation(bucket)
				endPoint := getEndpoint(region)
				errorResponse := ErrorResponse{
					Code:            "PermanentRedirect",
					Message:         "The bucket you are attempting to access must be addressed using the specified endpoint https://" + endPoint + ". Send all future requests to this endpoint.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
				return errorResponse
			}
			return BodyToErrorResponse(resp.Body)
		}
	}
	return nil
}

// getBucketACLRequest wrapper creates a new getBucketACL request.
func (a s3API) getBucketACLRequest(bucket string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + "?acl",
	}
	req, err := newRequest(op, a.config, requestMetadata{})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// getBucketACL get the acl information on an existing bucket.
func (a s3API) getBucketACL(bucket string) (accessControlPolicy, error) {
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
			if resp.StatusCode == http.StatusMovedPermanently {
				region, _ := a.getBucketLocation(bucket)
				endPoint := getEndpoint(region)
				errorResponse := ErrorResponse{
					Code:            "PermanentRedirect",
					Message:         "The bucket you are attempting to access must be addressed using the specified endpoint https://" + endPoint + ". Send all future requests to this endpoint.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
				return accessControlPolicy{}, errorResponse
			}
			return accessControlPolicy{}, BodyToErrorResponse(resp.Body)
		}
	}
	policy := accessControlPolicy{}
	err = xmlDecoder(resp.Body, &policy)
	if err != nil {
		return accessControlPolicy{}, err
	}
	// In-case of google private bucket policy doesn't have any Grant list.
	if a.config.Region == "google" {
		return policy, nil
	}
	if policy.AccessControlList.Grant == nil {
		errorResponse := ErrorResponse{
			Code:            "InternalError",
			Message:         "Access control Grant list is empty, please report this at https://github.com/minio/minio-go/issues.",
			Resource:        separator + bucket,
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
		}
		return accessControlPolicy{}, errorResponse
	}
	return policy, nil
}

// getBucketLocationRequest wrapper creates a new getBucketLocation request.
func (a s3API) getBucketLocationRequest(bucket string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + "?location",
	}
	req, err := newRequest(op, a.config, requestMetadata{})
	if err != nil {
		return nil, err
	}
	return req, nil
}

// getBucketLocation uses location subresource to return a bucket's region.
func (a s3API) getBucketLocation(bucket string) (string, error) {
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
			return "", BodyToErrorResponse(resp.Body)
		}
	}
	var locationConstraint string
	err = xmlDecoder(resp.Body, &locationConstraint)
	if err != nil {
		return "", err
	}
	return locationConstraint, nil
}

// listObjectsRequest wrapper creates a new listObjects request.
func (a s3API) listObjectsRequest(bucket, marker, prefix, delimiter string, maxkeys int) (*Request, error) {
	// resourceQuery - get resources properly escaped and lined up before using them in http request.
	resourceQuery := func() (*string, error) {
		switch {
		case marker != "":
			marker = fmt.Sprintf("&marker=%s", getURLEncodedPath(marker))
			fallthrough
		case prefix != "":
			prefix = fmt.Sprintf("&prefix=%s", getURLEncodedPath(prefix))
			fallthrough
		case delimiter != "":
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
	r, err := newRequest(op, a.config, requestMetadata{})
	if err != nil {
		return nil, err
	}
	return r, nil
}

/// Bucket Read Operations.

// listObjects - (List Objects) - List some or all (up to 1000) of the objects in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the objects in a bucket.
// request paramters :-
// ---------
// ?marker - Specifies the key to start with when listing objects in a bucket.
// ?delimiter - A delimiter is a character you use to group keys.
// ?prefix - Limits the response to keys that begin with the specified prefix.
// ?max-keys - Sets the maximum number of keys returned in the response body.
func (a s3API) listObjects(bucket, marker, prefix, delimiter string, maxkeys int) (listBucketResult, error) {
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
			if resp.StatusCode == http.StatusMovedPermanently {
				region, _ := a.getBucketLocation(bucket)
				endPoint := getEndpoint(region)
				errorResponse := ErrorResponse{
					Code:            "PermanentRedirect",
					Message:         "The bucket you are attempting to access must be addressed using the specified endpoint https://" + endPoint + ". Send all future requests to this endpoint.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
				return listBucketResult{}, errorResponse
			}
			return listBucketResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	listBucketResult := listBucketResult{}
	err = xmlDecoder(resp.Body, &listBucketResult)
	if err != nil {
		return listBucketResult, err
	}
	// close body while returning, along with any error.
	return listBucketResult, nil
}

// headBucketRequest wrapper creates a new headBucket request.
func (a s3API) headBucketRequest(bucket string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "HEAD",
		HTTPPath:   separator + bucket,
	}
	return newRequest(op, a.config, requestMetadata{})
}

// headBucket useful to determine if a bucket exists and you have permission to access it.
func (a s3API) headBucket(bucket string) error {
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
			// Head has no response body, handle it.
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusMovedPermanently:
				errorResponse = ErrorResponse{
					Code:            "PermanentRedirect",
					Message:         "The bucket you are attempting to access must be addressed using the specified endpoint https://" + getEndpoint(resp.Header.Get("x-amz-bucket-region")) + ". Send all future requests to this endpoint.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchBucket",
					Message:         "The specified bucket does not exist.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:            "AccessDenied",
					Message:         "Access Denied.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:            resp.Status,
					Message:         resp.Status,
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			}
			return errorResponse
		}
	}
	return nil
}

// deleteBucketRequest wrapper creates a new deleteBucket request.
func (a s3API) deleteBucketRequest(bucket string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   separator + bucket,
	}
	return newRequest(op, a.config, requestMetadata{})
}

// deleteBucket deletes the bucket named in the URI.
//
// NOTE: -
//  All objects (including all object versions and delete markers)
//  in the bucket must be deleted before successfully attempting this request.
func (a s3API) deleteBucket(bucket string) error {
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
		if resp.StatusCode != http.StatusNoContent {
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusMovedPermanently:
				region, _ := a.getBucketLocation(bucket)
				endPoint := getEndpoint(region)
				errorResponse = ErrorResponse{
					Code:            "PermanentRedirect",
					Message:         "The bucket you are attempting to access must be addressed using the specified endpoint https://" + endPoint + ". Send all future requests to this endpoint.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchBucket",
					Message:         "The specified bucket does not exist.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:            "AccessDenied",
					Message:         "Access Denied.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusConflict:
				errorResponse = ErrorResponse{
					Code:            "Conflict",
					Message:         "Bucket not empty.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:            resp.Status,
					Message:         resp.Status,
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			}
			return errorResponse
		}
	}
	return nil
}

/// Object Read/Write/Stat Operations

// putObjectRequest wrapper creates a new PutObject request.
func (a s3API) putObjectRequest(bucket, object string, putObjMetadata putObjectMetadata) (*Request, error) {
	if strings.TrimSpace(putObjMetadata.ContentType) == "" {
		putObjMetadata.ContentType = "application/octet-stream"
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   separator + bucket + separator + object,
	}
	rmetadata := requestMetadata{
		body:               putObjMetadata.ReadCloser,
		contentLength:      putObjMetadata.Size,
		contentType:        putObjMetadata.ContentType,
		sha256PayloadBytes: putObjMetadata.Sha256Sum,
		md5SumPayloadBytes: putObjMetadata.MD5Sum,
	}
	r, err := newRequest(op, a.config, rmetadata)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// putObject - add an object to a bucket.
// NOTE: You must have WRITE permissions on a bucket to add an object to it.
func (a s3API) putObject(bucket, object string, putObjMetadata putObjectMetadata) (ObjectStat, error) {
	req, err := a.putObjectRequest(bucket, object, putObjMetadata)
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
			// handle 301 sepcifically in case of wrong regions during path style.
			if resp.StatusCode == http.StatusMovedPermanently {
				region, _ := a.getBucketLocation(bucket)
				endPoint := getEndpoint(region)
				errorResponse := ErrorResponse{
					Code:            "PermanentRedirect",
					Message:         "The bucket you are attempting to access must be addressed using the specified endpoint https://" + endPoint + ". Send all future requests to this endpoint.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
				return ObjectStat{}, errorResponse
			}
			return ObjectStat{}, BodyToErrorResponse(resp.Body)
		}
	}
	var metadata ObjectStat
	metadata.ETag = strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	return metadata, nil
}

// presignedPostPolicy - generate post form data.
func (a s3API) presignedPostPolicy(p *PostPolicy) map[string]string {
	t := time.Now().UTC()
	r := new(Request)
	r.config = a.config
	if r.config.Signature.isV2() {
		policyBase64 := p.base64()
		p.formData["policy"] = policyBase64
		// for all other regions set this value to be 'AWSAccessKeyId'.
		if r.config.Region != "google" {
			p.formData["AWSAccessKeyId"] = r.config.AccessKeyID
		} else {
			p.formData["GoogleAccessId"] = r.config.AccessKeyID
		}
		p.formData["signature"] = r.PostPresignSignatureV2(policyBase64)
		return p.formData
	}
	credential := getCredential(r.config.AccessKeyID, r.config.Region, t)
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-date",
		value:     t.Format(iso8601DateFormat),
	})
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-algorithm",
		value:     authHeader,
	})
	p.addNewPolicy(policyCondition{
		matchType: "eq",
		condition: "$x-amz-credential",
		value:     credential,
	})
	policyBase64 := p.base64()
	p.formData["policy"] = policyBase64
	p.formData["x-amz-algorithm"] = authHeader
	p.formData["x-amz-credential"] = credential
	p.formData["x-amz-date"] = t.Format(iso8601DateFormat)
	p.formData["x-amz-signature"] = r.PostPresignSignatureV4(policyBase64, t)
	return p.formData
}

// presignedPutObject - generate presigned PUT url.
func (a s3API) presignedPutObject(bucket, object string, expires int64) (string, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   separator + bucket + separator + object,
	}
	r, err := newPresignedRequest(op, a.config, expires)
	if err != nil {
		return "", err
	}
	if r.config.Signature.isV2() {
		return r.PreSignV2()
	}
	return r.PreSignV4()
}

// presignedGetObjectRequest - presigned get object request
func (a s3API) presignedGetObjectRequest(bucket, object string, expires, offset, length int64) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + separator + object,
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

// presignedGetObject - generate presigned get object URL.
func (a s3API) presignedGetObject(bucket, object string, expires, offset, length int64) (string, error) {
	if err := invalidArgumentError(object); err != nil {
		return "", err
	}
	r, err := a.presignedGetObjectRequest(bucket, object, expires, offset, length)
	if err != nil {
		return "", err
	}
	if r.config.Signature.isV2() {
		return r.PreSignV2()
	}
	return r.PreSignV4()
}

// getObjectRequest wrapper creates a new getObject request.
func (a s3API) getObjectRequest(bucket, object string, offset, length int64) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + separator + object,
	}
	r, err := newRequest(op, a.config, requestMetadata{})
	if err != nil {
		return nil, err
	}
	switch {
	case length > 0 && offset >= 0:
		r.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))
	case offset > 0 && length == 0:
		r.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	// The final length bytes
	case length < 0 && offset == 0:
		r.Set("Range", fmt.Sprintf("bytes=%d", length))
	}
	return r, nil
}

// getObject - retrieve object from Object Storage.
//
// Additionally this function also takes range arguments to download the specified
// range bytes of an object. Setting offset and length = 0 will download the full object.
//
// For more information about the HTTP Range header.
// go to http://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35.
func (a s3API) getObject(bucket, object string, offset, length int64) (io.ReadCloser, ObjectStat, error) {
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
		// for HTTP status 200 and 204 are valid cases.
		case http.StatusOK:
		case http.StatusPartialContent:
		// handle 301 sepcifically in case of wrong regions during path style.
		case http.StatusMovedPermanently:
			region, _ := a.getBucketLocation(bucket)
			endPoint := getEndpoint(region)
			errorResponse := ErrorResponse{
				Code:            "PermanentRedirect",
				Message:         "The bucket you are attempting to access must be addressed using the specified endpoint https://" + endPoint + ". Send all future requests to this endpoint.",
				Resource:        separator + bucket,
				RequestID:       resp.Header.Get("x-amz-request-id"),
				HostID:          resp.Header.Get("x-amz-id-2"),
				AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
			}
			return nil, ObjectStat{}, errorResponse
		default:
			return nil, ObjectStat{}, BodyToErrorResponse(resp.Body)
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	date, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return nil, ObjectStat{}, ErrorResponse{
			Code:            "InternalError",
			Message:         "Last-Modified time format not recognized, please report this issue at https://github.com/minio/minio-go/issues.",
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
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

// deleteObjectRequest wrapper creates a new deleteObject request.
func (a s3API) deleteObjectRequest(bucket, object string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   separator + bucket + separator + object,
	}
	return newRequest(op, a.config, requestMetadata{})
}

// deleteObject deletes a given object from a bucket.
func (a s3API) deleteObject(bucket, object string) error {
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
	// DeleteObject always responds with http '204' even for
	// objects which do not exist. So no need to handle them
	// specifically.
	return nil
}

// headObjectRequest wrapper creates a new headObject request.
func (a s3API) headObjectRequest(bucket, object string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "HEAD",
		HTTPPath:   separator + bucket + separator + object,
	}
	return newRequest(op, a.config, requestMetadata{})
}

// headObject retrieves metadata from an object without returning the object itself.
func (a s3API) headObject(bucket, object string) (ObjectStat, error) {
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
			case http.StatusMovedPermanently:
				region, _ := a.getBucketLocation(bucket)
				endPoint := getEndpoint(region)
				errorResponse = ErrorResponse{
					Code:            "PermanentRedirect",
					Message:         "The bucket you are attempting to access must be addressed using the specified endpoint https://" + endPoint + ". Send all future requests to this endpoint.",
					Resource:        separator + bucket,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchKey",
					Message:         "The specified key does not exist.",
					Resource:        separator + bucket + separator + object,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			case http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:            "AccessDenied",
					Message:         "Access Denied.",
					Resource:        separator + bucket + separator + object,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}
			default:
				errorResponse = ErrorResponse{
					Code:            resp.Status,
					Message:         resp.Status,
					Resource:        separator + bucket + separator + object,
					RequestID:       resp.Header.Get("x-amz-request-id"),
					HostID:          resp.Header.Get("x-amz-id-2"),
					AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
				}

			}
			return ObjectStat{}, errorResponse
		}
	}
	md5sum := strings.Trim(resp.Header.Get("ETag"), "\"") // trim off the odd double quotes
	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return ObjectStat{}, ErrorResponse{
			Code:            "InternalError",
			Message:         "Content-Length not recognized, please report this issue at https://github.com/minio/minio-go/issues.",
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
		}
	}
	date, err := time.Parse(http.TimeFormat, resp.Header.Get("Last-Modified"))
	if err != nil {
		return ObjectStat{}, ErrorResponse{
			Code:            "InternalError",
			Message:         "Last-Modified time format not recognized, please report this issue at https://github.com/minio/minio-go/issues.",
			RequestID:       resp.Header.Get("x-amz-request-id"),
			HostID:          resp.Header.Get("x-amz-id-2"),
			AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
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

/// Service Operations.

// listBucketRequest wrapper creates a new listBuckets request.
func (a s3API) listBucketsRequest() (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator,
	}
	return newRequest(op, a.config, requestMetadata{})
}

// listBuckets list of all buckets owned by the authenticated sender of the request.
func (a s3API) listBuckets() (listAllMyBucketsResult, error) {
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
		// for un-authenticated requests, amazon sends a redirect handle it.
		if resp.StatusCode == http.StatusTemporaryRedirect {
			return listAllMyBucketsResult{}, ErrorResponse{
				Code:            "AccessDenied",
				Message:         "Anonymous access is forbidden for this operation.",
				RequestID:       resp.Header.Get("x-amz-request-id"),
				HostID:          resp.Header.Get("x-amz-id-2"),
				AmzBucketRegion: resp.Header.Get("x-amz-bucket-region"),
			}
		}
		if resp.StatusCode != http.StatusOK {
			return listAllMyBucketsResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	listAllMyBucketsResult := listAllMyBucketsResult{}
	err = xmlDecoder(resp.Body, &listAllMyBucketsResult)
	if err != nil {
		return listAllMyBucketsResult, err
	}
	return listAllMyBucketsResult, nil
}
