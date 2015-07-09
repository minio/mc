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

package minio

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// listMultipartUploadsRequest wrapper creates a new listMultipartUploads request
func (a lowLevelAPI) listMultipartUploadsRequest(bucket, keymarker, uploadIDMarker, prefix, delimiter string, maxuploads int) (*request, error) {
	// resourceQuery - get resources properly escaped and lined up before using them in http request
	resourceQuery := func() (string, error) {
		var err error
		switch {
		case keymarker != "":
			keymarker, err = urlEncodeName(keymarker)
			if err != nil {
				return "", err
			}
			keymarker = fmt.Sprintf("&key-marker=%s", keymarker)
			fallthrough
		case uploadIDMarker != "":
			uploadIDMarker, err = urlEncodeName(uploadIDMarker)
			if err != nil {
				return "", err
			}
			uploadIDMarker = fmt.Sprintf("&upload-id-marker=%s", uploadIDMarker)
			fallthrough
		case prefix != "":
			prefix, err = urlEncodeName(prefix)
			if err != nil {
				return "", err
			}
			prefix = fmt.Sprintf("&prefix=%s", prefix)
			fallthrough
		case delimiter != "":
			delimiter, err = urlEncodeName(delimiter)
			if err != nil {
				return "", err
			}
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
		HTTPPath:   "/" + bucket + query,
	}
	r, err := newRequest(op, a.config, nil)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// listMultipartUploads - (List Multipart Uploads) - Lists some or all (up to 1000) in-progress multipart uploads in a bucket.
//
// You can use the request parameters as selection criteria to return a subset of the uploads in a bucket.
// request paramters :-
// ---------
// ?key-marker - Specifies the multipart upload after which listing should begin
// ?upload-id-marker - Together with key-marker specifies the multipart upload after which listing should begin
// ?delimiter - A delimiter is a character you use to group keys.
// ?prefix - Limits the response to keys that begin with the specified prefix.
// ?max-uploads - Sets the maximum number of multipart uploads returned in the response body.
func (a lowLevelAPI) listMultipartUploads(bucket, keymarker, uploadIDMarker, prefix, delimiter string, maxuploads int) (listMultipartUploadsResult, error) {
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
			return listMultipartUploadsResult{}, a.responseToError(resp.Body)
		}
	}
	listMultipartUploadsResult := listMultipartUploadsResult{}
	err = acceptTypeDecoder(resp.Body, a.config.AcceptType, &listMultipartUploadsResult)
	if err != nil {
		return listMultipartUploadsResult, err
	}
	// close body while returning, along with any error
	return listMultipartUploadsResult, nil
}

// initiateMultipartRequest wrapper creates a new initiateMultiPart request
func (a lowLevelAPI) initiateMultipartRequest(bucket, object string) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "POST",
		HTTPPath:   "/" + bucket + "/" + encodedObject + "?uploads",
	}
	return newRequest(op, a.config, nil)
}

// initiateMultipartUpload initiates a multipart upload and returns an upload ID
func (a lowLevelAPI) initiateMultipartUpload(bucket, object string) (initiateMultipartUploadResult, error) {
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
			return initiateMultipartUploadResult{}, a.responseToError(resp.Body)
		}
	}
	initiateMultipartUploadResult := initiateMultipartUploadResult{}
	err = acceptTypeDecoder(resp.Body, a.config.AcceptType, &initiateMultipartUploadResult)
	if err != nil {
		return initiateMultipartUploadResult, err
	}
	return initiateMultipartUploadResult, nil
}

// completeMultipartUploadRequest wrapper creates a new CompleteMultipartUpload request
func (a lowLevelAPI) completeMultipartUploadRequest(bucket, object, uploadID string, complete completeMultipartUpload) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "POST",
		HTTPPath:   "/" + bucket + "/" + encodedObject + "?uploadId=" + uploadID,
	}
	var completeMultipartUploadBytes []byte
	switch {
	case a.config.AcceptType == "application/xml":
		completeMultipartUploadBytes, err = xml.Marshal(complete)
	case a.config.AcceptType == "application/json":
		completeMultipartUploadBytes, err = json.Marshal(complete)
	default:
		completeMultipartUploadBytes, err = xml.Marshal(complete)
	}
	if err != nil {
		return nil, err
	}
	completeMultipartUploadBuffer := bytes.NewReader(completeMultipartUploadBytes)
	r, err := newRequest(op, a.config, completeMultipartUploadBuffer)
	if err != nil {
		return nil, err
	}
	r.req.ContentLength = int64(completeMultipartUploadBuffer.Len())
	return r, nil
}

