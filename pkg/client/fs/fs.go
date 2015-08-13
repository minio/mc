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
	"os"
	"path/filepath"
	"strings"
	"time"

	"io/ioutil"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/probe"
)

type fsClient struct {
	path string
}

// New - instantiate a new fs client
func New(path string) (client.Client, *probe.Error) {
	if strings.TrimSpace(path) == "" {
		return nil, probe.NewError(client.EmptyPath{})
	}
	return &fsClient{path: normalizePath(path)}, nil
}

// URL get url
func (f *fsClient) URL() *client.URL {
	url, _ := client.Parse(f.path)
	return url
}

/// Object operations

// fsStat - wrapper function to get file stat
func (f *fsClient) fsStat() (os.FileInfo, *probe.Error) {
	fpath := f.path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(fpath, string(f.URL().Separator)) {
		fpath = fpath + "."
	}
	// Resolve symlinks
	fpath, err := filepath.EvalSymlinks(fpath)
	if os.IsNotExist(err) {
		return nil, probe.NewError(err)
	}
	if err != nil {
		return nil, probe.NewError(err)
	}
	st, err := os.Stat(fpath)
	if os.IsNotExist(err) {
		return nil, probe.NewError(client.NotFound{Path: fpath})
	}
	if err != nil {
		return nil, probe.NewError(err)
	}
	return st, nil
}

// PutObject - create a new file
func (f *fsClient) PutObject(size int64, data io.Reader) *probe.Error {
	objectDir, _ := filepath.Split(f.path)
	objectPath := f.path
	if objectDir != "" {
		if err := os.MkdirAll(objectDir, 0700); err != nil {
			return probe.NewError(err)
		}
	}
	fs, err := os.Create(objectPath)
	if err != nil {
		return probe.NewError(err)
	}
	defer fs.Close()

	// even if size is zero try to read from source
	if size > 0 {
		_, err = io.CopyN(fs, data, int64(size))
		if err != nil {
			return probe.NewError(err)
		}
	} else {
		// size could be 0 for virtual files on certain filesystems
		// for example /proc, so read till EOF for such files
		_, err = io.Copy(fs, data)
		if err != nil {
			return probe.NewError(err)
		}
	}
	return nil
}

// get - download an object from bucket
func (f *fsClient) get() (io.ReadCloser, int64, *probe.Error) {
	body, err := os.Open(f.path)
	if err != nil {
		return nil, 0, probe.NewError(err)
	}
	{
		content, err := f.getFSMetadata()
		if err != nil {
			return nil, content.Size, err.Trace()
		}
		return body, content.Size, nil
	}
}

func (f *fsClient) Share(expires time.Duration, offset, length int64) (string, *probe.Error) {
	return "", probe.NewError(client.APINotImplemented{API: "Share", APIType: "filesystem"})
}

// GetObject download an full or part object from bucket
// getobject returns a reader, length and nil for no errors
// with errors getobject will return nil reader, length and typed errors
func (f *fsClient) GetObject(offset, length int64) (io.ReadCloser, int64, *probe.Error) {
	if offset < 0 || length < 0 {
		return nil, 0, probe.NewError(client.InvalidRange{Offset: offset})
	}

	tmppath := f.path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(tmppath, string(f.URL().Separator)) {
		tmppath = tmppath + "."
	}

	// Resolve symlinks
	_, err := filepath.EvalSymlinks(tmppath)
	if os.IsNotExist(err) {
		return nil, length, probe.NewError(err)
	}
	if err != nil {
		return nil, length, probe.NewError(err)
	}
	if offset == 0 && length == 0 {
		return f.get()
	}
	body, err := os.Open(f.path)
	if err != nil {
		return nil, length, probe.NewError(err)

	}
	_, err = io.CopyN(ioutil.Discard, body, int64(offset))
	if err != nil {
		return nil, length, probe.NewError(err)
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
	if strings.HasSuffix(fpath, string(f.URL().Separator)) {
		fpath = fpath + "."
	}

	// Resolve symlinks
	fpath, err := filepath.EvalSymlinks(fpath)
	if os.IsNotExist(err) {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     probe.NewError(err),
		}
		return
	}
	if err != nil {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     probe.NewError(err),
		}
		return
	}

	fi, err := os.Stat(fpath)
	if err != nil {
		contentCh <- client.ContentOnChannel{
			Content: nil,
			Err:     probe.NewError(err),
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
		dir, err := os.Open(fpath)
		if err != nil {
			contentCh <- client.ContentOnChannel{
				Content: nil,
				Err:     probe.NewError(err),
			}
			return
		}
		defer dir.Close()

		files, err := dir.Readdir(-1)
		if err != nil {
			contentCh <- client.ContentOnChannel{
				Content: nil,
				Err:     probe.NewError(err),
			}
			return
		}
		for _, file := range files {
			fi := file
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				fi, err = os.Stat(filepath.Join(dir.Name(), fi.Name()))
				if os.IsNotExist(err) {
					contentCh <- client.ContentOnChannel{
						Content: nil,
						Err:     probe.NewError(client.ISBrokenSymlink{Path: file.Name()}),
					}
					continue
				}
				if err != nil {
					contentCh <- client.ContentOnChannel{
						Content: nil,
						Err:     probe.NewError(err),
					}
					continue
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

func (f *fsClient) delimited(fp string) string {
	var stripPrefix string
	stripPrefix = f.path[:strings.LastIndex(f.path, string(f.URL().Separator))+1]
	return strings.TrimPrefix(fp, stripPrefix)
}

func (f *fsClient) listRecursiveInRoutine(contentCh chan client.ContentOnChannel) {
	defer close(contentCh)
	visitFS := func(fp string, fi os.FileInfo, err error) error {
		// fp also sends back itself with visitFS, ignore it we don't need it
		if fp == f.path {
			return nil
		}
		if err != nil {
			if strings.Contains(err.Error(), "operation not permitted") {
				return nil
			}
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			fi, err = os.Stat(fp)
			if err != nil {
				if os.IsNotExist(err) || os.IsPermission(err) { // ignore broken symlinks and permission denied
					return nil
				}
				return err
			}
		}
		if fi.Mode().IsRegular() || fi.Mode().IsDir() {
			content := &client.Content{
				Name: f.delimited(fp),
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
			Err:     probe.NewError(err),
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
func (f *fsClient) MakeBucket() *probe.Error {
	err := os.MkdirAll(f.path, 0775)
	if err != nil {
		return probe.NewError(err)
	}
	return nil
}

// SetBucketACL - create a new bucket
func (f *fsClient) SetBucketACL(acl string) *probe.Error {
	if !isValidBucketACL(acl) {
		return probe.NewError(client.InvalidACLType{ACL: acl})
	}
	err := os.MkdirAll(f.path, aclToPerm(acl))
	if err != nil {
		return probe.NewError(err)
	}
	err = os.Chmod(f.path, aclToPerm(acl))
	if err != nil {
		return probe.NewError(err)
	}
	return nil
}

// getFSMetadata -
func (f *fsClient) getFSMetadata() (content *client.Content, err *probe.Error) {
	st, err := f.fsStat()
	if err != nil {
		return nil, err.Trace()
	}
	content = new(client.Content)
	content.Name = f.path
	content.Size = st.Size()
	content.Time = st.ModTime()
	content.Type = st.Mode()
	return content, nil
}

// Stat - get metadata from path
func (f *fsClient) Stat() (content *client.Content, err *probe.Error) {
	return f.getFSMetadata()
}
