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
	"os"
	"path/filepath"
	"sort"
)

// byDirName implements sort.Interface.
type byDirName []os.FileInfo

func (f byDirName) Len() int { return len(f) }
func (f byDirName) Less(i, j int) bool {
	// For directory add an ending separator fortrue lexical order.
	if f[i].Mode().IsDir() {
		return f[i].Name()+string(os.PathSeparator) < f[j].Name()
	}
	// For directory add an ending separator for true lexical order.
	if f[j].Mode().IsDir() {
		return f[i].Name() < f[j].Name()+string(os.PathSeparator)
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

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(dirname string) ([]string, error) {
	names, e := readDirUnsortedNames(dirname)
	if e != nil {
		return nil, e
	}
	sort.Strings(names)
	return names, nil
}

// getRealName - gets the proper filename for sorting purposes
// Readdir() filters out directory names without separators, add
// them back for proper sorting results.
func getRealName(info os.FileInfo) string {
	if info.IsDir() {
		// Make sure directory has its end separator.
		return info.Name() + string(os.PathSeparator)
	}
	return info.Name()
}

// readDirUnsortedNames reads the directory named by dirname and
// return a unsorted list of directory entries.
func readDirUnsortedNames(dirname string) ([]string, error) {
	f, e := os.Open(dirname)
	if e != nil {
		return nil, e
	}
	nameInfos, e := f.Readdir(-1)
	if e != nil {
		return nil, e
	}
	if e = f.Close(); e != nil {
		return nil, e
	}
	var names []string
	for _, nameInfo := range nameInfos {
		names = append(names, getRealName(nameInfo))
	}
	return names, nil
}

// walk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root.
func walk(root string, walkFn walkFunc) error {
	info, e := os.Lstat(root)
	if e != nil {
		return walkFn(root, nil, e)
	}
	return walkInternal(root, info, walkFn)
}

// WalkFunc is the type of the function called for each file or directory
// visited by Walk. The path argument contains the argument to Walk as a
// prefix; that is, if Walk is called with "dir", which is a directory
// containing the file "a", the walk function will be called with argument
// "dir/a". The info argument is the os.FileInfo for the named path.
type walkFunc func(path string, info os.FileInfo, e error) error

// ErrSkipDir is used as a return value from WalkFuncs to indicate that
// the directory named in the call is to be skipped. It is not returned
// as an error by any function.
var errSkipDir = errors.New("skip this directory")

// ErrSkipFile is used as a return value from WalkFuncs to indicate that
// the file named in the call is to be skipped. It is not returned
// as an error by any function.
var errSkipFile = errors.New("skip this file")

// walkInternal recursively descends path, calling w.
func walkInternal(path string, info os.FileInfo, walkFn walkFunc) error {
	e := walkFn(path, info, nil)
	if e != nil {
		if info.Mode().IsDir() && e == errSkipDir {
			return nil
		}
		if info.Mode().IsRegular() && e == errSkipFile {
			return nil
		}
		return e
	}
	if !info.IsDir() {
		return nil
	}
	names, e := readDirNames(path)
	if e != nil {
		return walkFn(path, info, e)
	}
	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, e := os.Lstat(filename)
		if e != nil {
			if e = walkFn(filename, fileInfo, e); e != nil && e != errSkipDir && e != errSkipFile {
				return e
			}
		} else {
			e = walkInternal(filename, fileInfo, walkFn)
			if e != nil {
				if e == errSkipDir || e == errSkipFile {
					return nil
				}
				return e
			}
		}
	}
	return nil
}
