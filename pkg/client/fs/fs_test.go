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

func (s *MySuite) TestList(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object1")
	fsc, perr := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	dataLen := len(data)

	perr = fsc.Put(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	objectPath = filepath.Join(root, "object2")
	fsc, perr = fs.New(objectPath)
	c.Assert(err, IsNil)

	perr = fsc.Put(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	fsc, perr = fs.New(root)
	c.Assert(err, IsNil)

	var contents []*client.Content
	for contentCh := range fsc.List(false, false) {
		if contentCh.Err != nil {
			perr = contentCh.Err
			break
		}
		contents = append(contents, contentCh.Content)
	}
	c.Assert(perr, IsNil)
	c.Assert(len(contents), Equals, 2)

	for _, content := range contents {
		c.Assert(content.Type.IsRegular(), Equals, true)
	}

	objectPath = filepath.Join(root, "test1/newObject1")
	fsc, perr = fs.New(objectPath)
	c.Assert(err, IsNil)

	perr = fsc.Put(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	fsc, perr = fs.New(root)
	c.Assert(err, IsNil)

	contents = nil
	for contentCh := range fsc.List(false, false) {
		if contentCh.Err != nil {
			perr = contentCh.Err
			break
		}
		contents = append(contents, contentCh.Content)
	}
	c.Assert(perr, IsNil)
	c.Assert(len(contents), Equals, 3)

	for _, content := range contents {
		// skip previous regular files
		if content.Type.IsRegular() {
			continue
		}
		c.Assert(content.Type.IsDir(), Equals, true)
	}

	fsc, perr = fs.New(root)
	c.Assert(err, IsNil)

	contents = nil
	for contentCh := range fsc.List(true, false) {
		if contentCh.Err != nil {
			perr = contentCh.Err
			break
		}
		contents = append(contents, contentCh.Content)
	}

	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 5)

	var regularFiles int
	var regularDirs int
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

func (s *MySuite) TestPutBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsc, perr := fs.New(bucketPath)
	c.Assert(perr, IsNil)
	perr = fsc.MakeBucket()
	c.Assert(perr, IsNil)
}

func (s *MySuite) TestStatBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")

	fsc, perr := fs.New(bucketPath)
	c.Assert(perr, IsNil)
	perr = fsc.MakeBucket()
	c.Assert(perr, IsNil)
	_, perr = fsc.Stat()
	c.Assert(perr, IsNil)
}

func (s *MySuite) TestBucketACLFails(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsc, perr := fs.New(bucketPath)
	c.Assert(perr, IsNil)
	perr = fsc.MakeBucket()
	c.Assert(perr, IsNil)

	perr = fsc.SetBucketAccess("private")
	c.Assert(perr, Not(IsNil))

	_, perr = fsc.GetBucketAccess()
	c.Assert(perr, Not(IsNil))
}

func (s *MySuite) TestPut(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, perr := fs.New(objectPath)
	c.Assert(perr, IsNil)

	data := "hello"
	dataLen := len(data)

	perr = fsc.Put(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(perr, IsNil)
}

func (s *MySuite) TestGet(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, perr := fs.New(objectPath)
	c.Assert(perr, IsNil)

	data := "hello"
	dataLen := len(data)

	perr = fsc.Put(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(perr, IsNil)

	reader, size, perr := fsc.Get(0, 0)
	c.Assert(perr, IsNil)
	var results bytes.Buffer
	_, err = io.CopyN(&results, reader, int64(size))
	c.Assert(err, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

}

func (s *MySuite) TestGetRange(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, perr := fs.New(objectPath)
	c.Assert(perr, IsNil)

	data := "hello world"
	dataLen := len(data)

	perr = fsc.Put(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(perr, IsNil)

	reader, size, perr := fsc.Get(0, 5)
	c.Assert(perr, IsNil)
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
	fsc, perr := fs.New(objectPath)
	c.Assert(perr, IsNil)

	data := "hello"
	dataLen := len(data)

	perr = fsc.Put(int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(perr, IsNil)

	content, perr := fsc.Stat()
	c.Assert(perr, IsNil)
	c.Assert(content.Size, Equals, int64(dataLen))
}
