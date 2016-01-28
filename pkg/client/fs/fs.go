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
	"github.com/minio/mc/pkg/hookreader"
	"github.com/minio/minio-xl/pkg/probe"
)

// filesystem client
type fsClient struct {
	PathURL *client.URL
}

const (
	partSuffix = ".part.mc"
)

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

// Put - create a new file.
func (f *fsClient) Put(reader io.Reader, size int64, contentType string, progress io.Reader) (int64, *probe.Error) {
	// ContentType is not handled on purpose.
	// For filesystem this is a redundant information.

	// Extract dir name.
	objectDir, _ := filepath.Split(f.PathURL.Path)
	objectPath := f.PathURL.Path

	// Verify if destination already exists.
	st, e := os.Stat(objectPath)
	if e == nil {
		// If the destination exists and is not a regular file.
		if !st.Mode().IsRegular() {
			return 0, probe.NewError(client.PathIsNotRegular{
				Path: objectPath,
			})
		}
	}

	// Proceed if file does not exist. return for all other errors.
	if e != nil {
		if !os.IsNotExist(e) {
			return 0, probe.NewError(e)
		}
	}

	// Write to a temporary file "object.part.mc" before committ.
	objectPartPath := objectPath + partSuffix
	if objectDir != "" {
		// Create any missing top level directories.
		if e := os.MkdirAll(objectDir, 0700); e != nil {
			err := f.toClientError(e, f.PathURL.Path)
			return 0, err.Trace(f.PathURL.Path)
		}
	}

	// If exists, open in append mode. If not create it the part file.
	partFile, e := os.OpenFile(objectPartPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		err := f.toClientError(e, f.PathURL.Path)
		return 0, err.Trace(f.PathURL.Path)
	}

	// Get stat to get the current size.
	partSt, e := partFile.Stat()
	if e != nil {
		err := f.toClientError(e, objectPartPath)
		return 0, err.Trace(objectPartPath)
	}

	var totalWritten int64
	// Current file offset.
	var currentOffset = partSt.Size()

	// Reflect and verify if incoming reader implements ReaderAt.
	readerAt, ok := reader.(io.ReaderAt)
	if ok {
		// Notify the progress bar if any till current size.
		if progress != nil {
			if _, e = io.CopyN(ioutil.Discard, progress, currentOffset); e != nil {
				return 0, probe.NewError(e)
			}
		}
		// Allocate buffer of 10MiB once.
		readAtBuffer := make([]byte, 10*1024*1024)

		// Loop through all offsets on incoming io.ReaderAt and write
		// to the destination.
		for {
			readAtSize, re := readerAt.ReadAt(readAtBuffer, currentOffset)
			if re != nil && re != io.EOF {
				// For any errors other than io.EOF, we return error
				// and breakout.
				err := f.toClientError(re, objectPartPath)
				return 0, err.Trace(objectPartPath)
			}
			writtenSize, we := partFile.Write(readAtBuffer[:readAtSize])
			if we != nil {
				err := f.toClientError(we, objectPartPath)
				return 0, err.Trace(objectPartPath)
			}
			// read size and subsequent write differ, a possible
			// corruption return here.
			if readAtSize != writtenSize {
				// Unexpected write (less data was written than expected).
				return 0, probe.NewError(client.UnexpectedShortWrite{
					InputSize: readAtSize,
					WriteSize: writtenSize,
				})
			}
			// Notify the progress bar if any for written size.
			if progress != nil {
				if _, e = io.CopyN(ioutil.Discard, progress, int64(writtenSize)); e != nil {
					return totalWritten, probe.NewError(e)
				}
			}
			currentOffset += int64(writtenSize)
			// Once we see io.EOF we break out of the loop.
			if re == io.EOF {
				break
			}
		}
		// Save currently copied total into totalWritten.
		totalWritten = currentOffset
	} else {
		reader = hookreader.NewHook(reader, progress)
		// Discard bytes until currentOffset.
		if _, e = io.CopyN(ioutil.Discard, reader, currentOffset); e != nil {
			return 0, probe.NewError(e)
		}
		var n int64
		n, e = io.Copy(partFile, reader)
		if e != nil {
			return 0, probe.NewError(e)
		}
		// Save currently copied total into totalWritten.
		totalWritten = n + currentOffset
	}

	// Close the input reader as well, if possible.
	closer, ok := reader.(io.Closer)
	if ok {
		if e := closer.Close(); e != nil {
			return totalWritten, probe.NewError(e)
		}
	}

	// Close the file before rename.
	if e := partFile.Close(); e != nil {
		return totalWritten, probe.NewError(e)
	}

	// Following verification is needed only for input size greater than '0'.
	if size > 0 {
		// Unexpected EOF reached (less data was written than expected).
		if totalWritten < size {
			return totalWritten, probe.NewError(client.UnexpectedEOF{
				TotalSize:    size,
				TotalWritten: totalWritten,
			})
		}
		// Unexpected ExcessRead (more data was written than expected).
		if totalWritten > size {
			return totalWritten, probe.NewError(client.UnexpectedExcessRead{
				TotalSize:    size,
				TotalWritten: totalWritten,
			})
		}
	}
	// Safely completed put. Now commit by renaming to actual filename.
	if e = os.Rename(objectPartPath, objectPath); e != nil {
		err := f.toClientError(e, objectPath)
		return totalWritten, err.Trace(objectPartPath, objectPath)
	}
	return totalWritten, nil
}