// completeMultipartUpload completes a multipart upload by assembling previously uploaded parts.
func (a lowLevelAPI) completeMultipartUpload(bucket, object, uploadID string, c completeMultipartUpload) (completeMultipartUploadResult, error) {
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
			return completeMultipartUploadResult{}, a.responseToError(resp.Body)
		}
	}
	completeMultipartUploadResult := completeMultipartUploadResult{}
	err = acceptTypeDecoder(resp.Body, a.config.AcceptType, &completeMultipartUploadResult)
	if err != nil {
		return completeMultipartUploadResult, err
	}
	return completeMultipartUploadResult, nil
}

// abortMultipartUploadRequest wrapper creates a new AbortMultipartUpload request
func (a lowLevelAPI) abortMultipartUploadRequest(bucket, object, uploadID string) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket + "/" + encodedObject + "?uploadId=" + uploadID,
	}
	return newRequest(op, a.config, nil)
}

// abortMultipartUpload aborts a multipart upload for the given uploadID, all parts are deleted
func (a lowLevelAPI) abortMultipartUpload(bucket, object, uploadID string) error {
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
			// Abort has no response body, handle it
			var errorResponse ErrorResponse
			switch {
			case resp.StatusCode == http.StatusNotFound:
				errorResponse = ErrorResponse{
					Code:      "NoSuchUpload",
					Message:   "The specified multipart upload does not exist.",
					Resource:  "/" + bucket + "/" + object,
					RequestID: resp.Header.Get("x-amz-request-id"),
				}
			case resp.StatusCode == http.StatusForbidden:
				errorResponse = ErrorResponse{
					Code:      "AccessDenied",
					Message:   "Access Denied",
					Resource:  "/" + bucket + "/" + object,
					RequestID: resp.Header.Get("x-amz-request-id"),
				}
			}
			return errorResponse
		}
	}
	return nil
}

// listObjectPartsRequest wrapper creates a new ListObjectParts request
func (a lowLevelAPI) listObjectPartsRequest(bucket, object, uploadID string, partNumberMarker, maxParts int) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	// resourceQuery - get resources properly escaped and lined up before using them in http request
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
		HTTPPath:   "/" + bucket + "/" + encodedObject + resourceQuery(),
	}
	return newRequest(op, a.config, nil)
}

// listObjectParts (List Parts) - lists some or all (up to 1000) parts that have been uploaded for a specific multipart upload
//
// You can use the request parameters as selection criteria to return a subset of the uploads in a bucket.
// request paramters :-
// ---------
// ?part-number-marker - Specifies the part after which listing should begin.
func (a lowLevelAPI) listObjectParts(bucket, object, uploadID string, partNumberMarker, maxParts int) (listObjectPartsResult, error) {
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
			return listObjectPartsResult{}, a.responseToError(resp.Body)
		}
	}
	listObjectPartsResult := listObjectPartsResult{}
	err = acceptTypeDecoder(resp.Body, a.config.AcceptType, &listObjectPartsResult)
	if err != nil {
		return listObjectPartsResult, err
	}
	return listObjectPartsResult, nil
}

// uploadPartRequest wrapper creates a new UploadPart request
func (a lowLevelAPI) uploadPartRequest(bucket, object, uploadID string, md5SumBytes []byte, partNumber int, size int64, body io.ReadSeeker) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.Endpoint,
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "/" + encodedObject + "?partNumber=" + strconv.Itoa(partNumber) + "&uploadId=" + uploadID,
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

// uploadPart uploads a part in a multipart upload.
func (a lowLevelAPI) uploadPart(bucket, object, uploadID string, md5SumBytes []byte, partNumber int, size int64, body io.ReadSeeker) (completePart, error) {
	req, err := a.uploadPartRequest(bucket, object, uploadID, md5SumBytes, partNumber, size, body)
	if err != nil {
		return completePart{}, err
	}
	cPart := completePart{}
	cPart.PartNumber = partNumber
	cPart.ETag = "\"" + hex.EncodeToString(md5SumBytes) + "\""

	// initiate the request
	resp, err := req.Do()
	defer closeResp(resp)
	if err != nil {
		return completePart{}, err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return completePart{}, a.responseToError(resp.Body)
		}
	}
	return cPart, nil
}
