// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

/*
 * Mini Copy (C) 2015 Minio, Inc.
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
	"path/filepath"
	"strings"

	"io/ioutil"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

type fsClient struct {
	path string
}

// New - instantiate a new fs client
func New(path string) client.Client {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	return &fsClient{path: path}
}

/// Object operations
// getObjectMetadata - wrapper function to get file stat
func (f *fsClient) getObjectMetadata() (os.FileInfo, error) {
	st, err := os.Stat(filepath.Clean(f.path))
	if os.IsNotExist(err) {
		return nil, iodine.New(FileNotFound{path: f.path}, nil)
	}
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if st.IsDir() {
		return nil, iodine.New(FileISDir{path: f.path}, nil)
	}
	return st, nil
}

// Get - download an object from bucket
func (f *fsClient) Get() (body io.ReadCloser, size int64, md5 string, err error) {
	st, err := f.getObjectMetadata()
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	body, err = os.Open(f.path)
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
	body, err = os.Open(f.path)
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

func (f *fsClient) List() <-chan client.ItemOnChannel {
	itemCh := make(chan client.ItemOnChannel)
	go f.listInGoroutine(itemCh)
	return itemCh
}

func (f *fsClient) listInGoroutine(itemCh chan client.ItemOnChannel) {
	defer close(itemCh)
	visitFS := func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) { // skip inaccessible files
				return nil
			}
			return err // fatal
		}
		item := &client.Item{
			Name:     fp,
			Time:     fi.ModTime(),
			Size:     fi.Size(),
			FileType: fi.Mode(),
		}
		itemCh <- client.ItemOnChannel{
			Item: item,
			Err:  nil,
		}
		return nil
	}
	err := filepath.Walk(f.path, visitFS)
	if err != nil {
		itemCh <- client.ItemOnChannel{
			Item: nil,
			Err:  iodine.New(err, nil),
		}
	}
}

// PutBucket - create a new bucket
func (f *fsClient) PutBucket(acl string) error {
	err := os.MkdirAll(f.path, 0700)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// Stat -
func (f *fsClient) Stat() error {
	st, err := os.Stat(f.path)
	if os.IsNotExist(err) {
		return iodine.New(err, nil)
	}
	if !st.IsDir() {
		return iodine.New(FileNotDir{path: f.path}, nil)
	}
	return nil
}
