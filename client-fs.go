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

package main

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"io/ioutil"

	"github.com/minio/mc/pkg/hookreader"
	"github.com/minio/mc/pkg/ioutils"
	"github.com/minio/minio/pkg/probe"
)

// filesystem client
type fsClient struct {
	PathURL *clientURL
}

const (
	partSuffix = ".part.minio"
)

var ( // GOOS specific ignore list.
	ignoreFiles = map[string][]string{
		"darwin": {".DS_Store"},
		// "default": []string{""},
	}
)

// fsNew - instantiate a new fs
func fsNew(path string) (Client, *probe.Error) {
	if strings.TrimSpace(path) == "" {
		return nil, probe.NewError(EmptyPath{})
	}
	return &fsClient{
		PathURL: newClientURL(normalizePath(path)),
	}, nil
}

// isIgnoredFile returns true if 'filename' is on the exclude list.
func isIgnoredFile(filename string) bool {
	matchFile := path.Base(filename)

	// OS specific ignore list.
	for _, ignoredFile := range ignoreFiles[runtime.GOOS] {
		if ignoredFile == matchFile {
			return true
		}
	}

	// Default ignore list for all OSes.
	for _, ignoredFile := range ignoreFiles["default"] {
		if ignoredFile == matchFile {
			return true
		}
	}

	return false
}

