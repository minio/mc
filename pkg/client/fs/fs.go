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
	"runtime"
	"strings"
	"time"

	"io/ioutil"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio-xl/pkg/probe"
)

// filesystem client
type fsClient struct {
	PathURL *client.URL
}

// New - instantiate a new fs client.
func New(path string) (client.Client, *probe.Error) {
	if strings.TrimSpace(path) == "" {
		return nil, probe.NewError(client.EmptyPath{})
	}
	return &fsClient{
		PathURL: client.NewURL(normalizePath(path)),
	}, nil
}

// URL get url.
func (f *fsClient) GetURL() client.URL {
	return *f.PathURL
}

/// Object operations.

// fsStat - wrapper function to get file stat.
func (f *fsClient) fsStat() (os.FileInfo, *probe.Error) {
	fpath := f.PathURL.Path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(fpath, string(f.PathURL.Separator)) {
		fpath = fpath + "."
	}
	// Resolve symlinks.
	fpath, err := filepath.EvalSymlinks(fpath)
	if err != nil {
		if os.IsPermission(err) {
			if runtime.GOOS == "windows" {
				// On windows there are directory symlinks which are called junction files.
				// These files carry special meaning on windows they cannot be,
				// accessed with regular operations.
				lfi, lerr := os.Lstat(fpath)
				if lerr != nil {
					if os.IsPermission(lerr) {
						return nil, probe.NewError(client.PathInsufficientPermission{Path: fpath})
					}
					return nil, probe.NewError(lerr)
				}
				return lfi, nil
			}
		}
		if os.IsNotExist(err) {
			return nil, probe.NewError(client.PathNotFound{Path: f.PathURL.Path})
		}
		if os.IsPermission(err) {
			return nil, probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
		}
		return nil, probe.NewError(err)
	}
	st, err := os.Stat(fpath)
	if err != nil {
		if os.IsPermission(err) {
			if runtime.GOOS == "windows" {
				// On windows there are directory symlinks which are called junction files.
				// These files carry special meaning on windows they cannot be,
				// accessed with regular operations.
				lst, lerr := os.Lstat(fpath)
				if lerr != nil {
					if os.IsPermission(lerr) {
						return nil, probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
					}
					return nil, probe.NewError(lerr)
				}
				return lst, nil
			}
			return nil, probe.NewError(err)
		}
		if os.IsNotExist(err) {
			return nil, probe.NewError(client.PathNotFound{Path: f.PathURL.Path})
		}
		if os.IsPermission(err) {
			return nil, probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
		}
		return nil, probe.NewError(err)
	}
	return st, nil
}

// Put - create a new file.
func (f *fsClient) Put(size int64, data io.Reader) *probe.Error {
	objectDir, _ := filepath.Split(f.PathURL.Path)
	objectPath := f.PathURL.Path
	if objectDir != "" {
		// Create any missing top level directories.
		if err := os.MkdirAll(objectDir, 0700); err != nil {
			if os.IsPermission(err) {
				return probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
			}
			return probe.NewError(err)
		}
	}
	fs, err := os.Create(objectPath)
	if err != nil {
		if os.IsPermission(err) {
			return probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
		}
		return probe.NewError(err)
	}
	defer fs.Close()

	// even if size is zero try to read from source.
	if size > 0 {
		_, err = io.CopyN(fs, data, int64(size))
		if err != nil {
			return probe.NewError(err)
		}
	} else {
		// Size could be 0 for virtual files on certain filesystems,
		// for example /proc, so read till EOF for such files.
		_, err = io.Copy(fs, data)
		if err != nil {
			return probe.NewError(err)
		}
	}
	return nil
}

// ShareDownload - share download not implemented for filesystem.
func (f *fsClient) ShareDownload(expires time.Duration) (string, *probe.Error) {
	return "", probe.NewError(client.APINotImplemented{API: "ShareDownload", APIType: "filesystem"})
}

// ShareUpload - share upload not implemented for filesystem.
func (f *fsClient) ShareUpload(recursive bool, expires time.Duration, contentType string) (map[string]string, *probe.Error) {
	return nil, probe.NewError(client.APINotImplemented{API: "ShareUpload", APIType: "filesystem"})
}

// get - convenience wrapper.
func (f *fsClient) get() (io.ReadCloser, int64, *probe.Error) {
	body, e := os.Open(f.PathURL.Path)
	if e != nil {
		if os.IsPermission(e) {
			return nil, 0, probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
		}
		return nil, 0, probe.NewError(e)
	}
	content, err := f.getFSMetadata()
	if err != nil {
		return nil, content.Size, err.Trace(f.PathURL.Path)
	}
	return body, content.Size, nil
}