// ShareDownload - share download not implemented for filesystem.
func (f *fsClient) ShareDownload(expires time.Duration) (string, *probe.Error) {
	return "", probe.NewError(client.APINotImplemented{
		API:     "ShareDownload",
		APIType: "filesystem",
	})
}

// ShareUpload - share upload not implemented for filesystem.
func (f *fsClient) ShareUpload(startsWith bool, expires time.Duration, contentType string) (map[string]string, *probe.Error) {
	return nil, probe.NewError(client.APINotImplemented{
		API:     "ShareUpload",
		APIType: "filesystem",
	})
}

// GetPartial download a part object from bucket.
// sets err for any errors, reader is nil for errors.
func (f *fsClient) Get() (io.Reader, *probe.Error) {
	tmppath := f.PathURL.Path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(tmppath, string(f.PathURL.Separator)) {
		tmppath = tmppath + "."
	}

	// Resolve symlinks.
	_, e := filepath.EvalSymlinks(tmppath)
	if e != nil {
		err := f.toClientError(e, f.PathURL.Path)
		return nil, err.Trace(f.PathURL.Path)
	}
	fileData, e := os.Open(f.PathURL.Path)
	if e != nil {
		err := f.toClientError(e, f.PathURL.Path)
		return nil, err.Trace(f.PathURL.Path)
	}
	return fileData, nil
}

// Remove - remove the path.
func (f *fsClient) Remove(incomplete bool) *probe.Error {
	if incomplete {
		return nil
	}
	e := os.Remove(f.PathURL.Path)
	err := f.toClientError(e, f.PathURL.Path)
	return err.Trace(f.PathURL.Path)
}

// List - list files and folders.
func (f *fsClient) List(recursive, incomplete bool) <-chan *client.Content {
	contentCh := make(chan *client.Content)
	switch recursive {
	case true:
		go f.listRecursiveInRoutine(contentCh, incomplete)
	default:
		go f.listInRoutine(contentCh, incomplete)
	}
	return contentCh
}