// URL get url.
func (f *fsClient) GetURL() clientURL {
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
			return 0, probe.NewError(PathIsNotRegular{
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

	// Write to a temporary file "object.part.mc" before commit.
	objectPartPath := objectPath + partSuffix
	if objectDir != "" {
		// Create any missing top level directories.
		if e = os.MkdirAll(objectDir, 0700); e != nil {
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
				return 0, probe.NewError(UnexpectedShortWrite{
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
		if e = closer.Close(); e != nil {
			return totalWritten, probe.NewError(e)
		}
	}

	// Close the file before rename.
	if e = partFile.Close(); e != nil {
		return totalWritten, probe.NewError(e)
	}

	// Following verification is needed only for input size greater than '0'.
	if size > 0 {
		// Unexpected EOF reached (less data was written than expected).
		if totalWritten < size {
			return totalWritten, probe.NewError(UnexpectedEOF{
				TotalSize:    size,
				TotalWritten: totalWritten,
			})
		}
		// Unexpected ExcessRead (more data was written than expected).
		if totalWritten > size {
			return totalWritten, probe.NewError(UnexpectedExcessRead{
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
	return "", probe.NewError(APINotImplemented{
		API:     "ShareDownload",
		APIType: "filesystem",
	})
}

// ShareUpload - share upload not implemented for filesystem.
func (f *fsClient) ShareUpload(startsWith bool, expires time.Duration, contentType string) (map[string]string, *probe.Error) {
	return nil, probe.NewError(APINotImplemented{
		API:     "ShareUpload",
		APIType: "filesystem",
	})
}

// readFile reads and returns the data inside the file located
// at the provided filepath.
func readFile(fpath string) (io.ReadCloser, error) {
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(fpath, "/") {
		fpath = fpath + "."
	}
	fpath, e := filepath.EvalSymlinks(fpath)
	if e != nil {
		return nil, e
	}
	fileData, e := os.Open(fpath)
	if e != nil {
		return nil, e
	}
	return fileData, nil
}

// createFile creates an empty file at the provided filepath
// if one does not exist already.
func createFile(fpath string) (io.WriteCloser, error) {
	st, e := os.Stat(fpath)
	// If destination exists but is not regular.
	if e == nil && !st.Mode().IsRegular() {
		return nil, PathIsNotRegular{Path: fpath}
	}
	// If file exists already.
	if e != nil && !os.IsNotExist(e) {
		return nil, e
	}
	if e = os.MkdirAll(filepath.Dir(fpath), 0775); e != nil {
		return nil, e
	}
	file, e := os.Create(fpath)
	if e != nil {
		return nil, e
	}
	return file, nil
}

// Copy - copy data from source to destination
func (f *fsClient) Copy(source string, size int64, progress io.Reader) *probe.Error {
	// Don't use f.Get() f.Put() directly. Instead use readFile and createFile
	destination := f.PathURL.Path
	if destination == source { // Cannot copy file into itself
		return errOverWriteNotAllowed(destination).Trace(destination)
	}
	rc, e := readFile(source)
	if e != nil {
		err := f.toClientError(e, destination)
		return err.Trace(destination)
	}
	defer rc.Close()
	wc, e := createFile(destination)
	if e != nil {
		err := f.toClientError(e, destination)
		return err.Trace(destination)
	}
	defer wc.Close()
	reader := hookreader.NewHook(rc, progress)
	// Perform copy
	n, e := io.CopyN(wc, reader, size) // e == nil only if n != size
	// Only check size related errors if size is positive
	if size > 0 {
		if n < size { // Unexpected early EOF
			return probe.NewError(UnexpectedEOF{
				TotalSize:    size,
				TotalWritten: n,
			})
		}
		if n > size { // Unexpected ExcessRead
			return probe.NewError(UnexpectedExcessRead{
				TotalSize:    size,
				TotalWritten: n,
			})
		}
	}
	return nil
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
func (f *fsClient) List(recursive, incomplete bool) <-chan *clientContent {
	contentCh := make(chan *clientContent)
	if recursive {
		go f.listRecursiveInRoutine(contentCh, incomplete)
	} else {
		go f.listInRoutine(contentCh, incomplete)
	}
	return contentCh
}

// byDirName implements sort.Interface.
type byDirName []os.FileInfo

func (f byDirName) Len() int { return len(f) }
func (f byDirName) Less(i, j int) bool {
	// For directory add an ending separator fortrue lexical
	// order.
	if f[i].Mode().IsDir() {
		return f[i].Name()+string(filepath.Separator) < f[j].Name()
	}
	// For directory add an ending separator for true lexical
	// order.
	if f[j].Mode().IsDir() {
		return f[i].Name() < f[j].Name()+string(filepath.Separator)
	}
	return f[i].Name() < f[j].Name()
}
func (f byDirName) Swap(i, j int) { f[i], f[j] = f[j], f[i] }

// readDir reads the directory named by dirname and returns
// a list of sorted directory entries.
func readDir(dirname string) ([]os.FileInfo, error) {
	f, e := os.Open(dirname)
	if e != nil {
		return nil, e
	}
	list, e := f.Readdir(-1)
	if e != nil {
		return nil, e
	}
	if e = f.Close(); e != nil {
		return nil, e
	}
	sort.Sort(byDirName(list))
	return list, nil
}

// listPrefixes - list all files for any given prefix.
func (f *fsClient) listPrefixes(prefix string, contentCh chan<- *clientContent, incomplete bool) {
	dirName := filepath.Dir(prefix)
	files, e := readDir(dirName)
	if e != nil {
		err := f.toClientError(e, dirName)
		contentCh <- &clientContent{
			Err: err.Trace(dirName),
		}
		return
	}
	pathURL := *f.PathURL
	for _, fi := range files {
		// Skip ignored files.
		if isIgnoredFile(fi.Name()) {
			continue
		}

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
							contentCh <- &clientContent{
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
						contentCh <- &clientContent{
							URL:  pathURL,
							Time: lfi.ModTime(),
							Size: lfi.Size(),
							Type: lfi.Mode(),
							Err: probe.NewError(PathInsufficientPermission{
								Path: pathURL.Path,
							}),
						}
						continue
					} else {
						contentCh <- &clientContent{
							Err: probe.NewError(PathInsufficientPermission{
								Path: pathURL.Path,
							}),
						}
						continue
					}
				}
				if os.IsNotExist(e) {
					contentCh <- &clientContent{
						Err: probe.NewError(BrokenSymlink{
							Path: pathURL.Path,
						}),
					}
					continue
				}
				if e != nil {
					contentCh <- &clientContent{
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
				contentCh <- &clientContent{
					URL:  *newClientURL(file),
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
			contentCh <- &clientContent{
				URL:  *newClientURL(file),
				Time: fi.ModTime(),
				Size: fi.Size(),
				Type: fi.Mode(),
				Err:  nil,
			}
		}
	}
	return
}

func (f *fsClient) listInRoutine(contentCh chan<- *clientContent, incomplete bool) {
	// close the channel when the function returns.
	defer close(contentCh)

	// save pathURL and file path for further usage.
	pathURL := *f.PathURL
	fpath := pathURL.Path

	fst, err := f.fsStat()
	if err != nil {
		if _, ok := err.ToGoError().(PathNotFound); ok {
			// If file does not exist treat it like a prefix and list all prefixes if any.
			prefix := fpath
			f.listPrefixes(prefix, contentCh, incomplete)
			return
		}
		// For all other errors we return genuine error back to the caller.
		contentCh <- &clientContent{Err: err.Trace(fpath)}
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
			contentCh <- &clientContent{Err: probe.NewError(e)}
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
							contentCh <- &clientContent{
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
						contentCh <- &clientContent{
							URL:  pathURL,
							Time: lfi.ModTime(),
							Size: lfi.Size(),
							Type: lfi.Mode(),
							Err:  probe.NewError(PathInsufficientPermission{Path: pathURL.Path}),
						}
						continue
					} else {
						contentCh <- &clientContent{
							Err: probe.NewError(PathInsufficientPermission{Path: pathURL.Path}),
						}
						continue
					}
				}
				if os.IsNotExist(e) {
					contentCh <- &clientContent{
						Err: probe.NewError(BrokenSymlink{Path: file.Name()}),
					}
					continue
				}
				if e != nil {
					contentCh <- &clientContent{
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

				// Skip ignored files.
				if isIgnoredFile(fi.Name()) {
					continue
				}

				contentCh <- &clientContent{
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
		contentCh <- &clientContent{
			URL:  pathURL,
			Time: fst.ModTime(),
			Size: fst.Size(),
			Type: fst.Mode(),
			Err:  nil,
		}
	}
}

func (f *fsClient) listRecursiveInRoutine(contentCh chan *clientContent, incomplete bool) {
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

		// Ignore files from ignore list.
		if isIgnoredFile(fi.Name()) {
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
						return ioutils.ErrSkipDir
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
				contentCh <- &clientContent{
					Err: probe.NewError(e),
				}
				return nil
			}
			if os.IsPermission(e) {
				if runtime.GOOS == "windows" {
					lfi, le := f.handleWindowsSymlinks(fp)
					if le != nil {
						contentCh <- &clientContent{
							Err: le.Trace(fp),
						}
						return nil
					}
					pathURL = *f.PathURL
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
					contentCh <- &clientContent{
						URL:  pathURL,
						Time: lfi.ModTime(),
						Size: lfi.Size(),
						Type: lfi.Mode(),
						Err:  probe.NewError(e),
					}
				} else {
					contentCh <- &clientContent{
						Err: probe.NewError(PathInsufficientPermission{Path: fp}),
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
							contentCh <- &clientContent{
								Err: le.Trace(fp),
							}
							return nil
						}
						pathURL = *f.PathURL
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
						contentCh <- &clientContent{
							URL:  pathURL,
							Time: lfi.ModTime(),
							Size: lfi.Size(),
							Type: lfi.Mode(),
							Err:  probe.NewError(e),
						}
						return nil
					}
					contentCh <- &clientContent{
						Err: probe.NewError(e),
					}
					return nil
				}
				// Ignore in-accessible broken symlinks.
				if os.IsNotExist(e) {
					contentCh <- &clientContent{
						Err: probe.NewError(BrokenSymlink{Path: fp}),
					}
					return nil
				}
				// Ignore symlink loops.
				if strings.Contains(e.Error(), "too many levels of symbolic links") {
					contentCh <- &clientContent{
						Err: probe.NewError(TooManyLevelsSymlink{Path: fp}),
					}
					return nil
				}
				return e
			}
		}
		if fi.Mode().IsRegular() {
			if incomplete {
				if !strings.HasSuffix(fi.Name(), partSuffix) {
					return nil
				}
			} else {
				if strings.HasSuffix(fi.Name(), partSuffix) {
					return nil
				}
			}
			contentCh <- &clientContent{
				URL:  *newClientURL(fp),
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
	e := ioutils.FTW(dirName, visitFS)
	if e != nil {
		contentCh <- &clientContent{
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

// GetAccess - get access policy permissions.
func (f *fsClient) GetAccess() (access string, err *probe.Error) {
	// For windows this feature is not implemented.
	if runtime.GOOS == "windows" {
		return "", probe.NewError(APINotImplemented{API: "GetAccess", APIType: "filesystem"})
	}
	st, err := f.fsStat()
	if err != nil {
		return "", err.Trace(f.PathURL.String())
	}
	if !st.Mode().IsDir() {
		return "", probe.NewError(APINotImplemented{API: "GetAccess", APIType: "filesystem"})
	}
	switch {
	case st.Mode() == os.FileMode(0777):
		return "readwrite", nil
	case st.Mode() == os.FileMode(0555):
		return "readonly", nil
	case st.Mode() == os.FileMode(0333):
		return "writeonly", nil
	}
	return "none", nil
}

// SetAccess - set access policy permissions.
func (f *fsClient) SetAccess(access string) *probe.Error {
	// For windows this feature is not implemented.
	if runtime.GOOS == "windows" {
		return probe.NewError(APINotImplemented{API: "SetAccess", APIType: "filesystem"})
	}
	st, err := f.fsStat()
	if err != nil {
		return err.Trace(f.PathURL.String())
	}
	if !st.Mode().IsDir() {
		return probe.NewError(APINotImplemented{API: "SetAccess", APIType: "filesystem"})
	}
	var mode os.FileMode
	switch access {
	case "readonly":
		mode = os.FileMode(0555)
	case "writeonly":
		mode = os.FileMode(0333)
	case "readwrite":
		mode = os.FileMode(0777)
	case "none":
		mode = os.FileMode(0755)
	}
	e := os.Chmod(f.PathURL.Path, mode)
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// Stat - get metadata from path.
func (f *fsClient) Stat() (content *clientContent, err *probe.Error) {
	st, err := f.fsStat()
	if err != nil {
		return nil, err.Trace(f.PathURL.String())
	}
	content = &clientContent{}
	content.URL = *f.PathURL
	content.Size = st.Size()
	content.Time = st.ModTime()
	content.Type = st.Mode()
	return content, nil
}

// toClientError error constructs a typed client error for known filesystem errors.
func (f *fsClient) toClientError(e error, fpath string) *probe.Error {
	if os.IsPermission(e) {
		return probe.NewError(PathInsufficientPermission{Path: fpath})
	}
	if os.IsNotExist(e) {
		return probe.NewError(PathNotFound{Path: fpath})
	}
	return probe.NewError(e)
}

// handle windows symlinks - eg: junction files.
func (f *fsClient) handleWindowsSymlinks(fpath string) (os.FileInfo, *probe.Error) {
	// On windows there are directory symlinks which are called junction files.
	// These files carry special meaning on windows they cannot be,
	// accessed with regular operations.
	file, e := os.Lstat(fpath)
	if e != nil {
		err := f.toClientError(e, fpath)
		return nil, err.Trace(fpath)
	}
	return file, nil
}

// fsStat - wrapper function to get file stat.
func (f *fsClient) fsStat() (os.FileInfo, *probe.Error) {
	fpath := f.PathURL.Path
	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(fpath, string(f.PathURL.Separator)) {
		fpath = fpath + "."
	}
	fpath, e := filepath.EvalSymlinks(fpath)
	if e != nil {
		if os.IsPermission(e) {
			if runtime.GOOS == "windows" {
				return f.handleWindowsSymlinks(f.PathURL.Path)
			}
			return nil, probe.NewError(PathInsufficientPermission{Path: f.PathURL.Path})
		}
		err := f.toClientError(e, f.PathURL.Path)
		return nil, err.Trace(fpath)
	}
	st, e := os.Stat(fpath)
	if e != nil {
		if os.IsPermission(e) {
			if runtime.GOOS == "windows" {
				return f.handleWindowsSymlinks(fpath)
			}
			return nil, probe.NewError(PathInsufficientPermission{Path: f.PathURL.Path})
		}
		if os.IsNotExist(e) {
			return nil, probe.NewError(PathNotFound{Path: f.PathURL.Path})
		}
		return nil, probe.NewError(e)
	}
	return st, nil
}