// Get download an full or part object from bucket.
// returns a reader, length and nil for no errors.
func (f *fsClient) Get(offset, length int64) (io.ReadCloser, int64, *probe.Error) {
	if offset < 0 || length < 0 {
		return nil, 0, probe.NewError(client.InvalidRange{Offset: offset})
	}

	tmppath := f.PathURL.Path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(tmppath, string(f.PathURL.Separator)) {
		tmppath = tmppath + "."
	}

	// Resolve symlinks.
	_, err := filepath.EvalSymlinks(tmppath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, length, probe.NewError(client.PathNotFound{Path: f.PathURL.Path})
		}
		if os.IsPermission(err) {
			return nil, length, probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
		}
		return nil, length, probe.NewError(err)
	}
	if offset == 0 && length == 0 {
		return f.get()
	}
	body, err := os.Open(f.PathURL.Path)
	if err != nil {
		return nil, length, probe.NewError(err)

	}
	_, err = io.CopyN(ioutil.Discard, body, int64(offset))
	if err != nil {
		return nil, length, probe.NewError(err)
	}
	return body, length, nil
}

// Remove - remove the path.
func (f *fsClient) Remove(incomplete bool) *probe.Error {
	if incomplete {
		return nil
	}
	err := os.Remove(f.PathURL.Path)
	return probe.NewError(err)
}

// List - list files and folders.
func (f *fsClient) List(recursive, incomplete bool) <-chan *client.Content {
	contentCh := make(chan *client.Content)
	if incomplete {
		go func() {
			defer close(contentCh)
		}()
		return contentCh
	}
	switch recursive {
	case true:
		go f.listRecursiveInRoutine(contentCh)
	default:
		go f.listInRoutine(contentCh)
	}
	return contentCh
}

// listPrefixes - list all files for any given prefix.
func (f *fsClient) listPrefixes(prefix string, contentCh chan<- *client.Content) {
	dirName := filepath.Dir(prefix)
	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		contentCh <- &client.Content{Err: probe.NewError(err)}
		return
	}
	pathURL := *f.PathURL
	for _, fi := range files {
		file := filepath.Join(dirName, fi.Name())
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			st, err := os.Stat(file)
			if err != nil {
				if os.IsPermission(err) {
					// On windows there are folder symlinks
					// which are called junction files which
					// carry special meaning on windows
					// - which cannot be accessed with regular operations
					if runtime.GOOS == "windows" {
						newPath := filepath.Join(prefix, fi.Name())
						lfi, lerr := os.Lstat(newPath)
						if lerr != nil {
							if os.IsPermission(lerr) {
								contentCh <- &client.Content{
									Err: probe.NewError(client.PathInsufficientPermission{Path: newPath}),
								}
								continue
							}
							if os.IsNotExist(lerr) {
								contentCh <- &client.Content{
									Err: probe.NewError(client.BrokenSymlink{Path: pathURL.Path}),
								}
								continue
							}
							contentCh <- &client.Content{
								Err: probe.NewError(lerr),
							}
							continue
						}
						pathURL.Path = filepath.Join(pathURL.Path, lfi.Name())
						contentCh <- &client.Content{
							URL:  pathURL,
							Time: lfi.ModTime(),
							Size: lfi.Size(),
							Type: lfi.Mode(),
							Err:  probe.NewError(client.PathInsufficientPermission{Path: pathURL.Path}),
						}
						continue
					} else {
						contentCh <- &client.Content{
							Err: probe.NewError(client.PathInsufficientPermission{Path: pathURL.Path}),
						}
						continue
					}
				}
				if os.IsNotExist(err) {
					contentCh <- &client.Content{
						Err: probe.NewError(client.BrokenSymlink{Path: pathURL.Path}),
					}
					continue
				}
				if err != nil {
					contentCh <- &client.Content{
						Err: probe.NewError(err),
					}
					continue
				}
			}
			if strings.HasPrefix(file, prefix) {
				contentCh <- &client.Content{
					URL:  *client.NewURL(file),
					Time: st.ModTime(),
					Size: st.Size(),
					Type: st.Mode(),
					Err:  nil,
				}
				continue
			}
		}
		if strings.HasPrefix(file, prefix) {
			contentCh <- &client.Content{
				URL:  *client.NewURL(file),
				Time: fi.ModTime(),
				Size: fi.Size(),
				Type: fi.Mode(),
				Err:  nil,
			}
		}
	}
	return
}

