/*
 * Modern Copy, (C) 2015 Minio, Inc.
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
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

type fsClient struct {
	*url.URL
}

// GetNewClient - instantiate a new fs client
func GetNewClient(path string) client.Client {
	u, err := url.Parse(path)
	if err != nil {
		return nil
	}
	return &fsClient{u}
}

/// Object operations

// isValidObject - wrapper function for input validation
func isValidObject(bucket, object string) (string, os.FileInfo, error) {
	if bucket == "" || object == "" {
		return "", nil, iodine.New(client.InvalidArgument{}, nil)
	}
	objectPath := filepath.Join(bucket, object)
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

// Get - download an object from bucket
func (f *fsClient) Get(bucket, object string) (body io.ReadCloser, size int64, md5 string, err error) {
	objectPath, st, err := isValidObject(bucket, object)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	body, err = os.Open(objectPath)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	// TODO: support md5sum - there is no easier way to do it right now without temporary buffer
	// so avoiding it to ensure no out of memory situations
	return body, st.Size(), "", nil
}

// GetPartial - download a partial object from bucket
func (f *fsClient) GetPartial(bucket, key string, offset, length int64) (body io.ReadCloser, size int64, md5 string, err error) {
	if offset < 0 {
		return nil, 0, "", iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	return nil, 0, "", iodine.New(client.APINotImplemented{API: "GetPartial"}, nil)
}

// StatObject -
func (f *fsClient) GetObjectMetadata(bucket, object string) (item *client.Item, reterr error) {
	if bucket == "" || object == "" {
		return nil, iodine.New(client.InvalidArgument{}, nil)
	}
	_, st, err := isValidObject(bucket, object)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	item = new(client.Item)
	item.Key = object
	item.Size = st.Size()
	item.LastModified = st.ModTime()
	item.ETag = "" // TODO
	return item, nil
}

/// Bucket operations

// ListBuckets - get list of buckets
func (f *fsClient) ListBuckets() ([]*client.Bucket, error) {
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
func (f *fsClient) ListObjects(bucket, prefix string) (items []*client.Item, err error) {
	visitFS := func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return nil
			}
			return err // fatal
		}
		if fi.IsDir() {
			return nil // not a fs skip
		}
		// If bucket path is not absolute, trim it
		// otherwise pass it down as is
		item := &client.Item{
			Key:          strings.TrimPrefix(fp, bucket),
			ETag:         "", // TODO
			LastModified: fi.ModTime(),
			Size:         fi.Size(),
		}
		items = append(items, item)
		return nil
	}
	err = filepath.Walk(filepath.Join(bucket, prefix), visitFS)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	sort.Sort(client.BySize(items))
	return items, nil
}

// PutBucket - create a new bucket
func (f *fsClient) PutBucket(bucket string) error {
	if bucket == "" || strings.TrimSpace(bucket) == "" {
		return iodine.New(client.InvalidArgument{}, nil)
	}
	err := os.MkdirAll(bucket, 0700)
	if os.IsExist(err) {
		return iodine.New(client.BucketExists{Bucket: bucket}, nil)
	}
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// StatBucket -
func (f *fsClient) StatBucket(bucket string) error {
	if bucket == "" {
		return iodine.New(client.InvalidArgument{}, nil)
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
