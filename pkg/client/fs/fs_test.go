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
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object1")
	fsc, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"

	err = fsc.Put(bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, IsNil)

	objectPath = filepath.Join(root, "object2")
	fsc, err = fs.New(objectPath)
	c.Assert(err, IsNil)

	err = fsc.Put(bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, IsNil)

	fsc, err = fs.New(root)
	c.Assert(err, IsNil)

	var contents []*client.Content
	for content := range fsc.List(false, false) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 1)
	c.Assert(contents[0].Type.IsDir(), Equals, true)

	objectPath = filepath.Join(root, "test1/newObject1")
	fsc, err = fs.New(objectPath)
	c.Assert(err, IsNil)

	err = fsc.Put(bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, IsNil)

	fsc, err = fs.New(root)
	c.Assert(err, IsNil)

	contents = nil
	for content := range fsc.List(false, false) {
		if content.Err != nil {
			err = content.Err
			break
		}
		contents = append(contents, content)
	}
	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 1)
	c.Assert(contents[0].Type.IsDir(), Equals, true)

	fsc, err = fs.New(root)
	c.Assert(err, IsNil)

	contents = nil
	for content := range fsc.List(true, false) {
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
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsc, err := fs.New(bucketPath)
	c.Assert(err, IsNil)
	err = fsc.MakeBucket()
	c.Assert(err, IsNil)
}

func (s *MySuite) TestStatBucket(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")

	fsc, err := fs.New(bucketPath)
	c.Assert(err, IsNil)
	err = fsc.MakeBucket()
	c.Assert(err, IsNil)
	_, err = fsc.Stat()
	c.Assert(err, IsNil)
}

func (s *MySuite) TestBucketACLFails(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsc, err := fs.New(bucketPath)
	c.Assert(err, IsNil)
	err = fsc.MakeBucket()
	c.Assert(err, IsNil)

	err = fsc.SetBucketAccess("private")
	c.Assert(err, Not(IsNil))

	_, err = fsc.GetBucketAccess()
	c.Assert(err, Not(IsNil))
}

func (s *MySuite) TestPut(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	err = fsc.Put(bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, IsNil)
}

func (s *MySuite) TestGet(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"

	err = fsc.Put(bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, IsNil)

	reader, err := fsc.Get(0, 0)
	c.Assert(err, IsNil)
	var results bytes.Buffer
	_, e = io.Copy(&results, reader)
	c.Assert(e, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

}

func (s *MySuite) TestGetRange(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello world"

	err = fsc.Put(bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, IsNil)

	reader, err := fsc.Get(0, 5)
	c.Assert(err, IsNil)
	var results bytes.Buffer
	_, e = io.Copy(&results, reader)
	c.Assert(e, IsNil)
	c.Assert([]byte("hello"), DeepEquals, results.Bytes())
}

func (s *MySuite) TestStatObject(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := fs.New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	dataLen := len(data)

	err = fsc.Put(bytes.NewReader([]byte(data)), int64(len(data)))
	c.Assert(err, IsNil)

	content, err := fsc.Stat()
	c.Assert(err, IsNil)
	c.Assert(content.Size, Equals, int64(dataLen))
}
