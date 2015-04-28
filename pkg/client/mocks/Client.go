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

package mocks

import (
	"io"

	"github.com/minio-io/mc/pkg/client"
	"github.com/stretchr/testify/mock"
)

// Client mock
type Client struct {
	mock.Mock
}

// PutBucket is a mock method
func (m *Client) PutBucket(acl string) error {
	ret := m.Called(acl)

	r0 := ret.Error(0)

	return r0
}

// Stat is a mock method
func (m *Client) Stat() (*client.Item, error) {
	ret := m.Called()

	var r0 *client.Item
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*client.Item)
	}

	r1 := ret.Error(1)

	return r0, r1
}

// List is a mock method
func (m *Client) List() <-chan client.ItemOnChannel {
	ret := m.Called()
	r0 := ret.Get(0).(chan client.ItemOnChannel)
	return r0
}

// ListRecursive is a mock method
func (m *Client) ListRecursive() <-chan client.ItemOnChannel {
	ret := m.Called()
	r0 := ret.Get(0).(chan client.ItemOnChannel)
	return r0
}

// Get is a mock method
func (m *Client) Get() (io.ReadCloser, int64, string, error) {
	ret := m.Called()

	r0 := ret.Get(0).(io.ReadCloser)
	r1 := ret.Get(1).(int64)
	r2 := ret.Get(2).(string)
	r3 := ret.Error(3)

	return r0, r1, r2, r3
}

// GetPartial is a mock method
func (m *Client) GetPartial(offset int64, length int64) (io.ReadCloser, int64, string, error) {
	ret := m.Called(offset, length)

	r0 := ret.Get(0).(io.ReadCloser)
	r1 := ret.Get(1).(int64)
	r2 := ret.Get(2).(string)
	r3 := ret.Error(3)

	return r0, r1, r2, r3
}

// Put is a mock method
func (m *Client) Put(md5 string, size int64) (io.WriteCloser, error) {
	ret := m.Called(md5, size)

	r0 := ret.Get(0).(io.WriteCloser)
	r1 := ret.Error(1)

	return r0, r1
}

// InitiateMultiPartUpload is a mock method
func (m *Client) InitiateMultiPartUpload() (string, error) {
	ret := m.Called()

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}

// UploadPart is a mock method
func (m *Client) UploadPart(uploadID string, body io.ReadSeeker, contentLength, partNumber int64) (string, error) {
	ret := m.Called(uploadID, body, contentLength, partNumber)

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}

// CompleteMultiPartUpload is a mock method
func (m *Client) CompleteMultiPartUpload(uploadID string) (string, string, error) {
	ret := m.Called(uploadID)

	r0 := ret.Get(0).(string)
	r1 := ret.Get(1).(string)
	r2 := ret.Error(2)

	return r0, r1, r2
}

// AbortMultiPartUpload is a mock method
func (m *Client) AbortMultiPartUpload(uploadID string) error {
	ret := m.Called(uploadID)

	r0 := ret.Error(0)

	return r0
}

// ListParts is a mock method
func (m *Client) ListParts(uploadID string) (*client.PartItems, error) {
	ret := m.Called(uploadID)

	var r0 *client.PartItems
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*client.PartItems)
	}
	r1 := ret.Error(1)

	return r0, r1
}
