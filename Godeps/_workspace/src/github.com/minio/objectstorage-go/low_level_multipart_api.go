package objectstorage

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

// initiateMultipartRequest wrapper creates a new InitiateMultiPart request
func (a *lowLevelAPI) initiateMultipartRequest(bucket, object string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "POST",
		HTTPPath:   "/" + bucket + "/" + object + "?uploads",
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
	return initiateMultipartUploadResult, resp.Body.Close()
}

// completeMultipartUploadRequest wrapper creates a new CompleteMultipartUpload request
func (a *lowLevelAPI) completeMultipartUploadRequest(bucket, object, uploadID string, complete *completeMultipartUpload) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "POST",
		HTTPPath:   "/" + bucket + "/" + object + "?uploadId=" + uploadID,
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
	return completeMultipartUploadResult, resp.Body.Close()
}

// abortMultipartUploadRequest wrapper creates a new AbortMultipartUpload request
func (a *lowLevelAPI) abortMultipartUploadRequest(bucket, object, uploadID string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "DELETE",
		HTTPPath:   "/" + bucket + "/" + object + "?uploadId=" + uploadID,
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
func (a *lowLevelAPI) listObjectPartsRequest(bucket, object, uploadID string) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "GET",
		HTTPPath:   "/" + bucket + "/" + object + "?uploadId=" + uploadID,
	}
	return newRequest(op, a.config, nil)
}

// listObjectParts lists the parts that have been uploaded for a specific multipart upload.
func (a *lowLevelAPI) listObjectParts(bucket, object, uploadID string) (*listObjectPartsResult, error) {
	req, err := a.listObjectPartsRequest(bucket, object, uploadID)
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
	listObjectPartsResult := new(listObjectPartsResult)
	decoder := xml.NewDecoder(resp.Body)
	err = decoder.Decode(listObjectPartsResult)
	if err != nil {
		return nil, err
	}
	return listObjectPartsResult, resp.Body.Close()
}

// uploadPartRequest wrapper creates a new UploadPart request
func (a *lowLevelAPI) uploadPartRequest(bucket, object, uploadID string, partNumber int, size int64, body io.ReadSeeker) (*request, error) {
	op := &operation{
		HTTPServer: a.config.MustGetEndpoint(),
		HTTPMethod: "PUT",
		HTTPPath:   "/" + bucket + "/" + object + "?partNumber=" + strconv.Itoa(partNumber) + "&uploadId=" + uploadID,
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
	if resp != nil {
		if resp.StatusCode != http.StatusOK {
			return nil, responseToError(resp)
		}
	}
	return completePart, resp.Body.Close()
}
