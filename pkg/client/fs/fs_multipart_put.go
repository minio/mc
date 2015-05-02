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

package fs

import (
	"io"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// Multipart object upload handlers

// InitiateMultiPartUpload -
func (c *fsClient) InitiateMultiPartUpload() (objectID string, err error) {
	return "", iodine.New(client.APINotImplemented{API: "InitiateMultiPartUpload"}, nil)
}

// UploadPart -
func (c *fsClient) UploadPart(uploadID string, body io.ReadSeeker, contentLength, partNumber int64) (md5hex string, err error) {
	return "", iodine.New(client.APINotImplemented{API: "UploadPart"}, nil)
}

// CompleteMultiPartUpload -
func (c *fsClient) CompleteMultiPartUpload(uploadID string) (location, md5hex string, err error) {
	return "", "", iodine.New(client.APINotImplemented{API: "CompleteMultiPartUpload"}, nil)
}

// AbortMultiPartUpload -
func (c *fsClient) AbortMultiPartUpload(uploadID string) error {
	return iodine.New(client.APINotImplemented{API: "AbortMultiPartUpload"}, nil)
}

// ListParts -
func (c *fsClient) ListParts(uploadID string) (contents *client.PartContents, err error) {
	return nil, iodine.New(client.APINotImplemented{API: "ListParts"}, nil)
}
