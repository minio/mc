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

package fs_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/client/fs"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

// Test list files in a folder.
func (s *MySuite) TestList(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	// Create multiple files.
	objectPath := filepath.Join(root, "object1")
	fsClient, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"

	reader := bytes.NewReader([]byte(data))
	var n int64
	n, err = fsClient.Put(reader, int64(len(data)), "application/octet-stream", nil)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	objectPath = filepath.Join(root, "object2")
	fsClient, err = fs.New(objectPath)
	c.Assert(err, IsNil)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(reader, int64(len(data)), "application/octet-stream", nil)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	fsClient, err = fs.New(root)
	c.Assert(err, IsNil)

	// Verify previously create files and list them.
	var contents []*client.Content
	for content := range fsClient.List(false, false) {
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
	fsClient, err = fs.New(objectPath)
	c.Assert(err, IsNil)

	reader = bytes.NewReader([]byte(data))
	n, err = fsClient.Put(reader, int64(len(data)), "application/octet-stream", nil)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	fsClient, err = fs.New(root)
	c.Assert(err, IsNil)

	contents = nil
	// List non recursive to list only top level files.
	for content := range fsClient.List(false, false) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 1)
	c.Assert(contents[0].Type.IsDir(), Equals, true)

	fsClient, err = fs.New(root)
	c.Assert(err, IsNil)

	contents = nil
	// List recursively all files and verify.
	for content := range fsClient.List(true, false) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}

	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 5)

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
	c.Assert(regularDirs, Equals, 2)
	c.Assert(regularFiles, Equals, 3)
}

// Test put bucket aka 'mkdir()' operation.
func (s *MySuite) TestPutBucket(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsClient, err := fs.New(bucketPath)
	c.Assert(err, IsNil)
	err = fsClient.MakeBucket("us-east-1")
	c.Assert(err, IsNil)
}

// Test stat bucket aka 'stat()' operation.
func (s *MySuite) TestStatBucket(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")

	fsClient, err := fs.New(bucketPath)
	c.Assert(err, IsNil)
	err = fsClient.MakeBucket("us-east-1")
	c.Assert(err, IsNil)
	_, err = fsClient.Stat()
	c.Assert(err, IsNil)
}

// Test bucket acl fails for directories.
func (s *MySuite) TestBucketACLFails(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsClient, err := fs.New(bucketPath)
	c.Assert(err, IsNil)
	err = fsClient.MakeBucket("us-east-1")
	c.Assert(err, IsNil)

	err = fsClient.SetBucketAccess("private")
	c.Assert(err, Not(IsNil))

	_, err = fsClient.GetBucketAccess()
	c.Assert(err, Not(IsNil))
}

// Test creating a file.
func (s *MySuite) TestPut(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	reader := bytes.NewReader([]byte(data))
	var n int64
	n, err = fsClient.Put(reader, int64(len(data)), "application/octet-stream", nil)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))
}

// Test read a file.
func (s *MySuite) TestGet(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	var reader io.Reader
	reader = bytes.NewReader([]byte(data))
	n, err := fsClient.Put(reader, int64(len(data)), "application/octet-stream", nil)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	reader, err = fsClient.Get()
	c.Assert(err, IsNil)
	var results bytes.Buffer
	_, e = io.Copy(&results, reader)
	c.Assert(e, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

}

// Test get range in a file.
func (s *MySuite) TestGetRange(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello world"
	var reader io.Reader
	reader = bytes.NewReader([]byte(data))
	n, err := fsClient.Put(reader, int64(len(data)), "application/octet-stream", nil)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	reader, err = fsClient.Get()
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
func (s *MySuite) TestStatObject(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsClient, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	dataLen := len(data)
	reader := bytes.NewReader([]byte(data))
	n, err := fsClient.Put(reader, int64(dataLen), "application/octet-stream", nil)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, int64(len(data)))

	content, err := fsClient.Stat()
	c.Assert(err, IsNil)
	c.Assert(content.Size, Equals, int64(dataLen))
}