func (f *fsClient) listInRoutine(contentCh chan<- *client.Content) {
	// close the channel when the function returns.
	defer close(contentCh)

	// save pathURL and file path for further usage.
	pathURL := *f.PathURL
	fpath := pathURL.Path

	fst, err := f.fsStat()
	if err != nil {
		if _, ok := err.ToGoError().(client.PathNotFound); ok {
			// If file does not exist treat it like a prefix and list all prefixes if any.
			prefix := fpath
			f.listPrefixes(prefix, contentCh)
			return
		}
		// For all other errors we return genuine error back to the caller.
		contentCh <- &client.Content{Err: err.Trace(fpath)}
		return
	}

	// Now if the file exists and doesn't end with a separator ('/') do not traverse it.
	// If the directory doesn't end with a separator, do not traverse it.
	if !strings.HasSuffix(fpath, string(pathURL.Separator)) && fst.Mode().IsDir() && fpath != "." {
		f.listPrefixes(fpath, contentCh)
		return
	}

	// If we really see the directory.
	switch fst.Mode().IsDir() {
	case true:
		files, err := ioutil.ReadDir(fpath)
		if err != nil {
			contentCh <- &client.Content{Err: probe.NewError(err)}
			return
		}
		for _, file := range files {
			fi := file
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				fi, err = os.Stat(filepath.Join(fpath, fi.Name()))
				if os.IsPermission(err) {
					// On windows there are folder symlinks
					// which are called junction files which
					// carry special meaning on windows
					// - which cannot be accessed with regular operations
					if runtime.GOOS == "windows" {
						newPath := filepath.Join(fpath, fi.Name())
						lfi, lerr := os.Lstat(newPath)
						if lerr != nil {
							if os.IsPermission(lerr) {
								contentCh <- &client.Content{
									Err: probe.NewError(client.PathInsufficientPermission{Path: newPath}),
								}
								continue
							}
							contentCh <- &client.Content{
								Err: probe.NewError(lerr),
							}
							continue
						}
						pathURL := *f.PathURL
						pathURL.Path = filepath.Join(pathURL.Path, lfi.Name())
						contentCh <- &client.Content{
							URL:  pathURL,
							Time: lfi.ModTime(),
							Size: lfi.Size(),
							Type: lfi.Mode(),
							Err:  probe.NewError(client.PathInsufficientPermission{Path: pathURL.Path}),
						}
						continue
					} else {
						contentCh <- &client.Content{
							Err: probe.NewError(client.PathInsufficientPermission{Path: pathURL.Path}),
						}
						continue
					}
				}
				if os.IsNotExist(err) {
					contentCh <- &client.Content{
						Err: probe.NewError(client.BrokenSymlink{Path: file.Name()}),
					}
					continue
				}
				if err != nil {
					contentCh <- &client.Content{
						Err: probe.NewError(err),
					}
					continue
				}
			}
			if fi.Mode().IsRegular() || fi.Mode().IsDir() {
				pathURL := *f.PathURL
				pathURL.Path = filepath.Join(pathURL.Path, fi.Name())
				contentCh <- &client.Content{
					URL:  pathURL,
					Time: fi.ModTime(),
					Size: fi.Size(),
					Type: fi.Mode(),
					Err:  nil,
				}
			}
		}
	default:
		contentCh <- &client.Content{
			URL:  pathURL,
			Time: fst.ModTime(),
			Size: fst.Size(),
			Type: fst.Mode(),
			Err:  nil,
		}
	}
}

