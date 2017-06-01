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

package cmd

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
	"github.com/rjeczalik/notify"
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

// isIgnoredFile returns true if 'filename' is on the exclude list.
func isIgnoredFile(filename string) bool {
	matchFile := path.Base(filename)

	// OS specific ignore list.
	for _, ignoredFile := range ignoreFiles[runtime.GOOS] {
		matched, err := filepath.Match(ignoredFile, matchFile)
		if err != nil {
			panic(err)
		}
		if matched {
			return true
		}
	}

	// Default ignore list for all OSes.
	for _, ignoredFile := range ignoreFiles["default"] {
		matched, err := filepath.Match(ignoredFile, matchFile)
		if err != nil {
			panic(err)
		}
		if matched {
			return true
		}
	}

	return false
}

// URL get url.
func (f *fsClient) GetURL() clientURL {
	return *f.PathURL
}

// Watches for all fs events on an input path.
func (f *fsClient) Watch(params watchParams) (*watchObject, *probe.Error) {
	eventChan := make(chan EventInfo)
	errorChan := make(chan *probe.Error)
	doneChan := make(chan bool)
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
				eventChan <- EventInfo{
					Time: UTCNow().Format(timeFormatFS),
					Size: i.Size(),
					Path: event.Path(),
					Type: EventCreate,
				}
			} else if IsDeleteEvent(event.Event()) {
				eventChan <- EventInfo{
					Time: UTCNow().Format(timeFormatFS),
					Path: event.Path(),
					Type: EventRemove,
				}
			} else if IsGetEvent(event.Event()) {
				eventChan <- EventInfo{
					Time: UTCNow().Format(timeFormatFS),
					Path: event.Path(),
					Type: EventAccessed,
				}
			}
		}
	}()

	return &watchObject{
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

/// Object operations.

func (f *fsClient) put(reader io.Reader, size int64, metadata map[string][]string, progress io.Reader) (int64, *probe.Error) {
	// ContentType is not handled on purpose.
	// For filesystem this is a redundant information.

	// Extract dir name.
	objectDir, _ := filepath.Split(f.PathURL.Path)
	objectPath := f.PathURL.Path

	avoidResumeUpload := isStreamFile(objectPath)
	// Write to a temporary file "object.part.minio" before commit.
	objectPartPath := objectPath + partSuffix
	if avoidResumeUpload {
		objectPartPath = objectPath
	}
	if objectDir != "" {
		// Create any missing top level directories.
		if e := os.MkdirAll(objectDir, 0777); e != nil {
			err := f.toClientError(e, f.PathURL.Path)
			return 0, err.Trace(f.PathURL.Path)
		}
	}

	// If exists, open in append mode. If not create it the part file.
	partFile, e := os.OpenFile(objectPartPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
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

	// Use ReadAt() capability when reader implements it, but also avoid it in two cases:
	// *) reader represents a standard input/output stream since they return illegal seek error when ReadAt() is invoked
	// *) we know in advance that reader will provide zero length data
	if readerAt, ok := reader.(io.ReaderAt); ok && !isStdIO(reader) && size > 0 {
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
		for currentOffset < size {
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
	if !avoidResumeUpload {
		// Safely completed put. Now commit by renaming to actual filename.
		if e = os.Rename(objectPartPath, objectPath); e != nil {
			err := f.toClientError(e, objectPath)
			return totalWritten, err.Trace(objectPartPath, objectPath)
		}
	}
	return totalWritten, nil
}

// Put - create a new file with metadata.
func (f *fsClient) Put(reader io.Reader, size int64, metadata map[string][]string, progress io.Reader) (int64, *probe.Error) {
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
	if e := os.MkdirAll(filepath.Dir(fpath), 0777); e != nil {
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
	n, _ := io.CopyN(wc, reader, size) // e == nil only if n != size
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

// get - get wrapper returning reader and additional metadata if any.
// currently only returns metadata.
func (f *fsClient) get() (io.Reader, map[string][]string, *probe.Error) {
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
		return nil, nil, err.Trace(f.PathURL.Path)
	}
	fileData, e := os.Open(f.PathURL.Path)
	if e != nil {
		err := f.toClientError(e, f.PathURL.Path)
		return nil, nil, err.Trace(f.PathURL.Path)
	}
	metadata := map[string][]string{
		"Content-Type": {guessURLContentType(f.PathURL.Path)},
	}
	return fileData, metadata, nil
}

// Get returns reader and any additional metadata.
func (f *fsClient) Get() (io.Reader, map[string][]string, *probe.Error) {
	return f.get()
}

// Remove - remove entry read from clientContent channel.
func (f *fsClient) Remove(isIncomplete bool, contentCh <-chan *clientContent) <-chan *probe.Error {
	errorCh := make(chan *probe.Error)

	// Goroutine reads from contentCh and removes the entry in content.
	go func() {
		defer close(errorCh)

		for content := range contentCh {
			name := content.URL.Path
			// Add partSuffix for incomplete uploads.
			if isIncomplete {
				name += partSuffix
			}
			if err := os.Remove(name); err != nil {
				if os.IsPermission(err) {
					// Ignore permission error.
					errorCh <- probe.NewError(PathInsufficientPermission{Path: content.URL.Path})
				} else {
					errorCh <- probe.NewError(err)
					return
				}
			}
		}
	}()

	return errorCh
}

// List - list files and folders.
func (f *fsClient) List(isRecursive, isIncomplete bool, showDir DirOpt) <-chan *clientContent {
	contentCh := make(chan *clientContent)
	filteredCh := make(chan *clientContent)

	if isRecursive {
		if showDir == DirNone {
			go f.listRecursiveInRoutine(contentCh)
		} else {
			go f.listDirOpt(contentCh, isIncomplete, showDir)
		}
	} else {
		go f.listInRoutine(contentCh)
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
func (f *fsClient) listPrefixes(prefix string, contentCh chan<- *clientContent) {
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
					contentCh <- &clientContent{
						Err: probe.NewError(PathInsufficientPermission{
							Path: pathURL.Path,
						}),
					}
					continue
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

func (f *fsClient) listInRoutine(contentCh chan<- *clientContent) {
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
		contentCh <- &clientContent{Err: err.Trace(fpath)}
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
			contentCh <- &clientContent{Err: probe.NewError(e)}
			return
		}
		for _, file := range files {
			fi := file
			if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
				fi, e = os.Stat(filepath.Join(fpath, fi.Name()))
				if os.IsPermission(e) {
					contentCh <- &clientContent{
						Err: probe.NewError(PathInsufficientPermission{Path: pathURL.Path}),
					}
					continue
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
		contentCh <- &clientContent{
			URL:  pathURL,
			Time: fst.ModTime(),
			Size: fst.Size(),
			Type: fst.Mode(),
			Err:  nil,
		}
	}
}

// List files recursively using non-recursive mode.
func (f *fsClient) listDirOpt(contentCh chan *clientContent, isIncomplete bool, dirOpt DirOpt) {
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
		files, err := readDir(currentPath)
		if err != nil {
			if os.IsPermission(err) {
				contentCh <- &clientContent{Err: probe.NewError(PathInsufficientPermission{Path: currentPath})}
				return false
			}

			contentCh <- &clientContent{Err: probe.NewError(err)}
			return true
		}

		for _, file := range files {
			name := filepath.Join(currentPath, file.Name())
			content := clientContent{
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
		contentCh <- &clientContent{URL: *newClientURL(currentPath), Type: os.ModeDir}
	}

	listDir(currentPath)

	if dirOpt == DirLast && !isIncomplete {
		contentCh <- &clientContent{URL: *newClientURL(currentPath), Type: os.ModeDir}
	}
}

func (f *fsClient) listRecursiveInRoutine(contentCh chan *clientContent) {
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
				contentCh <- &clientContent{
					Err: probe.NewError(PathInsufficientPermission{Path: fp}),
				}
				return nil
			}
			return e
		}
		if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
			fi, e = os.Stat(fp)
			if e != nil {
				if os.IsPermission(e) {
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
func (f *fsClient) MakeBucket(region string, ignoreExisting bool) *probe.Error {
	// TODO: ignoreExisting has no effect currently. In the future, we want
	// to call os.Mkdir() when ignoredExisting is disabled and os.MkdirAll()
	// otherwise.
	e := os.MkdirAll(f.PathURL.Path, 0777)
	if e != nil {
		return probe.NewError(e)
	}
	return nil
}

// GetAccessRules - unsupported API
func (f *fsClient) GetAccessRules() (map[string]string, *probe.Error) {
	return map[string]string{}, probe.NewError(APINotImplemented{
		API:     "ListBucketPolicies",
		APIType: "filesystem",
	})
}

// GetAccess - get access policy permissions.
func (f *fsClient) GetAccess() (access string, err *probe.Error) {
	// For windows this feature is not implemented.
	if runtime.GOOS == "windows" {
		return "", probe.NewError(APINotImplemented{API: "GetAccess", APIType: "filesystem"})
	}
	st, err := f.fsStat(false)
	if err != nil {
		return "", err.Trace(f.PathURL.String())
	}
	if !st.Mode().IsDir() {
		return "", probe.NewError(APINotImplemented{API: "GetAccess", APIType: "filesystem"})
	}
	// Mask with os.ModePerm to get only inode permissions
	switch st.Mode() & os.ModePerm {
	case os.FileMode(0777):
		return "readwrite", nil
	case os.FileMode(0555):
		return "readonly", nil
	case os.FileMode(0333):
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
func (f *fsClient) Stat(isIncomplete bool) (content *clientContent, err *probe.Error) {
	st, err := f.fsStat(isIncomplete)
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

	// Golang strips trailing / if you clean(..) or
	// EvalSymlinks(..). Adding '.' prevents it from doing so.
	if strings.HasSuffix(fpath, string(f.PathURL.Separator)) {
		fpath = fpath + "."
	}
	fpath, e = filepath.EvalSymlinks(fpath)
	if e != nil {
		if os.IsPermission(e) {
			return nil, probe.NewError(PathInsufficientPermission{Path: f.PathURL.Path})
		}
		err := f.toClientError(e, f.PathURL.Path)
		return nil, err.Trace(fpath)
	}

	st, e = os.Stat(fpath)
	if e != nil {
		if os.IsPermission(e) {
			return nil, probe.NewError(PathInsufficientPermission{Path: f.PathURL.Path})
		}
		if os.IsNotExist(e) {
			return nil, probe.NewError(PathNotFound{Path: f.PathURL.Path})
		}
		return nil, probe.NewError(e)
	}
	return st, nil
}
