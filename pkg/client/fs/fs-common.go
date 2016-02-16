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
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/probe"
)

// toClientError error constructs a typed client error for known filesystem errors.
func (f *fsClient) toClientError(e error, fpath string) *probe.Error {
	if os.IsPermission(e) {
		return probe.NewError(client.PathInsufficientPermission{Path: fpath})
	}
	if os.IsNotExist(e) {
		return probe.NewError(client.PathNotFound{Path: fpath})
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
			return nil, probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
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
			return nil, probe.NewError(client.PathInsufficientPermission{Path: f.PathURL.Path})
		}
		if os.IsNotExist(e) {
			return nil, probe.NewError(client.PathNotFound{Path: f.PathURL.Path})
		}
		return nil, probe.NewError(e)
	}
	return st, nil
}
