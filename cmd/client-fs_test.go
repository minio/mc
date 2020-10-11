/*
 * MinIO Client (C) 2015 MinIO, Inc.
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
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	. "gopkg.in/check.v1"
)

// Test list files in a folder.
func (s *TestSuite) TestList(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	// Create multiple files.
	objectPath := filepath.Join(root, "object1")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, IsNil)

	data := "hello"

	reader := bytes.NewReader([]byte(data))
	var n int64
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	objectPath = filepath.Join(root, "object2")
	fsClient, err = fsNew(objectPath)
	c.Assert(err, IsNil)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	fsClient, err = fsNew(root)
	c.Assert(err, IsNil)

	// Verify previously create files and list them.
	var contents []*ClientContent
	for content := range fsClient.List(globalContext, ListOptions{ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 1)
	c.Assert(contents[0].Type.IsDir(), Equals, true)

	// Create another file.
	objectPath = filepath.Join(root, "test1/newObject1")
	fsClient, err = fsNew(objectPath)
	c.Assert(err, IsNil)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	fsClient, err = fsNew(root)
	c.Assert(err, IsNil)

	contents = nil
	// List non recursive to list only top level files.
	for content := range fsClient.List(globalContext, ListOptions{ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 1)
	c.Assert(contents[0].Type.IsDir(), Equals, true)

	fsClient, err = fsNew(root)
	c.Assert(err, IsNil)

	contents = nil
	// List recursively all files and verify.
	for content := range fsClient.List(globalContext, ListOptions{IsRecursive: true, ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}

	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 3)

	var regularFiles int
	var regularDirs int
	// Test number of expected files and directories.
	for _, content := range contents {
		if content.Type.IsRegular() {
			regularFiles++
			continue
		}
		if content.Type.IsDir() {
			regularDirs++
			continue
		}
	}
	c.Assert(regularDirs, Equals, 0)
	c.Assert(regularFiles, Equals, 3)

	// Create an ignored file and list to verify if its ignored.
	objectPath = filepath.Join(root, "test1/.DS_Store")
	fsClient, err = fsNew(objectPath)
	c.Assert(err, IsNil)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	fsClient, err = fsNew(root)
	c.Assert(err, IsNil)

	contents = nil
	// List recursively all files and verify.
	for content := range fsClient.List(globalContext, ListOptions{IsRecursive: true, ShowDir: DirNone}) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}

	c.Assert(err, IsNil)
	switch runtime.GOOS {
	case "darwin":
		c.Assert(len(contents), Equals, 3)
	default:
		c.Assert(len(contents), Equals, 4)
	}

	regularFiles = 0
	// Test number of expected files.
	for _, content := range contents {
		if content.Type.IsRegular() {
			regularFiles++
			continue
		}
	}
	switch runtime.GOOS {
	case "darwin":
		c.Assert(regularFiles, Equals, 3)
	default:
		c.Assert(regularFiles, Equals, 4)
	}
}

// Test put bucket aka 'mkdir()' operation.
func (s *TestSuite) TestPutBucket(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsClient, err := fsNew(bucketPath)
	c.Assert(err, IsNil)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	c.Assert(err, IsNil)
}

// Test stat bucket aka 'stat()' operation.
func (s *TestSuite) TestStatBucket(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")

	fsClient, err := fsNew(bucketPath)
	c.Assert(err, IsNil)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	c.Assert(err, IsNil)
	_, err = fsClient.Stat(context.Background(), StatOptions{})
	c.Assert(err, IsNil)
}

// Test bucket acl fails for directories.
func (s *TestSuite) TestBucketACLFails(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsClient, err := fsNew(bucketPath)
	c.Assert(err, IsNil)
	err = fsClient.MakeBucket(context.Background(), "us-east-1", true, false)
	c.Assert(err, IsNil)

	// On windows setting permissions is not supported.
	if runtime.GOOS != "windows" {
		err = fsClient.SetAccess(context.Background(), "readonly", false)
		c.Assert(err, IsNil)

		_, _, err = fsClient.GetAccess(context.Background())
		c.Assert(err, IsNil)
	}
}

// Test creating a file.
func (s *TestSuite) TestPut(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	reader := bytes.NewReader([]byte(data))
	var n int64
	n, err = fsClient.Put(context.Background(), reader, int64(len(data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))
}

// Test read a file.
func (s *TestSuite) TestGet(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	var reader io.Reader
	reader = bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(len(data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	reader, err = fsClient.Get(context.Background(), GetOptions{})
	c.Assert(err, IsNil)
	var results bytes.Buffer
	_, e = io.Copy(&results, reader)
	c.Assert(e, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

}

// Test get range in a file.
func (s *TestSuite) TestGetRange(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, IsNil)

	data := "hello world"
	var reader io.Reader
	reader = bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(len(data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	reader, err = fsClient.Get(context.Background(), GetOptions{})
	c.Assert(err, IsNil)
	var results bytes.Buffer
	buf := make([]byte, 5)
	m, e := reader.(io.ReaderAt).ReadAt(buf, 0)
	c.Assert(e, IsNil)
	c.Assert(m, Equals, 5)
	_, e = results.Write(buf)
	c.Assert(e, IsNil)
	c.Assert([]byte("hello"), DeepEquals, results.Bytes())
}

// Test stat file.
func (s *TestSuite) TestStatObject(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fsNew(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	dataLen := len(data)
	reader := bytes.NewReader([]byte(data))
	n, err := fsClient.Put(context.Background(), reader, int64(dataLen), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	content, err := fsClient.Stat(context.Background(), StatOptions{})
	c.Assert(err, IsNil)
	c.Assert(content.Size, Equals, int64(dataLen))
}

// Test copy.
func (s *TestSuite) TestCopy(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)
	sourcePath := filepath.Join(root, "source")
	targetPath := filepath.Join(root, "target")
	fsClientTarget, err := fsNew(targetPath)
	c.Assert(err, IsNil)
	fsClientSource, err := fsNew(sourcePath)
	c.Assert(err, IsNil)

	data := "hello world"
	reader := bytes.NewReader([]byte(data))
	n, err := fsClientSource.Put(context.Background(), reader, int64(len(data)), map[string]string{
		"Content-Type": "application/octet-stream",
	}, nil, nil, false, false, false)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))
	err = fsClientTarget.Copy(context.Background(), sourcePath, CopyOptions{size: int64(len(data))}, nil)
	c.Assert(err, IsNil)
}
