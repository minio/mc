package mocks

import "github.com/minio-io/mc/pkg/client"
import "github.com/stretchr/testify/mock"

import "io"

// Client mock
type Client struct {
	mock.Mock
}

// PutBucket is a mock method
func (m *Client) PutBucket(bucket string) error {
	ret := m.Called(bucket)

	r0 := ret.Error(0)

	return r0
}

// StatBucket is a mock method
func (m *Client) StatBucket(bucket string) error {
	ret := m.Called(bucket)

	r0 := ret.Error(0)

	return r0
}

// ListBuckets is a mock method
func (m *Client) ListBuckets() ([]*client.Bucket, error) {
	ret := m.Called()

	r0 := ret.Get(0).([]*client.Bucket)
	r1 := ret.Error(1)

	return r0, r1
}

// ListObjects is a mock method
func (m *Client) ListObjects(bucket string, keyPrefix string) ([]*client.Item, error) {
	ret := m.Called(bucket, keyPrefix)

	r0 := ret.Get(0).([]*client.Item)
	r1 := ret.Error(1)

	return r0, r1
}

// Get is a mock method
func (m *Client) Get(bucket string, object string) (io.ReadCloser, int64, string, error) {
	ret := m.Called(bucket, object)

	r0 := ret.Get(0).(io.ReadCloser)
	r1 := ret.Get(1).(int64)
	r2 := ret.Get(2).(string)
	r3 := ret.Error(3)

	return r0, r1, r2, r3
}

// GetPartial is a mock method
func (m *Client) GetPartial(bucket string, key string, offset int64, length int64) (io.ReadCloser, int64, string, error) {
	ret := m.Called(bucket, key, offset, length)

	r0 := ret.Get(0).(io.ReadCloser)
	r1 := ret.Get(1).(int64)
	r2 := ret.Get(2).(string)
	r3 := ret.Error(3)

	return r0, r1, r2, r3
}

// Put is a mock method
func (m *Client) Put(bucket string, object string, md5 string, size int64) (io.WriteCloser, error) {
	ret := m.Called(bucket, object, md5, size)

	r0 := ret.Get(0).(io.WriteCloser)
	r1 := ret.Error(1)

	return r0, r1
}

// GetObjectMetadata is a mock method
func (m *Client) GetObjectMetadata(bucket string, object string) (*client.Item, error) {
	ret := m.Called(bucket, object)

	var r0 *client.Item
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*client.Item)
	}
	r1 := ret.Error(1)

	return r0, r1
}

// InitiateMultiPartUpload is a mock method
func (m *Client) InitiateMultiPartUpload(bucket string, object string) (string, error) {
	ret := m.Called(bucket, object)

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}

// UploadPart is a mock method
func (m *Client) UploadPart(bucket string, object string, uploadID string, partNumber int) (string, error) {
	ret := m.Called(bucket, object, uploadID, partNumber)

	r0 := ret.Get(0).(string)
	r1 := ret.Error(1)

	return r0, r1
}

// CompleteMultiPartUpload is a mock method
func (m *Client) CompleteMultiPartUpload(bucket string, object string, uploadID string) (string, string, error) {
	ret := m.Called(bucket, object, uploadID)

	r0 := ret.Get(0).(string)
	r1 := ret.Get(1).(string)
	r2 := ret.Error(2)

	return r0, r1, r2
}

// AbortMultiPartUpload is a mock method
func (m *Client) AbortMultiPartUpload(bucket string, object string, uploadID string) error {
	ret := m.Called(bucket, object, uploadID)

	r0 := ret.Error(0)

	return r0
}

// ListParts is a mock method
func (m *Client) ListParts(bucket string, object string, uploadID string) (*client.PartItems, error) {
	ret := m.Called(bucket, object, uploadID)

	var r0 *client.PartItems
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*client.PartItems)
	}
	r1 := ret.Error(1)

	return r0, r1
}
