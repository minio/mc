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
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "gopkg.in/check.v1"
	"github.com/minio/mc/pkg/client"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestList(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object1")
	fsc, err := New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	dataLen := len(data)

	err = fsc.PutObject(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	objectPath = filepath.Join(root, "object2")
	fsc, err = New(objectPath)
	c.Assert(err, IsNil)

	err = fsc.PutObject(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	fsc, err = New(root)
	c.Assert(err, IsNil)

	var contents []*client.Content
	for contentCh := range fsc.List(false) {
		contents = append(contents, contentCh.Content)
	}
	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 2)

	for _, content := range contents {
		c.Assert(content.Type.IsRegular(), Equals, true)
	}

	objectPath = filepath.Join(root, "test1/newObject1")
	fsc, err = New(objectPath)
	c.Assert(err, IsNil)

	err = fsc.PutObject(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	fsc, err = New(root)
	c.Assert(err, IsNil)

	contents = nil
	for contentCh := range fsc.List(false) {
		contents = append(contents, contentCh.Content)
	}
	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 3)

	for _, content := range contents {
		// skip previous regular files
		if content.Type.IsRegular() {
			continue
		}
		c.Assert(content.Type.IsDir(), Equals, true)
	}

	fsc, err = New(root)
	c.Assert(err, IsNil)

	contents = nil
	for contentCh := range fsc.List(true) {
		contents = append(contents, contentCh.Content)
	}

	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 4)

	var regularFiles int
	var directories int
	for _, content := range contents {
		if content.Type.IsRegular() {
			regularFiles++
			continue
		}
		if content.Type.IsDir() {
			directories++
		}
	}
	c.Assert(regularFiles, Equals, 3)
	c.Assert(directories, Equals, 1)
}

func (s *MySuite) TestPutBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsc, err := New(bucketPath)
	c.Assert(err, IsNil)
	err = fsc.MakeBucket()
	c.Assert(err, IsNil)
}

func (s *MySuite) TestStatBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")

	fsc, err := New(bucketPath)
	c.Assert(err, IsNil)
	err = fsc.MakeBucket()
	c.Assert(err, IsNil)
	_, err = fsc.Stat()
	c.Assert(err, IsNil)
}

func (s *MySuite) TestPutBucketACL(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsc, err := New(bucketPath)
	c.Assert(err, IsNil)
	err = fsc.MakeBucket()
	c.Assert(err, IsNil)

	err = fsc.SetBucketACL("private")
	c.Assert(err, IsNil)
}

func (s *MySuite) TestPutObject(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	dataLen := len(data)

	err = fsc.PutObject(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)
}

func (s *MySuite) TestGetObject(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	dataLen := len(data)

	err = fsc.PutObject(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	reader, size, err := fsc.GetObject(0, 0)
	c.Assert(err, IsNil)
	var results bytes.Buffer
	_, err = io.CopyN(&results, reader, int64(size))
	c.Assert(err, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

}

func (s *MySuite) TestGetObjectRange(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := New(objectPath)
	c.Assert(err, IsNil)

	data := "hello world"
	dataLen := len(data)

	err = fsc.PutObject(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	reader, size, err := fsc.GetObject(0, 5)
	c.Assert(err, IsNil)
	var results bytes.Buffer
	_, err = io.CopyN(&results, reader, int64(size))
	c.Assert(err, IsNil)
	c.Assert([]byte("hello"), DeepEquals, results.Bytes())
}

func (s *MySuite) TestStatObject(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	dataLen := len(data)

	err = fsc.PutObject(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	content, err := fsc.Stat()
	c.Assert(err, IsNil)
	c.Assert(content.Name, Equals, objectPath)
	c.Assert(content.Size, Equals, int64(dataLen))
}
