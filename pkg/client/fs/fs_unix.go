/*
 * Mini Copy, (C) 2015 Minio, Inc.
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
	"errors"
	"io"
	"os"
	"sort"
	"strings"

	"io/ioutil"
	"net/url"
	"path/filepath"

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
func isValidObject(fpath, bucket, object string) (string, os.FileInfo, error) {
	// bucket is deliberately ignored here, since we already have
	// this path, bucket is provided just for compatibility sake
	// at this point
	if object == "" {
		return "", nil, iodine.New(client.InvalidArgument{Err: errors.New("invalid argument")}, nil)
	}
	objectPath := filepath.Join(fpath, object)
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
	objectPath, st, err := isValidObject(f.Path, bucket, object)
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
func (f *fsClient) GetPartial(bucket, object string, offset, length int64) (body io.ReadCloser, size int64, md5 string, err error) {
	if offset < 0 {
		return nil, 0, "", iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	objectPath, st, err := isValidObject(f.Path, bucket, object)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	body, err = os.Open(objectPath)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	if offset > st.Size() || (offset+length-1) > st.Size() {
		return nil, 0, "", iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	_, err = io.CopyN(ioutil.Discard, body, offset)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	return body, length, "", nil
}

// StatObject -
func (f *fsClient) GetObjectMetadata(bucket, object string) (item *client.Item, reterr error) {
	_, st, err := isValidObject(f.Path, bucket, object)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	item = new(client.Item)
	item.Key = object
	item.Size = st.Size()
	item.LastModified = st.ModTime()
	item.ETag = "" // TODO, doesn't exist yet
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
	// bucket and prefix are deliberately ignored here, since we already have
	// this path, bucket and prefix are provided just for compatibility sake at this point
	visitFS := func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) { // skip inaccessible files
				return nil
			}
			return err // fatal
		}
		if fi.IsDir() {
			return nil // not a fs skip
		}
		// trim f.Path
		item := &client.Item{
			Key:          strings.TrimPrefix(fp, f.Path+string(filepath.Separator)),
			ETag:         "", // TODO md5sum
			LastModified: fi.ModTime(),
			Size:         fi.Size(),
		}
		items = append(items, item)
		return nil
	}
	err = filepath.Walk(f.Path, visitFS)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	sort.Sort(client.BySize(items))
	return items, nil
}

// PutBucket - create a new bucket
func (f *fsClient) PutBucket(bucket string) error {
	// bucket is deliberately ignored here, since we already have
	// this path, bucket is provided just for compatibility sake
	// at this point
	bucketDir, _ := filepath.Split(f.Path)
	absPath, err := filepath.Abs(bucketDir)
	if err != nil {
		return iodine.New(err, nil)
	}
	err = os.MkdirAll(absPath, 0700)
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
	// bucket is deliberately ignored here, since we already have
	// this path, bucket is provided just for compatibility sake
	// at this point
	st, err := os.Stat(filepath.Dir(f.Path))
	if os.IsNotExist(err) {
		return iodine.New(client.BucketNotFound{Bucket: bucket}, nil)
	}
	if !st.IsDir() {
		return iodine.New(client.InvalidBucketName{Bucket: bucket}, nil)
	}
	return nil
}