func (f *fsClient) listRecursiveInRoutine(contentCh chan *client.Content) {
	// close channels upon return.
	defer close(contentCh)
	var dirName string
	var filePrefix string
	pathURL := *f.PathURL
	visitFS := func(fp string, fi os.FileInfo, err error) error {
		// If file path ends with os.PathSeparator and equals to root path, skip it.
		if strings.HasSuffix(fp, string(pathURL.Separator)) {
			if fp == dirName {
				return nil
			}
		}
		// We would never need to print system root path '/'.
		if fp == "/" {
			return nil
		}

		/// In following situations we need to handle listing properly.
		// - When filepath is '/usr' and prefix is '/usr/bi'
		// - When filepath is '/usr/bin/subdir' and prefix is '/usr/bi'
		// - Do not check filePrefix if its '.'
		if filePrefix != "." {
			if !strings.HasPrefix(fp, filePrefix) &&
				!strings.HasPrefix(filePrefix, fp) {
				if err == nil {
					if fi.IsDir() {
						return ErrSkipDir
					}
					return nil
				}
			}
			// - Skip when fp is /usr and prefix is '/usr/bi'
			// - Do not check filePrefix if its '.'
			if filePrefix != "." {
				if !strings.HasPrefix(fp, filePrefix) {
					return nil
				}
			}
		}

		if err != nil {
			// If operation is not permitted, we throw quickly back.
			if strings.Contains(err.Error(), "operation not permitted") {
				contentCh <- &client.Content{
					Err: probe.NewError(err),
				}
				return nil
			}
			if os.IsPermission(err) {
				if runtime.GOOS == "windows" {
					// On windows there are folder symlinks which are called junction files.
					// These files carry special meaning on windows which cannot be
					// accessed with regular operations.
					lfi, lerr := os.Lstat(fp)
					if lerr != nil {
						contentCh <- &client.Content{
							Err: probe.NewError(lerr),
						}
						return nil
					}
					pathURL := *f.PathURL
					pathURL.Path = filepath.Join(pathURL.Path, dirName)
					contentCh <- &client.Content{
						URL:  pathURL,
						Time: lfi.ModTime(),
						Size: lfi.Size(),
						Type: lfi.Mode(),
						Err:  probe.NewError(err),
					}
				} else {
					contentCh <- &client.Content{
						Err: probe.NewError(client.PathInsufficientPermission{Path: fp}),
					}
				}
				return nil
			}
			return err
		}
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			fi, err = os.Stat(fp)
			if err != nil {
				if os.IsPermission(err) {
					if runtime.GOOS == "windows" {
						// On windows there are folder symlinks which are called junction files.
						// These files carry special meaning on windows which cannot be
						// accessed with regular operations.
						lfi, lerr := os.Lstat(fp)
						if lerr != nil {
							if os.IsPermission(lerr) {
								contentCh <- &client.Content{
									Err: probe.NewError(client.PathInsufficientPermission{Path: fp}),
								}
								return nil
							}
							contentCh <- &client.Content{
								Err: probe.NewError(lerr),
							}
							return nil
						}
						pathURL := *f.PathURL
						pathURL.Path = filepath.Join(pathURL.Path, dirName)
						contentCh <- &client.Content{
							URL:  pathURL,
							Time: lfi.ModTime(),
							Size: lfi.Size(),
							Type: lfi.Mode(),
							Err:  probe.NewError(err),
						}
						return nil
					}
					contentCh <- &client.Content{
						Err: probe.NewError(err),
					}
					return nil
				}
				// Ignore in-accessible broken symlinks.
				if os.IsNotExist(err) {
					contentCh <- &client.Content{
						Err: probe.NewError(client.BrokenSymlink{Path: fp}),
					}
					return nil
				}
				// Ignore symlink loops.
				if strings.Contains(err.Error(), "too many levels of symbolic links") {
					contentCh <- &client.Content{
						Err: probe.NewError(client.TooManyLevelsSymlink{Path: fp}),
					}
					return nil
				}
				return err
			}
		}
		if fi.Mode().IsRegular() || fi.Mode().IsDir() {
			contentCh <- &client.Content{
				URL:  *client.NewURL(fp),
				Time: fi.ModTime(),
				Size: fi.Size(),
				Type: fi.Mode(),
				Err:  nil,
			}
		}
		return nil
	}
	// No prefix to be filtered by default.
	filePrefix = ""
	// if f.Path ends with os.PathSeparator - assuming it to be a directory and moving on.
	if strings.HasSuffix(pathURL.Path, string(pathURL.Separator)) {
		dirName = pathURL.Path
	} else {
		// if not a directory, take base path to navigate through WalkFunc.
		dirName = filepath.Dir(pathURL.Path)
		if !strings.HasSuffix(dirName, string(pathURL.Separator)) {
			// basepath truncates the os.PathSeparator,
			// add it deligently useful for trimming file path inside WalkFunc
			dirName = dirName + string(pathURL.Separator)
		}
		// filePrefix is kept for filtering incoming contents through WalkFunc.
		filePrefix = pathURL.Path
	}
	// Walks invokes our custom function.
	err := Walk(dirName, visitFS)
	if err != nil {
		contentCh <- &client.Content{
			Err: probe.NewError(err),
		}
	}
}

// MakeBucket - create a new bucket.
func (f *fsClient) MakeBucket() *probe.Error {
	err := os.MkdirAll(f.PathURL.Path, 0775)
	if err != nil {
		return probe.NewError(err)
	}
	return nil
}

// GetBucketACL - get bucket access.
func (f *fsClient) GetBucketAccess() (acl string, error *probe.Error) {
	return "", probe.NewError(client.APINotImplemented{API: "GetBucketAccess", APIType: "filesystem"})
}

// SetBucketAccess - set bucket access.
func (f *fsClient) SetBucketAccess(acl string) *probe.Error {
	return probe.NewError(client.APINotImplemented{API: "SetBucketAccess", APIType: "filesystem"})
}

// getFSMetadata - get metadata for files and folders.
func (f *fsClient) getFSMetadata() (content *client.Content, err *probe.Error) {
	st, err := f.fsStat()
	if err != nil {
		return nil, err.Trace()
	}
	content = new(client.Content)
	content.URL = *f.PathURL
	content.Size = st.Size()
	content.Time = st.ModTime()
	content.Type = st.Mode()
	return content, nil
}

// Stat - get metadata from path.
func (f *fsClient) Stat() (content *client.Content, err *probe.Error) {
	return f.getFSMetadata()
}
