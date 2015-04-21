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

// New - instantiate a new fs client
func New(path string) client.Client {
	u, err := url.Parse(path)
	if err != nil {
		return nil
	}
	return &fsClient{u}
}

/// Object operations

// url2Object converts URL to bucketName and objectName
func (f *fsClient) url2Object() (bucketName, objectName string) {
	unescapedURL, _ := url.QueryUnescape(f.String())
	return filepath.Split(unescapedURL)
}

// getObjectMetadata - wrapper function to get file stat
func (f *fsClient) getObjectMetadata() (os.FileInfo, error) {
	bucket, object := f.url2Object()
	st, err := os.Stat(f.normalizedPath())
	if os.IsNotExist(err) {
		return nil, iodine.New(client.ObjectNotFound{Bucket: bucket, Object: object}, nil)
	}
	if st.IsDir() {
		return nil, iodine.New(client.InvalidObjectName{Bucket: bucket, Object: object}, nil)
	}
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return st, nil
}

func (f *fsClient) normalizedPath() string {
	unescapedURL, _ := url.QueryUnescape(f.String())
	return filepath.Clean(unescapedURL)
}

// Get - download an object from bucket
func (f *fsClient) Get() (body io.ReadCloser, size int64, md5 string, err error) {
	st, err := f.getObjectMetadata()
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	body, err = os.Open(f.normalizedPath())
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	// TODO: support md5sum - there is no easier way to do it right now without temporary buffer
	// so avoiding it to ensure no out of memory situations
	return body, st.Size(), "", nil
}

// GetPartial - download a partial object from bucket
func (f *fsClient) GetPartial(offset, length int64) (body io.ReadCloser, size int64, md5 string, err error) {
	if offset < 0 {
		return nil, 0, "", iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	st, err := f.getObjectMetadata()
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	body, err = os.Open(f.normalizedPath())
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

// GetObjectMetadata -
func (f *fsClient) GetObjectMetadata() (item *client.Item, reterr error) {
	st, err := f.getObjectMetadata()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	item = new(client.Item)
	item.Name = st.Name()
	item.Size = st.Size()
	item.Time = st.ModTime()
	return item, nil
}

/// Bucket operations

// listBuckets - get list of buckets
func (f *fsClient) listBuckets() ([]*client.Item, error) {
	buckets, err := ioutil.ReadDir(f.normalizedPath())
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	var results []*client.Item
	for _, bucket := range buckets {
		result := new(client.Item)
		result.Name = bucket.Name()
		result.Time = bucket.ModTime()
		results = append(results, result)
	}
	return results, nil
}

// List - get a list of items
func (f *fsClient) List() (items []*client.Item, err error) {
	item, err := f.GetObjectMetadata()
	switch err {
	case nil:
		items = append(items, item)
		return items, nil
	default:
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
			// trim f.String()
			item := &client.Item{
				Name: strings.TrimPrefix(filepath.Clean(fp), f.normalizedPath()+string(filepath.Separator)),
				Time: fi.ModTime(),
				Size: fi.Size(),
			}
			items = append(items, item)
			return nil
		}
		err = filepath.Walk(f.normalizedPath(), visitFS)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
		sort.Sort(client.BySize(items))
		return items, nil
	}
}

// PutBucket - create a new bucket
func (f *fsClient) PutBucket() error {
	err := os.MkdirAll(f.normalizedPath(), 0700)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// Stat -
func (f *fsClient) Stat() error {
	st, err := os.Stat(f.normalizedPath())
	if os.IsNotExist(err) {
		return iodine.New(client.BucketNotFound{Bucket: f.normalizedPath()}, nil)
	}
	if !st.IsDir() {
		return iodine.New(client.InvalidBucketName{Bucket: f.normalizedPath()}, nil)
	}
	return nil
}
