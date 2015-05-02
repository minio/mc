/*
 * Minio Client (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this fs except in compliance with the License.
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
	"io"

	"github.com/awslabs/aws-sdk-go/service/s3"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// Multipart object upload handlers

// InitiateMultiPartUpload - start multipart upload session
func (c *s3Client) InitiateMultiPartUpload() (uploadID string, err error) {
	bucket, object := c.url2BucketAndObject()

	multiparthUploadInput := new(s3.CreateMultipartUploadInput)
	multiparthUploadInput.Bucket = &bucket
	multiparthUploadInput.Key = &object
	multiparthUploadOutput, err := c.S3.CreateMultipartUpload(multiparthUploadInput)
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return *multiparthUploadOutput.UploadID, nil
}

// UploadPart - start uploading individual parts
func (c *s3Client) UploadPart(uploadID string, body io.ReadSeeker, contentLength, partNumber int64) (md5hex string, err error) {
	bucket, object := c.url2BucketAndObject()

	uploadPartInput := new(s3.UploadPartInput)
	uploadPartInput.Bucket = &bucket
	uploadPartInput.Key = &object
	uploadPartInput.Body = body
	uploadPartInput.PartNumber = &partNumber
	uploadPartInput.UploadID = &uploadID
	uploadPartOutput, err := c.S3.UploadPart(uploadPartInput)
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return *uploadPartOutput.ETag, nil
}

// CompleteMultiPartUpload
func (c *s3Client) CompleteMultiPartUpload(uploadID string) (location, md5hex string, err error) {
	bucket, object := c.url2BucketAndObject()

	completeMultiPartUploadInput := new(s3.CompleteMultipartUploadInput)
	completeMultiPartUploadInput.Bucket = &bucket
	completeMultiPartUploadInput.Key = &object
	completeMultiPartUploadInput.UploadID = &uploadID

	completeMultiPartUploadOutput, err := c.S3.CompleteMultipartUpload(completeMultiPartUploadInput)
	if err != nil {
		return "", "", iodine.New(err, nil)
	}
	return *completeMultiPartUploadOutput.Location, *completeMultiPartUploadOutput.ETag, nil
}

// AbortMultiPartUpload
func (c *s3Client) AbortMultiPartUpload(uploadID string) error {
	bucket, object := c.url2BucketAndObject()
	abortMultiPartUploadInput := new(s3.AbortMultipartUploadInput)
	abortMultiPartUploadInput.Bucket = &bucket
	abortMultiPartUploadInput.Key = &object
	abortMultiPartUploadInput.UploadID = &uploadID
	if _, err := c.S3.AbortMultipartUpload(abortMultiPartUploadInput); err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// ListParts
func (c *s3Client) ListParts(uploadID string) (contents *client.PartContents, err error) {
	bucket, object := c.url2BucketAndObject()
	listPartsInput := new(s3.ListPartsInput)
	listPartsInput.Bucket = &bucket
	listPartsInput.Key = &object
	listPartsInput.UploadID = &uploadID

	listPartsOutput, err := c.S3.ListParts(listPartsInput)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	contents = new(client.PartContents)
	contents.Key = *listPartsOutput.Key
	contents.IsTruncated = *listPartsOutput.IsTruncated
	contents.UploadID = *listPartsOutput.UploadID
	for _, part := range listPartsOutput.Parts {
		newPart := new(client.Part)
		newPart.ETag = *part.ETag
		newPart.LastModified = *part.LastModified
		newPart.PartNumber = *part.PartNumber
		newPart.Size = *part.Size
		contents.Parts = append(contents.Parts, newPart)
	}
	return contents, nil

}