// listPrefixes - list all files for any given prefix.
func (f *fsClient) listPrefixes(prefix string, contentCh chan<- *client.Content, incomplete bool) {
	dirName := filepath.Dir(prefix)
	files, e := readDir(dirName)
	if e != nil {
		err := f.toClientError(e, dirName)
		contentCh <- &client.Content{
			Err: err.Trace(dirName),
		}
		return
	}
	pathURL := *f.PathURL
	for _, fi := range files {
		file := filepath.Join(dirName, fi.Name())
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			st, e := os.Stat(file)
			if e != nil {
				if os.IsPermission(e) {
					// On windows there are folder symlinks
					// which are called junction files which
					// carry special meaning on windows
					// - which cannot be accessed with regular operations
					if runtime.GOOS == "windows" {
						newPath := filepath.Join(prefix, fi.Name())
						lfi, le := f.handleWindowsSymlinks(newPath)
						if le != nil {
							contentCh <- &client.Content{
								Err: le.Trace(newPath),
							}
							continue
						}
						if incomplete {
							if !strings.HasSuffix(lfi.Name(), partSuffix) {
								continue
							}
						} else {
							if strings.HasSuffix(lfi.Name(), partSuffix) {
								continue
							}
						}
						pathURL.Path = filepath.Join(pathURL.Path, lfi.Name())
						contentCh <- &client.Content{
							URL:  pathURL,
							Time: lfi.ModTime(),
							Size: lfi.Size(),
							Type: lfi.Mode(),
							Err: probe.NewError(client.PathInsufficientPermission{
								Path: pathURL.Path,
							}),
						}
						continue
					} else {
						contentCh <- &client.Content{
							Err: probe.NewError(client.PathInsufficientPermission{
								Path: pathURL.Path,
							}),
						}
						continue
					}
				}
				if os.IsNotExist(e) {
					contentCh <- &client.Content{
						Err: probe.NewError(client.BrokenSymlink{
							Path: pathURL.Path,
						}),
					}
					continue
				}
				if e != nil {
					contentCh <- &client.Content{
						Err: probe.NewError(e),
					}
					continue
				}
			}
			if strings.HasPrefix(file, prefix) {
				if incomplete {
					if !strings.HasSuffix(st.Name(), partSuffix) {
						continue
					}
				} else {
					if strings.HasSuffix(st.Name(), partSuffix) {
						continue
					}
				}
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
			if incomplete {
				if !strings.HasSuffix(fi.Name(), partSuffix) {
					continue
				}
			} else {
				if strings.HasSuffix(fi.Name(), partSuffix) {
					continue
				}
			}
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

func (f *fsClient) listInRoutine(contentCh chan<- *client.Content, incomplete bool) {
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
			f.listPrefixes(prefix, contentCh, incomplete)
			return
		}
		// For all other errors we return genuine error back to the caller.
		contentCh <- &client.Content{Err: err.Trace(fpath)}
		return
	}

	// Now if the file exists and doesn't end with a separator ('/') do not traverse it.
	// If the directory doesn't end with a separator, do not traverse it.
	if !strings.HasSuffix(fpath, string(pathURL.Separator)) && fst.Mode().IsDir() && fpath != "." {
		f.listPrefixes(fpath, contentCh, incomplete)
		return
	}

	// If we really see the directory.
	switch fst.Mode().IsDir() {
	case true:
		files, e := readDir(fpath)
		if err != nil {
			contentCh <- &client.Content{Err: probe.NewError(e)}
			return
		}
		for _, file := range files {
			fi := file
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				fi, e = os.Stat(filepath.Join(fpath, fi.Name()))
				if os.IsPermission(e) {
					// On windows there are folder symlinks
					// which are called junction files which
					// carry special meaning on windows
					// - which cannot be accessed with regular operations
					if runtime.GOOS == "windows" {
						newPath := filepath.Join(fpath, fi.Name())
						lfi, le := f.handleWindowsSymlinks(newPath)
						if le != nil {
							contentCh <- &client.Content{
								Err: le.Trace(newPath),
							}
							continue
						}
						if incomplete {
							if !strings.HasSuffix(lfi.Name(), partSuffix) {
								continue
							}
						} else {
							if strings.HasSuffix(lfi.Name(), partSuffix) {
								continue
							}
						}
						pathURL = *f.PathURL
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
				if os.IsNotExist(e) {
					contentCh <- &client.Content{
						Err: probe.NewError(client.BrokenSymlink{Path: file.Name()}),
					}
					continue
				}
				if e != nil {
					contentCh <- &client.Content{
						Err: probe.NewError(e),
					}
					continue
				}
			}
			if fi.Mode().IsRegular() || fi.Mode().IsDir() {
				if incomplete {
					if !strings.HasSuffix(fi.Name(), partSuffix) {
						continue
					}
				} else {
					if strings.HasSuffix(fi.Name(), partSuffix) {
						continue
					}
				}
				pathURL = *f.PathURL
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
		if incomplete {
			if !strings.HasSuffix(fst.Name(), partSuffix) {
				return
			}
		} else {
			if strings.HasSuffix(fst.Name(), partSuffix) {
				return
			}
		}
		contentCh <- &client.Content{
			URL:  pathURL,
			Time: fst.ModTime(),
			Size: fst.Size(),
			Type: fst.Mode(),
			Err:  nil,
		}
	}
}

func (f *fsClient) listRecursiveInRoutine(contentCh chan *client.Content, incomplete bool) {
	// close channels upon return.
	defer close(contentCh)
	var dirName string
	var filePrefix string
	pathURL := *f.PathURL
	visitFS := func(fp string, fi os.FileInfo, e error) error {
		// If file path ends with filepath.Separator and equals to root path, skip it.
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
				if e == nil {
					if fi.IsDir() {
						return errSkipDir
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
		if e != nil {
			// If operation is not permitted, we throw quickly back.
			if strings.Contains(e.Error(), "operation not permitted") {
				contentCh <- &client.Content{
					Err: probe.NewError(e),
				}
				return nil
			}
			if os.IsPermission(e) {
				if runtime.GOOS == "windows" {
					lfi, le := f.handleWindowsSymlinks(fp)
					if le != nil {
						contentCh <- &client.Content{
							Err: le.Trace(fp),
						}
						return nil
					}
					pathURL := *f.PathURL
					pathURL.Path = filepath.Join(pathURL.Path, dirName)
					if incomplete {
						if !strings.HasSuffix(lfi.Name(), partSuffix) {
							return nil
						}
					} else {
						if strings.HasSuffix(lfi.Name(), partSuffix) {
							return nil
						}
					}
					contentCh <- &client.Content{
						URL:  pathURL,
						Time: lfi.ModTime(),
						Size: lfi.Size(),
						Type: lfi.Mode(),
						Err:  probe.NewError(e),
					}
				} else {
					contentCh <- &client.Content{
						Err: probe.NewError(client.PathInsufficientPermission{Path: fp}),
					}
				}
				return nil
			}
			return e
		}
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			fi, e = os.Stat(fp)
			if e != nil {
				if os.IsPermission(e) {
					if runtime.GOOS == "windows" {
						lfi, le := f.handleWindowsSymlinks(fp)
						if le != nil {
							contentCh <- &client.Content{
								Err: le.Trace(fp),
							}
							return nil
						}
						pathURL := *f.PathURL
						pathURL.Path = filepath.Join(pathURL.Path, dirName)
						if incomplete {
							if !strings.HasSuffix(lfi.Name(), partSuffix) {
								return nil
							}
						} else {
							if !strings.HasSuffix(lfi.Name(), partSuffix) {
								return nil
							}
						}
						contentCh <- &client.Content{
							URL:  pathURL,
							Time: lfi.ModTime(),
							Size: lfi.Size(),
							Type: lfi.Mode(),
							Err:  probe.NewError(e),
						}
						return nil
					}
					contentCh <- &client.Content{
						Err: probe.NewError(e),
					}
					return nil
				}
				// Ignore in-accessible broken symlinks.
				if os.IsNotExist(e) {
					contentCh <- &client.Content{
						Err: probe.NewError(client.BrokenSymlink{Path: fp}),
					}
					return nil
				}
				// Ignore symlink loops.
				if strings.Contains(e.Error(), "too many levels of symbolic links") {
					contentCh <- &client.Content{
						Err: probe.NewError(client.TooManyLevelsSymlink{Path: fp}),
					}
					return nil
				}
				return e
			}
		}
		if fi.Mode().IsRegular() || fi.Mode().IsDir() {
			if incomplete {
				if !strings.HasSuffix(fi.Name(), partSuffix) {
					return nil
				}
			} else {
				if strings.HasSuffix(fi.Name(), partSuffix) {
					return nil
				}
			}
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
	// if f.Path ends with filepath.Separator - assuming it to be a directory and moving on.
	if strings.HasSuffix(pathURL.Path, string(pathURL.Separator)) {
		dirName = pathURL.Path
	} else {
		// if not a directory, take base path to navigate through WalkFunc.
		dirName = filepath.Dir(pathURL.Path)
		if !strings.HasSuffix(dirName, string(pathURL.Separator)) {
			// basepath truncates the filepath.Separator,
			// add it deligently useful for trimming file path inside WalkFunc
			dirName = dirName + string(pathURL.Separator)
		}
		// filePrefix is kept for filtering incoming contents through WalkFunc.
		filePrefix = pathURL.Path
	}
	// walks invokes our custom function.
	e := walk(dirName, visitFS)
	if e != nil {
		contentCh <- &client.Content{
			Err: probe.NewError(e),
		}
	}
}

// MakeBucket - create a new bucket.
func (f *fsClient) MakeBucket(region string) *probe.Error {
	e := os.MkdirAll(f.PathURL.Path, 0775)
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// GetBucketACL - get bucket access.
func (f *fsClient) GetBucketAccess() (acl string, err *probe.Error) {
	return "", probe.NewError(client.APINotImplemented{API: "GetBucketAccess", APIType: "filesystem"})
}

// SetBucketAccess - set bucket access.
func (f *fsClient) SetBucketAccess(acl string) *probe.Error {
	return probe.NewError(client.APINotImplemented{API: "SetBucketAccess", APIType: "filesystem"})
}

// Stat - get metadata from path.
func (f *fsClient) Stat() (content *client.Content, err *probe.Error) {
	st, err := f.fsStat()
	if err != nil {
		return nil, err.Trace(f.PathURL.String())
	}
	content = new(client.Content)
	content.URL = *f.PathURL
	content.Size = st.Size()
	content.Time = st.ModTime()
	content.Type = st.Mode()
	return content, nil
}
