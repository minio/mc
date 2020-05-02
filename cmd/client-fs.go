/*
 * MinIO Client (C) 2015-2020 MinIO, Inc.
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

package cmd

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/xattr"
	"github.com/rjeczalik/notify"

	"github.com/minio/mc/pkg/disk"
	"github.com/minio/mc/pkg/hookreader"
	"github.com/minio/mc/pkg/ioutils"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/encrypt"
	"github.com/minio/minio/pkg/bucket/object/tagging"
)

// filesystem client
type fsClient struct {
	PathURL *ClientURL
}

const (
	partSuffix     = ".part.minio"
	slashSeperator = "/"
)

var ( // GOOS specific ignore list.
	ignoreFiles = map[string][]string{
		"darwin":  {"*.DS_Store"},
		"default": {""},
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

func isNotSupported(e error) bool {
	if e == nil {
		return false
	}
	errno := e.(*xattr.Error)
	if errno == nil {
		return false
	}

	// check if filesystem supports extended attributes
	return errno.Err == syscall.Errno(syscall.ENOTSUP) || errno.Err == syscall.Errno(syscall.EOPNOTSUPP)
}

// isIgnoredFile returns true if 'filename' is on the exclude list.
func isIgnoredFile(filename string) bool {
	matchFile := filepath.Base(filename)

	// OS specific ignore list.
	for _, ignoredFile := range ignoreFiles[runtime.GOOS] {
		matched, e := filepath.Match(ignoredFile, matchFile)
		if e != nil {
			panic(e)
		}
		if matched {
			return true
		}
	}

	// Default ignore list for all OSes.
	for _, ignoredFile := range ignoreFiles["default"] {
		matched, e := filepath.Match(ignoredFile, matchFile)
		if e != nil {
			panic(e)
		}
		if matched {
			return true
		}
	}

	return false
}

// URL get url.
func (f *fsClient) GetURL() ClientURL {
	return *f.PathURL
}

// Select replies a stream of query results.
func (f *fsClient) Select(expression string, sse encrypt.ServerSide, opts SelectObjectOpts) (io.ReadCloser, *probe.Error) {
	return nil, probe.NewError(APINotImplemented{
		API:     "Select",
		APIType: "filesystem",
	})
}

// Watches for all fs events on an input path.
func (f *fsClient) Watch(params watchParams) (*WatchObject, *probe.Error) {
	eventChan := make(chan []EventInfo)
	errorChan := make(chan *probe.Error)
	doneChan := make(chan struct{})
	// Make the channel buffered to ensure no event is dropped. Notify will drop
	// an event if the receiver is not able to keep up the sending pace.
	in, out := PipeChan(1000)

	var fsEvents []notify.Event
	for _, event := range params.events {
		switch event {
		case "put":
			fsEvents = append(fsEvents, EventTypePut...)
		case "delete":
			fsEvents = append(fsEvents, EventTypeDelete...)
		case "get":
			fsEvents = append(fsEvents, EventTypeGet...)
		default:
			return nil, errInvalidArgument().Trace(event)
		}
	}

	// Set up a watchpoint listening for events within a directory tree rooted
	// at current working directory. Dispatch remove events to c.
	recursivePath := f.PathURL.Path
	if params.recursive {
		recursivePath = f.PathURL.Path + "..."
	}
	if e := notify.Watch(recursivePath, in, fsEvents...); e != nil {
		return nil, probe.NewError(e)
	}

	// wait for doneChan to close the watcher, eventChan and errorChan
	go func() {
		<-doneChan

		close(eventChan)
		close(errorChan)
		notify.Stop(in)
	}()

	timeFormatFS := "2006-01-02T15:04:05.000Z"

	// Get fsnotify notifications for events and errors, and sent them
	// using eventChan and errorChan
	go func() {
		for event := range out {
			if isIgnoredFile(event.Path()) {
				continue
			}
			var i os.FileInfo
			if IsPutEvent(event.Event()) {
				// Look for any writes, send a response to indicate a full copy.
				var e error
				i, e = os.Stat(event.Path())
				if e != nil {
					if os.IsNotExist(e) {
						continue
					}
					errorChan <- probe.NewError(e)
					continue
				}
				if i.IsDir() {
					// we want files
					continue
				}
				eventChan <- []EventInfo{{
					Time: UTCNow().Format(timeFormatFS),
					Size: i.Size(),
					Path: event.Path(),
					Type: EventCreate,
				}}
			} else if IsDeleteEvent(event.Event()) {
				eventChan <- []EventInfo{{
					Time: UTCNow().Format(timeFormatFS),
					Path: event.Path(),
					Type: EventRemove,
				}}
			} else if IsGetEvent(event.Event()) {
				eventChan <- []EventInfo{{
					Time: UTCNow().Format(timeFormatFS),
					Path: event.Path(),
					Type: EventAccessed,
				}}
			}
		}
	}()

	return &WatchObject{
		eventInfoChan: eventChan,
		errorChan:     errorChan,
		doneChan:      doneChan,
	}, nil
}

func isStreamFile(objectPath string) bool {
	switch objectPath {
	case os.DevNull:
		fallthrough
	case os.Stdin.Name():
		fallthrough
	case os.Stdout.Name():
		fallthrough
	case os.Stderr.Name():
		return true
	}
	return false
}

func preserveAttributes(fd *os.File, attr map[string]string) *probe.Error {
	mode, e := strconv.ParseUint(attr["mode"], 10, 32)
	if e != nil {
		return probe.NewError(e)
	}

	// Attempt to change the file mode.
	if e := fd.Chmod(os.FileMode(mode)); e != nil {
		return probe.NewError(e)
	}

	uid, e := strconv.Atoi(attr["uid"])
	if e != nil {
		return probe.NewError(e)
	}

	gid, e := strconv.Atoi(attr["gid"])
	if e != nil {
		return probe.NewError(e)
	}

	// Attempt to change the owner.
	if e := fd.Chown(uid, gid); e != nil {
		return probe.NewError(e)
	}

	return nil
}

/// Object operations.

func (f *fsClient) put(reader io.Reader, size int64, metadata map[string][]string, progress io.Reader) (int64, *probe.Error) {
	// ContentType is not handled on purpose.
	// For filesystem this is a redundant information.

	// Extract dir name.
	objectDir, objectName := filepath.Split(f.PathURL.Path)

	if objectDir != "" {
		// Create any missing top level directories.
		if e := os.MkdirAll(objectDir, 0777); e != nil {
			err := f.toClientError(e, f.PathURL.Path)
			return 0, err.Trace(f.PathURL.Path)
		}

		// Check if object name is empty, it must be an empty directory
		if objectName == "" {
			return 0, nil
		}
	}

	objectPath := f.PathURL.Path
	avoidResumeUpload := isStreamFile(objectPath)
	// Write to a temporary file "object.part.minio" before commit.
	objectPartPath := objectPath + partSuffix
	if avoidResumeUpload {
		objectPartPath = objectPath
	}

	// If exists, open in append mode. If not create it the part file.
	partFile, e := os.OpenFile(objectPartPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if e != nil {
		err := f.toClientError(e, f.PathURL.Path)
		return 0, err.Trace(f.PathURL.Path)
	}

	attr := make(map[string]string)
	if len(metadata["mc-attrs"]) != 0 {
		attr, e = parseAttribute(metadata["mc-attrs"][0])
		if e != nil {
			return 0, probe.NewError(e)
		}
		if err := preserveAttributes(partFile, attr); err != nil {
			return 0, err.Trace(objectPath)
		}
	}

	// Get stat to get the current size.
	partSt, e := os.Stat(objectPartPath)
	if e != nil {
		err := f.toClientError(e, objectPartPath)
		return 0, err.Trace(objectPartPath)
	}

	var totalWritten int64
	// Current file offset.
	var currentOffset = partSt.Size()

	if !isStdIO(reader) && size > 0 {
		reader = hookreader.NewHook(reader, progress)
		if seeker, ok := reader.(io.Seeker); ok {
			if _, e = seeker.Seek(currentOffset, 0); e != nil {
				return 0, probe.NewError(e)
			}
			// Discard bytes until currentOffset.
			if _, e = io.CopyN(ioutil.Discard, progress, currentOffset); e != nil {
				return 0, probe.NewError(e)
			}
		}
	} else {
		reader = hookreader.NewHook(reader, progress)
		// Discard bytes until currentOffset.
		if _, e = io.CopyN(ioutil.Discard, reader, currentOffset); e != nil {
			return 0, probe.NewError(e)
		}
	}

	n, e := io.Copy(partFile, reader)
	if e != nil {
		return 0, probe.NewError(e)
	}

	// Save currently copied total into totalWritten.
	totalWritten = n + currentOffset

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
	if !avoidResumeUpload {
		// Safely completed put. Now commit by renaming to actual filename.
		if e = os.Rename(objectPartPath, objectPath); e != nil {
			err := f.toClientError(e, objectPath)
			return totalWritten, err.Trace(objectPartPath, objectPath)
		}

		if len(attr) != 0 {
			atime, e := strconv.ParseInt(attr["atime"], 10, 64)
			if e != nil {
				return totalWritten, probe.NewError(e)
			}

			mtime, e := strconv.ParseInt(attr["mtime"], 10, 64)
			if e != nil {
				return totalWritten, probe.NewError(e)
			}

			// Attempt to change the access and modify time
			if e := os.Chtimes(objectPath, time.Unix(atime, 0), time.Unix(mtime, 0)); e != nil {
				return totalWritten, probe.NewError(e)
			}
		}
	}
	return totalWritten, nil
}

// Put - create a new file with metadata.
func (f *fsClient) Put(ctx context.Context, reader io.Reader, size int64, metadata map[string]string, progress io.Reader, sse encrypt.ServerSide, md5, disableMultipart bool) (int64, *probe.Error) {
	if metadata["mc-attrs"] != "" {
		meta := make(map[string][]string)
		meta["mc-attrs"] = append(meta["mc-attrs"], metadata["mc-attrs"])
		return f.put(reader, size, meta, progress)
	}
	return f.put(reader, size, nil, progress)
}

// ShareDownload - share download not implemented for filesystem.
func (f *fsClient) ShareDownload(expires time.Duration) (string, *probe.Error) {
	return "", probe.NewError(APINotImplemented{
		API:     "ShareDownload",
		APIType: "filesystem",
	})
}

// ShareUpload - share upload not implemented for filesystem.
func (f *fsClient) ShareUpload(startsWith bool, expires time.Duration, contentType string) (string, map[string]string, *probe.Error) {
	return "", nil, probe.NewError(APINotImplemented{
		API:     "ShareUpload",
		APIType: "filesystem",
	})
}

// Copy - copy data from source to destination
func (f *fsClient) Copy(source string, size int64, progress io.Reader, srcSSE, tgtSSE encrypt.ServerSide, metadata map[string]string, disableMultipart bool) *probe.Error {
	rc, e := os.Open(source)
	if e != nil {
		err := f.toClientError(e, source)
		return err.Trace(source)
	}
	defer rc.Close()

	destination := f.PathURL.Path
	if _, err := f.put(rc, size, map[string][]string{}, progress); err != nil {
		return err.Trace(destination, source)
	}
	return nil
}

// Get returns reader and any additional metadata.
func (f *fsClient) Get(sse encrypt.ServerSide) (io.ReadCloser, *probe.Error) {
	fileData, e := os.Open(f.PathURL.Path)
	if e != nil {
		err := f.toClientError(e, f.PathURL.Path)
		return nil, err.Trace(f.PathURL.Path)
	}
	return fileData, nil
}

// Check if the given error corresponds to ENOTEMPTY for unix
// and ERROR_DIR_NOT_EMPTY for windows (directory not empty).
func isSysErrNotEmpty(err error) bool {
	if err == syscall.ENOTEMPTY {
		return true
	}
	if pathErr, ok := err.(*os.PathError); ok {
		if runtime.GOOS == "windows" {
			if errno, _ok := pathErr.Err.(syscall.Errno); _ok && errno == 0x91 {
				// ERROR_DIR_NOT_EMPTY
				return true
			}
		}
		if pathErr.Err == syscall.ENOTEMPTY {
			return true
		}
	}
	return false
}

// deleteFile deletes a file path if its empty. If it's successfully deleted,
// it will recursively delete empty parent directories
// until it finds one with files in it. Returns nil for a non-empty directory.
func deleteFile(deletePath string) error {
	// Attempt to remove path.
	if e := os.Remove(deletePath); e != nil {
		if isSysErrNotEmpty(e) {
			return nil
		}
		return e
	}

	// Trailing slash is removed when found to ensure
	// slashpath.Dir() to work as intended.
	parentPath := strings.TrimSuffix(deletePath, slashSeperator)
	parentPath = path.Dir(parentPath)

	if parentPath != "." {
		return deleteFile(parentPath)
	}

	return nil
}

// Remove - remove entry read from clientContent channel.
func (f *fsClient) Remove(isIncomplete, isRemoveBucket, isBypass bool, contentCh <-chan *ClientContent) <-chan *probe.Error {
	errorCh := make(chan *probe.Error)

	// Goroutine reads from contentCh and removes the entry in content.
	go func() {
		defer close(errorCh)

		for content := range contentCh {
			if content.Err != nil {
				errorCh <- content.Err
				continue
			}
			name := content.URL.Path
			// Add partSuffix for incomplete uploads.
			if isIncomplete {
				name += partSuffix
			}
			e := deleteFile(name)
			if e == nil {
				continue
			}
			if os.IsNotExist(e) && isRemoveBucket {
				// ignore PathNotFound for dir removal.
				return
			}
			if os.IsPermission(e) {
				// Ignore permission error.
				errorCh <- probe.NewError(PathInsufficientPermission{Path: content.URL.Path})
			} else {
				errorCh <- probe.NewError(e)
				return
			}
		}
	}()

	return errorCh
}

// List - list files and folders.
func (f *fsClient) List(isRecursive, isIncomplete, isMetadata bool, showDir DirOpt) <-chan *ClientContent {
	contentCh := make(chan *ClientContent)
	filteredCh := make(chan *ClientContent)

	if isRecursive {
		if showDir == DirNone {
			go f.listRecursiveInRoutine(contentCh, isMetadata)
		} else {
			go f.listDirOpt(contentCh, isIncomplete, isMetadata, showDir)
		}
	} else {
		go f.listInRoutine(contentCh, isMetadata)
	}

	// This function filters entries from any  listing go routine
	// created previously. If isIncomplete is activated, we will
	// only show partly uploaded files,
	go func() {
		for c := range contentCh {
			if isIncomplete {
				if !strings.HasSuffix(c.URL.Path, partSuffix) {
					continue
				}
				// Strip part suffix
				c.URL.Path = strings.Split(c.URL.Path, partSuffix)[0]
			} else {
				if strings.HasSuffix(c.URL.Path, partSuffix) {
					continue
				}
			}
			// Send to filtered channel
			filteredCh <- c
		}
		defer close(filteredCh)
	}()

	return filteredCh
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
func (f *fsClient) listPrefixes(prefix string, contentCh chan<- *ClientContent) {
	dirName := filepath.Dir(prefix)
	files, e := readDir(dirName)
	if e != nil {
		err := f.toClientError(e, dirName)
		contentCh <- &ClientContent{
			Err: err.Trace(dirName),
		}
		return
	}
	for _, fi := range files {
		// Skip ignored files.
		if isIgnoredFile(fi.Name()) {
			continue
		}

		file := filepath.Join(dirName, fi.Name())
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			st, e := os.Stat(file)
			if e != nil {
				// Ignore any errors on symlink
				continue
			}
			if strings.HasPrefix(file, prefix) {
				contentCh <- &ClientContent{
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
			contentCh <- &ClientContent{
				URL:  *newClientURL(file),
				Time: fi.ModTime(),
				Size: fi.Size(),
				Type: fi.Mode(),
				Err:  nil,
			}
		}
	}
}

func (f *fsClient) listInRoutine(contentCh chan<- *ClientContent, isMetadata bool) {
	// close the channel when the function returns.
	defer close(contentCh)

	// save pathURL and file path for further usage.
	pathURL := *f.PathURL
	fpath := pathURL.Path

	fst, err := f.fsStat(false)
	if err != nil {
		if _, ok := err.ToGoError().(PathNotFound); ok {
			// If file does not exist treat it like a prefix and list all prefixes if any.
			prefix := fpath
			f.listPrefixes(prefix, contentCh)
			return
		}
		// For all other errors we return genuine error back to the caller.
		contentCh <- &ClientContent{Err: err.Trace(fpath)}
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
		files, e := readDir(fpath)
		if err != nil {
			contentCh <- &ClientContent{Err: probe.NewError(e)}
			return
		}
		for _, file := range files {
			fi := file
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				fp := filepath.Join(fpath, fi.Name())
				fi, e = os.Stat(fp)
				if e != nil {
					// Ignore all errors on symlinks
					continue
				}
			}
			if fi.Mode().IsRegular() || fi.Mode().IsDir() {
				pathURL = *f.PathURL
				pathURL.Path = filepath.Join(pathURL.Path, fi.Name())

				// Skip ignored files.
				if isIgnoredFile(fi.Name()) {
					continue
				}

				contentCh <- &ClientContent{
					URL:  pathURL,
					Time: fi.ModTime(),
					Size: fi.Size(),
					Type: fi.Mode(),
					Err:  nil,
				}
			}
		}
	default:
		contentCh <- &ClientContent{
			URL:  pathURL,
			Time: fst.ModTime(),
			Size: fst.Size(),
			Type: fst.Mode(),
			Err:  nil,
		}
	}
}

// List files recursively using non-recursive mode.
func (f *fsClient) listDirOpt(contentCh chan *ClientContent, isIncomplete bool, isMetadata bool, dirOpt DirOpt) {
	defer close(contentCh)

	// Trim trailing / or \.
	currentPath := f.PathURL.Path
	currentPath = strings.TrimSuffix(currentPath, "/")
	if runtime.GOOS == "windows" {
		currentPath = strings.TrimSuffix(currentPath, `\`)
	}

	// Closure function reads currentPath and sends to contentCh. If a directory is found, it lists the directory content recursively.
	var listDir func(currentPath string) bool
	listDir = func(currentPath string) (isStop bool) {
		files, e := readDir(currentPath)
		if e != nil {
			if os.IsPermission(e) {
				contentCh <- &ClientContent{
					Err: probe.NewError(PathInsufficientPermission{
						Path: currentPath,
					}),
				}
				return false
			}

			contentCh <- &ClientContent{Err: probe.NewError(e)}
			return true
		}

		for _, file := range files {
			name := filepath.Join(currentPath, file.Name())
			content := ClientContent{
				URL:  *newClientURL(name),
				Time: file.ModTime(),
				Size: file.Size(),
				Type: file.Mode(),
				Err:  nil,
			}
			if file.Mode().IsDir() {
				if dirOpt == DirFirst && !isIncomplete {
					contentCh <- &content
				}
				if listDir(filepath.Join(name)) {
					return true
				}
				if dirOpt == DirLast && !isIncomplete {
					contentCh <- &content
				}

				continue
			}

			contentCh <- &content
		}

		return false
	}

	// listDir() does not send currentPath to contentCh.  We send it here depending on dirOpt.

	if dirOpt == DirFirst && !isIncomplete {
		contentCh <- &ClientContent{URL: *newClientURL(currentPath), Type: os.ModeDir}
	}

	listDir(currentPath)

	if dirOpt == DirLast && !isIncomplete {
		contentCh <- &ClientContent{URL: *newClientURL(currentPath), Type: os.ModeDir}
	}
}

func (f *fsClient) listRecursiveInRoutine(contentCh chan *ClientContent, isMetadata bool) {
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
				contentCh <- &ClientContent{
					Err: probe.NewError(e),
				}
				return nil
			}
			if os.IsPermission(e) {
				contentCh <- &ClientContent{
					Err: probe.NewError(PathInsufficientPermission{Path: fp}),
				}
				return nil
			}
			return e
		}
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			fi, e = os.Stat(fp)
			if e != nil {
				// Ignore any errors for symlink
				return nil
			}
		}
		if fi.Mode().IsRegular() {
			contentCh <- &ClientContent{
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
		contentCh <- &ClientContent{
			Err: probe.NewError(e),
		}
	}
}

// MakeBucket - create a new bucket.
func (f *fsClient) MakeBucket(region string, ignoreExisting, withLock bool) *probe.Error {
	// TODO: ignoreExisting has no effect currently. In the future, we want
	// to call os.Mkdir() when ignoredExisting is disabled and os.MkdirAll()
	// otherwise.
	// NOTE: withLock=true has no meaning here.
	e := os.MkdirAll(f.PathURL.Path, 0777)
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// Set object lock configuration of bucket.
func (f *fsClient) SetObjectLockConfig(mode *minio.RetentionMode, validity *uint, unit *minio.ValidityUnit) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "SetObjectLockConfig",
		APIType: "filesystem",
	})
}

// Get object lock configuration of bucket.
func (f *fsClient) GetObjectLockConfig() (mode *minio.RetentionMode, validity *uint, unit *minio.ValidityUnit, perr *probe.Error) {
	return nil, nil, nil, probe.NewError(APINotImplemented{
		API:     "GetObjectLockConfig",
		APIType: "filesystem",
	})
}

// GetAccessRules - unsupported API
func (f *fsClient) GetAccessRules() (map[string]string, *probe.Error) {
	return map[string]string{}, probe.NewError(APINotImplemented{
		API:     "GetBucketPolicy",
		APIType: "filesystem",
	})
}

// Set object retention for a given object.
func (f *fsClient) PutObjectRetention(mode *minio.RetentionMode, retainUntilDate *time.Time, bypassGovernance bool) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "PutObjectRetention",
		APIType: "filesystem",
	})
}

// Set object legal hold for a given object.
func (f *fsClient) PutObjectLegalHold(lhold *minio.LegalHoldStatus) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "PutObjectLegalHold",
		APIType: "filesystem",
	})
}

// GetAccess - get access policy permissions.
func (f *fsClient) GetAccess() (access string, policyJSON string, err *probe.Error) {
	// For windows this feature is not implemented.
	if runtime.GOOS == "windows" {
		return "", "", probe.NewError(APINotImplemented{API: "GetAccess", APIType: "filesystem"})
	}
	st, err := f.fsStat(false)
	if err != nil {
		return "", "", err.Trace(f.PathURL.String())
	}
	if !st.Mode().IsDir() {
		return "", "", probe.NewError(APINotImplemented{API: "GetAccess", APIType: "filesystem"})
	}
	// Mask with os.ModePerm to get only inode permissions
	switch st.Mode() & os.ModePerm {
	case os.FileMode(0777):
		return "readwrite", "", nil
	case os.FileMode(0555):
		return "readonly", "", nil
	case os.FileMode(0333):
		return "writeonly", "", nil
	}
	return "none", "", nil
}

// SetAccess - set access policy permissions.
func (f *fsClient) SetAccess(access string, isJSON bool) *probe.Error {
	// For windows this feature is not implemented.
	// JSON policy for fs is not yet implemented.
	if runtime.GOOS == "windows" || isJSON {
		return probe.NewError(APINotImplemented{API: "SetAccess", APIType: "filesystem"})
	}
	st, err := f.fsStat(false)
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
func (f *fsClient) Stat(isIncomplete, isPreserve bool, sse encrypt.ServerSide) (content *ClientContent, err *probe.Error) {
	st, err := f.fsStat(isIncomplete)
	if err != nil {
		return nil, err.Trace(f.PathURL.String())
	}

	content = &ClientContent{}
	content.URL = *f.PathURL
	content.Size = st.Size()
	content.Time = st.ModTime()
	content.Type = st.Mode()
	content.Metadata = map[string]string{
		"Content-Type": guessURLContentType(f.PathURL.Path),
	}

	path := f.PathURL.String()
	metaData, pErr := getAllXattrs(path)
	if pErr != nil {
		return content, nil
	}
	for k, v := range metaData {
		content.Metadata[k] = v
	}
	// Populates meta data with file system attribute only in case of
	// when preserve flag is passed.
	if isPreserve {
		fileAttr, err := disk.GetFileSystemAttrs(path)
		if err != nil {
			return content, nil
		}
		content.Metadata["mc-attrs"] = fileAttr
	}

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
	if errors.Is(e, syscall.ELOOP) {
		return probe.NewError(TooManyLevelsSymlink{Path: fpath})
	}
	return probe.NewError(e)
}

// fsStat - wrapper function to get file stat.
func (f *fsClient) fsStat(isIncomplete bool) (os.FileInfo, *probe.Error) {
	fpath := f.PathURL.Path

	// Check if the path corresponds to a directory and returns
	// the successful result whether isIncomplete is specified or not.
	st, e := os.Stat(fpath)
	if e == nil && st.IsDir() {
		return st, nil
	}

	if isIncomplete {
		fpath += partSuffix
	}

	st, e = os.Stat(fpath)
	if e != nil {
		return nil, f.toClientError(e, fpath)
	}
	return st, nil
}

func (f *fsClient) AddUserAgent(_, _ string) {
}

// Get Object Tags
func (f *fsClient) GetObjectTagging() (tagging.Tagging, *probe.Error) {
	return tagging.Tagging{}, probe.NewError(APINotImplemented{
		API:     "GetObjectTagging",
		APIType: "filesystem",
	})
}

// Set Object tags
func (f *fsClient) SetObjectTagging(tagMap map[string]string) *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "SetObjectTagging",
		APIType: "filesystem",
	})
}

// Delete object tags
func (f *fsClient) DeleteObjectTagging() *probe.Error {
	return probe.NewError(APINotImplemented{
		API:     "DeleteObjectTagging",
		APIType: "filesystem",
	})
}
