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
	"io/ioutil"
	"net/http"
	"strconv"
)

// listMultipartUploadsRequest wrapper creates a new listMultipartUploads request.
func (a s3API) listMultipartUploadsRequest(bucket, keymarker, uploadIDMarker, prefix, delimiter string, maxuploads int) (*Request, error) {
	// resourceQuery get resources properly escaped and lined up before using them in http request.
	resourceQuery := func() (string, error) {
		switch {
		case keymarker != "":
			keymarker = fmt.Sprintf("&key-marker=%s", getURLEncodedPath(keymarker))
			fallthrough
		case uploadIDMarker != "":
			uploadIDMarker = fmt.Sprintf("&upload-id-marker=%s", uploadIDMarker)
			fallthrough
		case prefix != "":
			prefix = fmt.Sprintf("&prefix=%s", getURLEncodedPath(prefix))
			fallthrough
		case delimiter != "":
			delimiter = fmt.Sprintf("&delimiter=%s", delimiter)
		}
		query := fmt.Sprintf("?uploads&max-uploads=%d", maxuploads) + keymarker + uploadIDMarker + prefix + delimiter
		return query, nil
	}
	query, err := resourceQuery()
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + query,
	}
	r, err := newRequest(op, a.config, requestMetadata{})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// listMultipartUploads - (List Multipart Uploads).
//   - Lists some or all (up to 1000) in-progress multipart uploads in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the uploads in a bucket.
// request paramters. :-
// ---------
// ?key-marker - Specifies the multipart upload after which listing should begin.
// ?upload-id-marker - Together with key-marker specifies the multipart upload after which listing should begin.
// ?delimiter - A delimiter is a character you use to group keys.
// ?prefix - Limits the response to keys that begin with the specified prefix.
// ?max-uploads - Sets the maximum number of multipart uploads returned in the response body.
func (a s3API) listMultipartUploads(bucket, keymarker, uploadIDMarker, prefix, delimiter string, maxuploads int) (listMultipartUploadsResult, error) {
	req, err := a.listMultipartUploadsRequest(bucket, keymarker, uploadIDMarker, prefix, delimiter, maxuploads)
	if err != nil {
		return listMultipartUploadsResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return listMultipartUploadsResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return listMultipartUploadsResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	listMultipartUploadsResult := listMultipartUploadsResult{}
	err = xmlDecoder(resp.Body, &listMultipartUploadsResult)
	if err != nil {
		return listMultipartUploadsResult, err
	}
	return listMultipartUploadsResult, nil
}

// initiateMultipartRequest wrapper creates a new initiateMultiPart request.
func (a s3API) initiateMultipartRequest(bucket, object string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "POST",
		HTTPPath:   separator + bucket + separator + object + "?uploads",
	}
	return newRequest(op, a.config, requestMetadata{})
}

// initiateMultipartUpload initiates a multipart upload and returns an upload ID.
func (a s3API) initiateMultipartUpload(bucket, object string) (initiateMultipartUploadResult, error) {
	req, err := a.initiateMultipartRequest(bucket, object)
	if err != nil {
		return initiateMultipartUploadResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return initiateMultipartUploadResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return initiateMultipartUploadResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	initiateMultipartUploadResult := initiateMultipartUploadResult{}
	err = xmlDecoder(resp.Body, &initiateMultipartUploadResult)
	if err != nil {
		return initiateMultipartUploadResult, err
	}
	return initiateMultipartUploadResult, nil
}

// completeMultipartUploadRequest wrapper creates a new CompleteMultipartUpload request.
func (a s3API) completeMultipartUploadRequest(bucket, object, uploadID string, complete completeMultipartUpload) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "POST",
		HTTPPath:   separator + bucket + separator + object + "?uploadId=" + uploadID,
	}
	completeMultipartUploadBytes, err := xml.Marshal(complete)
	if err != nil {
		return nil, err
	}
	completeMultipartUploadBuffer := bytes.NewBuffer(completeMultipartUploadBytes)
	rmetadata := requestMetadata{
		body:               ioutil.NopCloser(completeMultipartUploadBuffer),
		contentLength:      int64(completeMultipartUploadBuffer.Len()),
		sha256PayloadBytes: sum256(completeMultipartUploadBuffer.Bytes()),
	}
	r, err := newRequest(op, a.config, rmetadata)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// completeMultipartUpload completes a multipart upload by assembling previously uploaded parts.
