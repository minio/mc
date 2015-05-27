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
func New(path string) (client.Client, error) {
	if strings.TrimSpace(path) == "" {
		return nil, iodine.New(errors.New("Path is empty."), nil)
	}

	return &fsClient{path: path}, nil
}

/// Object operations

// fsStat - wrapper function to get file stat
func (f *fsClient) fsStat() (os.FileInfo, error) {
	fpath := f.path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(fpath, string(filepath.Separator)) {
		fpath = fpath + "."
	}

	// Resolve symlinks
	fpath, err := filepath.EvalSymlinks(fpath)
	if os.IsNotExist(err) {
		return nil, iodine.New(NotFound{path: f.path}, nil)
	}
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	st, err := os.Stat(fpath)
	if os.IsNotExist(err) {
		return nil, iodine.New(NotFound{path: fpath}, nil)
	}
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return st, nil
}

// get - download an object from bucket
func (f *fsClient) get(content *client.Content) (io.ReadCloser, uint64, error) {
	body, err := os.Open(f.path)
	if err != nil {
		return nil, 0, iodine.New(err, nil)
	}
	return body, uint64(content.Size), nil
}

// GetObject download an full or part object from bucket
func (f *fsClient) GetObject(offset, length uint64) (io.ReadCloser, uint64, error) {
	content, err := f.getFSMetadata()
	if err != nil {
		return nil, 0, iodine.New(err, nil)
	}
	if content.Type.IsDir() {
		return nil, 0, iodine.New(ISFolder{path: f.path}, nil)
	}
	if int64(offset) > content.Size || int64(offset+length-1) > content.Size {
		return nil, 0, iodine.New(client.InvalidRange{Offset: offset}, nil)
	}

	fpath := f.path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(fpath, string(filepath.Separator)) {
		fpath = fpath + "."
	}

	// Resolve symlinks
	fpath, err = filepath.EvalSymlinks(fpath)
	if os.IsNotExist(err) {
		return nil, 0, iodine.New(NotFound{path: f.path}, nil)
	}
	if offset == 0 && length == 0 {
		return f.get(content)
	}
	body, err := os.Open(f.path)
	if err != nil {
		return nil, 0, iodine.New(err, nil)

	}
	_, err = io.CopyN(ioutil.Discard, body, int64(offset))
	if err != nil {
		return nil, 0, iodine.New(err, nil)
	}
	return body, length, nil
}

// List - list files and folders
func (f *fsClient) List(recursive bool) <-chan client.ContentOnChannel {
	contentCh := make(chan client.ContentOnChannel)
	switch recursive {
	case true:
		go f.listRecursiveInRoutine(contentCh)
	default:
		go f.listInRoutine(contentCh)
	}
	return contentCh
}

func (f *fsClient) listInRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)

	fpath := f.path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(fpath, string(filepath.Separator)) {
		fpath = fpath + "."
	}

	// Resolve symlinks
	fpath, err := filepath.EvalSymlinks(fpath)
	if os.IsNotExist(err) {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     iodine.New(NotFound{path: f.path}, nil),
		}
		return
	}
	if err != nil {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     iodine.New(err, nil),
		}
		return
	}

	dir, err := os.Open(fpath)
	if err != nil {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     iodine.New(err, nil),
		}
		return
	}
	defer dir.Close()

	fi, err := os.Stat(fpath)
	if err != nil {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     iodine.New(err, nil),
		}
		return
	}
	switch fi.Mode().IsDir() {
	case true:
		// do not use ioutil.ReadDir(), since it tries to sort its
		// output at our scale we are expecting that to slow down
		// instead we take raw output and provide it back to the
		// user - this is the correct style when are moving large
		// quantities of files
		files, err := dir.Readdir(-1)
		if err != nil {
			contentCh <- client.ContentOnChannel{
				Content: nil,
				Err:     iodine.New(err, nil),
			}
			return
		}
		for _, file := range files {
			var fi os.FileInfo
			fi = file
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				fi, err = os.Stat(filepath.Join(dir.Name(), file.Name()))
				if err != nil {
					contentCh <- client.ContentOnChannel{
						Content: nil,
						Err:     iodine.New(err, nil),
					}
					return
				}
			}
			if fi.Mode().IsRegular() || fi.Mode().IsDir() {
				content := &client.Content{
					Name: fi.Name(),
					Time: fi.ModTime(),
					Size: fi.Size(),
					Type: fi.Mode(),
				}
				contentCh <- client.ContentOnChannel{
					Content: content,
					Err:     nil,
				}
			}
		}
	default:
		content := &client.Content{
			Name: f.path,
			Time: fi.ModTime(),
			Size: fi.Size(),
			Type: fi.Mode(),
		}
		contentCh <- client.ContentOnChannel{
			Content: content,
			Err:     nil,
		}
	}
}

func (f *fsClient) listRecursiveInRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	visitFS := func(fp string, fi os.FileInfo, err error) error {
		// fp also sends back itself with visitFS, ignore it we don't need it
		if fp == f.path {
			return nil
		}
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			fi, err = os.Stat(fp)
			if err != nil {
				if os.IsNotExist(err) { // ignore broken symlinks
					return nil
				}
				return iodine.New(err, nil)
			}
		}
		if fi.Mode().IsRegular() || fi.Mode().IsDir() {
			if err != nil {
				if strings.Contains(err.Error(), "operation not permitted") {
					return nil
				}
				if os.IsPermission(err) {
					return nil
				}
				return iodine.New(err, nil) // abort
			}
			content := &client.Content{
				Name: fp,
				Time: fi.ModTime(),
				Size: fi.Size(),
				Type: fi.Mode(),
			}
			contentCh <- client.ContentOnChannel{
				Content: content,
				Err:     nil,
			}
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

// MakeBucket - create a new bucket
func (f *fsClient) MakeBucket() error {
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
	content.Name = f.path
	content.Size = st.Size()
	content.Time = st.ModTime()
	content.Type = st.Mode()
	return content, nil
}

// Stat - get metadata from path
func (f *fsClient) Stat() (content *client.Content, err error) {
	return f.getFSMetadata()
}
