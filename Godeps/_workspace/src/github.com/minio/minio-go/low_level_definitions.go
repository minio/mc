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

package client

import "time"

// listAllMyBucketsResult container for ListBucets response
type listAllMyBucketsResult struct {
	// Container for one or more buckets.
	Buckets struct {
		Bucket []*BucketStat
	}
	Owner owner
}

// owner container for bucket owner information
type owner struct {
	DisplayName string
	ID          string
}

// commonPrefix container for prefix response in ListObjects
type commonPrefix struct {
	Prefix string
}

// listBucketResult container for ListObjects response
type listBucketResult struct {
	CommonPrefixes []*commonPrefix // A response can contain CommonPrefixes only if you specify a delimiter
	Contents       []*ObjectStat   // Metadata about each object returned
	Delimiter      string

	// Encoding type used to encode object keys in the response.
	EncodingType string

	// A flag that indicates whether or not ListObjects returned all of the results
	// that satisfied the search criteria.
	IsTruncated bool
	Marker      string
	MaxKeys     int64
	Name        string

	// When response is truncated (the IsTruncated element value in the response
	// is true), you can use the key name in this field as marker in the subsequent
	// request to get next set of objects. Object storage lists objects in alphabetical
	// order Note: This element is returned only if you have delimiter request parameter
	// specified. If response does not include the NextMaker and it is truncated,
	// you can use the value of the last Key in the response as the marker in the
	// subsequent request to get the next set of object keys.
	NextMarker string
	Prefix     string
}

type upload struct {
	Key      string
	UploadID string `xml:"UploadId"`

	Initiator initiator
	Owner     owner

	StorageClass string
	Initiated    time.Time `type:"timestamp" timestampFormat:"iso8601"`
}

// listMultipartUploadsResult container for ListMultipartUploads response
type listMultipartUploadsResult struct {
	Bucket             string
	KeyMarker          string
	UploadIDMarker     string `xml:"UploadIdMarker"`
	NextKeyMarker      string
	NextUploadIDMarker string `xml:"NextUploadIdMarker"`
	EncodingType       string
	MaxUploads         int64
	IsTruncated        bool
	Upload             []*upload
	Prefix             string
	Delimiter          string
	CommonPrefixes     []*commonPrefix // A response can contain CommonPrefixes only if you specify a delimiter
}

// initiator container for who initiated multipart upload
type initiator struct {
	ID          string
	DisplayName string
}

// partMetadata container for particular part of an object
type partMetadata struct {
	PartNumber   int
	LastModified time.Time
	ETag         string
	Size         int64
}

// listObjectPartsResult container for ListObjectParts response
type listObjectPartsResult struct {
	Bucket   string
	Key      string
	UploadID string `xml:"UploadId"`

	Initiator initiator
	Owner     owner

	StorageClass         string
	PartNumberMarker     int
	NextPartNumberMarker int
	MaxParts             int

	// Indicates whether the returned list of parts is truncated.
	IsTruncated bool
	Part        []*partMetadata

	EncodingType string
}

// initiateMultipartUploadResult container for InitiateMultiPartUpload response
type initiateMultipartUploadResult struct {
	Bucket   string
	Key      string
	UploadID string `xml:"UploadId"`
}

// completeMultipartUploadResult containe for completed multipart upload response
type completeMultipartUploadResult struct {
	Location string
	Bucket   string
	Key      string
	ETag     string
}

// completePart sub container lists individual part numbers and their md5sum, part of CompleteMultipartUpload
type completePart struct {
	PartNumber int
	ETag       string
}

// completeMultipartUpload container for completing multipart upload
type completeMultipartUpload struct {
	Part []*completePart
}

// createBucketConfiguration container for bucket configuration
type createBucketConfiguration struct {
	Location string `xml:"LocationConstraint"`
}

type grantee struct {
	ID           string
	DisplayName  string
	EmailAddress string
	Type         string
	URI          string
}

type grant struct {
	Grantee    grantee
	Permission string
}

type accessControlPolicy struct {
	Owner             owner
	AccessControlList struct {
		Grant *grant
	}
}