func (a s3API) completeMultipartUpload(bucket, object, uploadID string, c completeMultipartUpload) (completeMultipartUploadResult, error) {
	req, err := a.completeMultipartUploadRequest(bucket, object, uploadID, c)
	if err != nil {
		return completeMultipartUploadResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return completeMultipartUploadResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return completeMultipartUploadResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	completeMultipartUploadResult := completeMultipartUploadResult{}
	err = xmlDecoder(resp.Body, &completeMultipartUploadResult)
	if err != nil {
		return completeMultipartUploadResult, err
	}
	return completeMultipartUploadResult, nil
}

// abortMultipartUploadRequest wrapper creates a new AbortMultipartUpload request.
func (a s3API) abortMultipartUploadRequest(bucket, object, uploadID string) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   separator + bucket + separator + object + "?uploadId=" + uploadID,
	}
	return newRequest(op, a.config, requestMetadata{})
}

// abortMultipartUpload aborts a multipart upload for the given uploadID, all parts are deleted.
func (a s3API) abortMultipartUpload(bucket, object, uploadID string) error {
	req, err := a.abortMultipartUploadRequest(bucket, object, uploadID)
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
			// Abort has no response body, handle it.
			var errorResponse ErrorResponse
			switch resp.StatusCode {
			case http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:            "NoSuchUpload",
					Message:         "The specified multipart upload does not exist.",
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
					Message:         "Unknown error, please report this at https://github.com/minio/minio-go-legacy/issues.",
					Resource:        separator + bucket + separator + object,
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

// listObjectPartsRequest wrapper creates a new ListObjectParts request.
func (a s3API) listObjectPartsRequest(bucket, object, uploadID string, partNumberMarker, maxParts int) (*Request, error) {
	// resourceQuery - get resources properly escaped and lined up before using them in http request.
	resourceQuery := func() string {
		var partNumberMarkerStr string
		switch {
		case partNumberMarker != 0:
			partNumberMarkerStr = fmt.Sprintf("&part-number-marker=%d", partNumberMarker)
		}
		return fmt.Sprintf("?uploadId=%s&max-parts=%d", uploadID, maxParts) + partNumberMarkerStr
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "GET",
		HTTPPath:   separator + bucket + separator + object + resourceQuery(),
	}
	return newRequest(op, a.config, requestMetadata{})
}

// listObjectParts (List Parts)
//     - lists some or all (up to 1000) parts that have been uploaded for a specific multipart upload
//
// You can use the request parameters as selection criteria to return a subset of the uploads in a bucket.
// request paramters :-
// ---------
// ?part-number-marker - Specifies the part after which listing should begin.
func (a s3API) listObjectParts(bucket, object, uploadID string, partNumberMarker, maxParts int) (listObjectPartsResult, error) {
	req, err := a.listObjectPartsRequest(bucket, object, uploadID, partNumberMarker, maxParts)
	if err != nil {
		return listObjectPartsResult{}, err
	}
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return listObjectPartsResult{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return listObjectPartsResult{}, BodyToErrorResponse(resp.Body)
		}
	}
	listObjectPartsResult := listObjectPartsResult{}
	err = xmlDecoder(resp.Body, &listObjectPartsResult)
	if err != nil {
		return listObjectPartsResult, err
	}
	return listObjectPartsResult, nil
}

// uploadPartRequest wrapper creates a new UploadPart request.
func (a s3API) uploadPartRequest(bucket, object, uploadID string, uploadingPart partMetadata) (*Request, error) {
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath: separator + bucket + separator + object +
			"?partNumber=" + strconv.Itoa(uploadingPart.Number) + "&uploadId=" + uploadID,
	}
	rmetadata := requestMetadata{
		body:               uploadingPart.ReadCloser,
		contentLength:      uploadingPart.Size,
		sha256PayloadBytes: uploadingPart.Sha256Sum,
		md5SumPayloadBytes: uploadingPart.MD5Sum,
	}
	r, err := newRequest(op, a.config, rmetadata)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// uploadPart uploads a part in a multipart upload.
func (a s3API) uploadPart(bucket, object, uploadID string, uploadingPart partMetadata) (completePart, error) {
	req, err := a.uploadPartRequest(bucket, object, uploadID, uploadingPart)
	if err != nil {
		return completePart{}, err
	}

	// initiate the request.
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return completePart{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return completePart{}, BodyToErrorResponse(resp.Body)
		}
	}
	cPart := completePart{}
	cPart.PartNumber = uploadingPart.Number
	cPart.ETag = resp.Header.Get("ETag")
	return cPart, nil
}
