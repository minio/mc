/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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

package file

import (
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

type fileClient struct {
	// Supports URL in following formats
	//  - http://<ipaddress>/<bucketname>/<object>
	//  - http://<bucketname>.<domain>/<object>
	*url.URL
}

// GetNewClient -
func GetNewClient(path string) client.Client {
	u, err := url.Parse(path)
	if err != nil {
		return nil
	}
	return &fileClient{u}
}

/// Object operations

// isValidObject - wrapper function for input validation
func isValidObject(bucket, object string) (string, os.FileInfo, error) {
	if bucket == "" || object == "" {
		return "", nil, iodine.New(client.InvalidArgument{}, nil)
	}
	objectPath := path.Join(bucket, object)
	st, err := os.Stat(objectPath)
	if os.IsNotExist(err) {
		return "", nil, iodine.New(client.ObjectNotFound{Bucket: bucket, Object: object}, nil)
	}
	if st.IsDir() {
		return "", nil, iodine.New(client.InvalidObjectName{Bucket: bucket, Object: object}, nil)
	}
	if err != nil {
		return "", nil, iodine.New(err, nil)
	}
	return objectPath, st, nil
}

// Put - upload new object to bucket
func (f *fileClient) Put(bucket, object, md5HexString string, size int64, contents io.Reader) error {
	// handle md5HexString match internally
	if bucket == "" || object == "" {
		return iodine.New(client.InvalidArgument{}, nil)
	}
	objectPath := path.Join(bucket, object)
	if size < 0 {
		return iodine.New(client.InvalidArgument{}, nil)
	}
	file, err := os.OpenFile(objectPath, os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return iodine.New(client.ObjectExists{Bucket: bucket, Object: object}, nil)
	}
	if err != nil {
		return iodine.New(err, nil)
	}
	_, err = io.CopyN(file, contents, size)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// Get - download an object from bucket
func (f *fileClient) Get(bucket, object string) (body io.ReadCloser, size int64, md5 string, err error) {
	objectPath, st, err := isValidObject(bucket, object)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	body, err = os.OpenFile(objectPath, os.O_RDONLY, 0600)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	// TODO: support md5sum - there is no easier way to do it right now without temporary buffer
	// so avoiding it to ensure no out of memory situations
	return body, st.Size(), "", nil
}

// GetPartial - download a partial object from bucket
func (f *fileClient) GetPartial(bucket, key string, offset, length int64) (body io.ReadCloser, size int64, md5 string, err error) {
	if offset < 0 {
		return nil, 0, "", iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	return nil, 0, "", iodine.New(client.APINotImplemented{API: "GetPartial"}, nil)
}

// StatObject -
func (f *fileClient) StatObject(bucket, object string) (size int64, date time.Time, reterr error) {
	_, st, err := isValidObject(bucket, object)
	if size < 0 {
		return 0, date, iodine.New(client.InvalidArgument{}, nil)
	}
	if err != nil {
		return 0, date, iodine.New(err, nil)
	}
	return st.Size(), st.ModTime(), nil
}

/// Bucket operations

// ListBuckets - get list of buckets
func (f *fileClient) ListBuckets() ([]*client.Bucket, error) {
	buckets, err := ioutil.ReadDir(f.Path)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	var results []*client.Bucket
	for _, bucket := range buckets {
		result := &client.Bucket{
			Name:         bucket.Name(),
			CreationDate: bucket.ModTime(), // no easier way on Linux
		}
		results = append(results, result)
	}
	return results, nil
}

// ListObjects - get a list of objects
func (f *fileClient) ListObjects(bucket, keyPrefix string) (items []*client.Item, err error) {
	return nil, iodine.New(client.APINotImplemented{API: "ListObjects"}, nil)
}

// PutBucket - create a new bucket
func (f *fileClient) PutBucket(bucket string) error {
	if bucket == "" || strings.TrimSpace(bucket) == "" {
		return iodine.New(client.InvalidArgument{}, nil)
	}
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(client.InvalidBucketName{Bucket: bucket}, nil)
	}
	err := os.Mkdir(bucket, os.ModeDir)
	if os.IsExist(err) {
		return iodine.New(client.BucketExists{Bucket: bucket}, nil)
	}
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// StatBucket -
func (f *fileClient) StatBucket(bucket string) error {
	if bucket == "" {
		return iodine.New(client.InvalidArgument{}, nil)
	}
	if !client.IsValidBucketName(bucket) || strings.Contains(bucket, ".") {
		return iodine.New(client.InvalidBucketName{Bucket: bucket}, nil)
	}
	st, err := os.Stat(bucket)
	if os.IsNotExist(err) {
		return iodine.New(client.BucketNotFound{Bucket: bucket}, nil)
	}
	if !st.IsDir() {
		return iodine.New(client.InvalidBucketName{Bucket: bucket}, nil)
	}
	return nil
}
