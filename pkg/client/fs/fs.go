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
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"io/ioutil"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/iodine"
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

// fsStat - wrapper function to get file stat
func (f *fsClient) fsStat() (os.FileInfo, error) {
	st, err := os.Lstat(filepath.Clean(f.path))
	if os.IsNotExist(err) {
		return nil, iodine.New(NotFound{path: f.path}, nil)
	}
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return st, nil
}

func (f *fsClient) get(content *client.Content) (io.ReadCloser, uint64, string, error) {
	body, err := os.Open(f.path)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	h := md5.New()
	// calculate md5sum
	_, err = io.Copy(h, body)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	// seek back
	_, err = body.Seek(0, 0)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	md5Str := hex.EncodeToString(h.Sum(nil))
	return body, uint64(content.Size), md5Str, nil
}

// GetObject download an full or part object from bucket
func (f *fsClient) GetObject(offset, length uint64) (io.ReadCloser, uint64, string, error) {
	content, err := f.getFSMetadata()
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	if content.FileType.IsDir() {
		return nil, 0, "", iodine.New(ISFolder{path: f.path}, nil)
	}
	if int64(offset) > content.Size || int64(offset+length-1) > content.Size {
		return nil, 0, "", iodine.New(client.InvalidRange{Offset: offset}, nil)
	}
	if offset == 0 && length == 0 {
		return f.get(content)
	}
	body, err := os.Open(f.path)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	_, err = io.CopyN(ioutil.Discard, body, int64(offset))
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	h := md5.New()
	// calculate md5sum
	_, err = io.Copy(h, body)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	// seek back
	_, err = body.Seek(0, 0)
	if err != nil {
		return nil, 0, "", iodine.New(err, nil)
	}
	md5Str := hex.EncodeToString(h.Sum(nil))
	return body, length, md5Str, nil
}

// List - list files and folders
func (f *fsClient) List() <-chan client.ContentOnChannel {
	contentCh := make(chan client.ContentOnChannel)
	go f.list(contentCh)
	return contentCh
}

func (f *fsClient) list(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	dir, err := os.Open(f.path)
	if err != nil {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     iodine.New(err, nil),
		}
	}
	fi, err := os.Lstat(f.path)
	if err != nil {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     iodine.New(err, nil),
		}
	}
	defer dir.Close()
	switch fi.Mode().IsDir() {
	case true:
		// do not use ioutil.ReadDir(), since it tries to sort its
		// output at our scale we are expecting that to slow down
		// instead we take raw output and provide it back to the user
		// - such a thing is helpful when we are moving in and out
		// large quantities of files
		files, err := dir.Readdir(-1)
		if err != nil {
			contentCh <- client.ContentOnChannel{
				Content: nil,
				Err:     iodine.New(err, nil),
			}
		}
		for _, file := range files {
			content := &client.Content{
				Name:     file.Name(),
				Time:     file.ModTime(),
				Size:     file.Size(),
				FileType: file.Mode(),
			}
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
		}
	default:
		content := &client.Content{
			Name:     dir.Name(),
			Time:     fi.ModTime(),
			Size:     fi.Size(),
			FileType: fi.Mode(),
		}
		contentCh <- client.ContentOnChannel{
			Content: content,
			Err:     nil,
		}
	}
}

// ListRecursive - list files and folders recursively
func (f *fsClient) ListRecursive() <-chan client.ContentOnChannel {
	contentCh := make(chan client.ContentOnChannel)
	go f.listRecursive(contentCh)
	return contentCh
}

func (f *fsClient) listRecursive(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	visitFS := func(fp string, fi os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) { // skip inaccessible files
				return nil
			}
			return err // fatal
		}
		content := &client.Content{
			Name:     fp,
			Time:     fi.ModTime(),
			Size:     fi.Size(),
			FileType: fi.Mode(),
		}
		contentCh <- client.ContentOnChannel{
			Content: content,
			Err:     nil,
		}
		return nil
	}
	err := filepath.Walk(f.path, visitFS)
	if err != nil {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     iodine.New(err, nil),
		}
	}
}

// isValidBucketACL - is acl a valid ACL?
func isValidBucketACL(acl string) bool {
	switch acl {
	case "private":
		fallthrough
	case "public-read":
		fallthrough
	case "public-read-write":
		fallthrough
	case "authenticated-read":
		fallthrough
	case "":
		return true
	default:
		return false
	}
}

// aclToPerm - convert acl to filesystem mode
func aclToPerm(acl string) os.FileMode {
	switch acl {
	case "private":
		return os.FileMode(0700)
	case "public-read":
		return os.FileMode(0500)
	case "public-read-write":
		return os.FileMode(0777)
	case "authenticated-read":
		return os.FileMode(0770)
	default:
		return os.FileMode(0700)
	}
}

// CreateBucket - create a new bucket
func (f *fsClient) CreateBucket() error {
	err := os.MkdirAll(f.path, 0775)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// SetBucketACL - create a new bucket
func (f *fsClient) SetBucketACL(acl string) error {
	if !isValidBucketACL(acl) {
		return iodine.New(errors.New("invalid acl"), nil)
	}
	err := os.MkdirAll(f.path, aclToPerm(acl))
	if err != nil {
		return iodine.New(err, nil)
	}
	err = os.Chmod(f.path, aclToPerm(acl))
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// getFSMetadata -
func (f *fsClient) getFSMetadata() (content *client.Content, err error) {
	st, err := f.fsStat()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	content = new(client.Content)
	content.Name = st.Name()
	content.Size = st.Size()
	content.Time = st.ModTime()
	return content, nil
}

// Stat - get metadata from path
func (f *fsClient) Stat() (content *client.Content, err error) {
	return f.getFSMetadata()
}
