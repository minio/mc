package client

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// listMultipartUploadsRequest wrapper creates a new listMultipartUploads request
func (a *lowLevelAPI) listMultipartUploadsRequest(bucket, keymarker, uploadIDMarker, prefix, delimiter string, maxuploads int) (*request, error) {
	// resourceQuery - get resources properly escaped and lined up before using them in http request
	resourceQuery := func() (*string, error) {
		var err error
		switch {
		case keymarker != "":
			keymarker, err = urlEncodeName(keymarker)
			if err != nil {
				return nil, err
			}
			keymarker = fmt.Sprintf("&key-marker=%s", keymarker)
			fallthrough
		case uploadIDMarker != "":
			uploadIDMarker, err = urlEncodeName(uploadIDMarker)
			if err != nil {
				return nil, err
			}
			uploadIDMarker = fmt.Sprintf("&upload-id-marker=%s", uploadIDMarker)
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
		query := fmt.Sprintf("?uploads&max-uploads=%d", maxuploads) + keymarker + uploadIDMarker + prefix + delimiter
		return &query, nil
	}
	query, err := resourceQuery()
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + *query,
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
func (a *lowLevelAPI) listMultipartUploads(bucket, keymarker, uploadIDMarker, prefix, delimiter string, maxuploads int) (*listMultipartUploadsResult, error) {
	req, err := a.listMultipartUploadsRequest(bucket, keymarker, uploadIDMarker, prefix, delimiter, maxuploads)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	listMultipartUploadsResult := new(listMultipartUploadsResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listMultipartUploadsResult)
	if err != nil {
		return nil, err
	}

	// close body while returning, along with any error
	return listMultipartUploadsResult, nil
}

// initiateMultipartRequest wrapper creates a new initiateMultiPart request
func (a *lowLevelAPI) initiateMultipartRequest(bucket, object string) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "POST",
		HTTPPath:   "/" + bucket + "/" + encodedObject + "?uploads",
	}
	return newRequest(op, a.config, nil)
}

// initiateMultipartUpload initiates a multipart upload and returns an upload ID
func (a *lowLevelAPI) initiateMultipartUpload(bucket, object string) (*initiateMultipartUploadResult, error) {
	req, err := a.initiateMultipartRequest(bucket, object)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	initiateMultipartUploadResult := new(initiateMultipartUploadResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(initiateMultipartUploadResult)
	if err != nil {
		return nil, err
	}
	return initiateMultipartUploadResult, nil
}

// completeMultipartUploadRequest wrapper creates a new CompleteMultipartUpload request
func (a *lowLevelAPI) completeMultipartUploadRequest(bucket, object, uploadID string, complete *completeMultipartUpload) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "POST",
		HTTPPath:   "/" + bucket + "/" + encodedObject + "?uploadId=" + uploadID,
	}
	completeMultipartUploadBytes, err := xml.Marshal(complete)
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
func (a *lowLevelAPI) completeMultipartUpload(bucket, object, uploadID string, c *completeMultipartUpload) (*completeMultipartUploadResult, error) {
	req, err := a.completeMultipartUploadRequest(bucket, object, uploadID, c)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	completeMultipartUploadResult := new(completeMultipartUploadResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(completeMultipartUploadResult)
	if err != nil {
		return nil, err
	}
	return completeMultipartUploadResult, nil
}

// abortMultipartUploadRequest wrapper creates a new AbortMultipartUpload request
func (a *lowLevelAPI) abortMultipartUploadRequest(bucket, object, uploadID string) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket + "/" + encodedObject + "?uploadId=" + uploadID,
	}
	return newRequest(op, a.config, nil)
}

// abortMultipartUpload aborts a multipart upload for the given uploadID, all parts are deleted
func (a *lowLevelAPI) abortMultipartUpload(bucket, object, uploadID string) error {
	req, err := a.abortMultipartUploadRequest(bucket, object, uploadID)
	if err != nil {
		return err
	}
	resp, err := req.Do()
	if err != nil {
		return err
	}
	if resp != nil {
		if resp.StatusCode != http.StatusNoContent {
			// Abort has no response body, handle it
			return fmt.Errorf("%s", resp.Status)
		}
	}
	return resp.Body.Close()
}

// listObjectPartsRequest wrapper creates a new ListObjectParts request
func (a *lowLevelAPI) listObjectPartsRequest(bucket, object, uploadID string, partNumberMarker, maxParts int) (*request, error) {
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
		HTTPServer: a.config.MustGetEndpoint(),
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
func (a *lowLevelAPI) listObjectParts(bucket, object, uploadID string, partNumberMarker, maxParts int) (*listObjectPartsResult, error) {
	req, err := a.listObjectPartsRequest(bucket, object, uploadID, partNumberMarker, maxParts)
	if err != nil {
		return nil, err
	}
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	listObjectPartsResult := new(listObjectPartsResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listObjectPartsResult)
	if err != nil {
		return nil, err
	}
	return listObjectPartsResult, nil
}

// uploadPartRequest wrapper creates a new UploadPart request
func (a *lowLevelAPI) uploadPartRequest(bucket, object, uploadID string, partNumber int, size int64, body io.ReadSeeker) (*request, error) {
	encodedObject, err := urlEncodeName(object)
	if err != nil {
		return nil, err
	}
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "/" + encodedObject + "?partNumber=" + strconv.Itoa(partNumber) + "&uploadId=" + uploadID,
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

// uploadPart uploads a part in a multipart upload.
func (a *lowLevelAPI) uploadPart(bucket, object, uploadID string, partNumber int, size int64, body io.ReadSeeker) (*completePart, error) {
	req, err := a.uploadPartRequest(bucket, object, uploadID, partNumber, size, body)
	if err != nil {
		return nil, err
	}
	// get hex encoding for md5sum in base64
	md5SumBytes, err := base64.StdEncoding.DecodeString(req.Get("Content-MD5"))
	if err != nil {
		return nil, err
	}
	completePart := new(completePart)
	completePart.PartNumber = partNumber
	completePart.ETag = "\"" + hex.EncodeToString(md5SumBytes) + "\""

	// initiate the request
	resp, err := req.Do()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	return completePart, nil
}
