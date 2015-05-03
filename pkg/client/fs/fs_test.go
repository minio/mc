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
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/minio/check"
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
	binarySum := md5.Sum([]byte(data))
	etag := base64.StdEncoding.EncodeToString(binarySum[:])
	dataLen := len(data)

	writer, err := fsc.CreateObject(etag, uint64(dataLen))
	c.Assert(err, IsNil)

	size, err := io.CopyN(writer, bytes.NewBufferString(data), int64(dataLen))
	c.Assert(err, IsNil)
	c.Assert(size, Equals, int64(dataLen))

	objectPath = filepath.Join(root, "object2")
	fsc, err = New(objectPath)
	c.Assert(err, IsNil)

	writer, err = fsc.CreateObject(etag, uint64(dataLen))
	c.Assert(err, IsNil)

	size, err = io.CopyN(writer, bytes.NewBufferString(data), int64(dataLen))
	c.Assert(err, IsNil)
	c.Assert(size, Equals, int64(dataLen))

	fsc, err = New(root)
	c.Assert(err, IsNil)

	var contents []*client.Content
	for contentCh := range fsc.ListRecursive() {
		contents = append(contents, contentCh.Content)
	}
	c.Assert(err, IsNil)
	c.Assert(len(contents), Equals, 3)
}

func (s *MySuite) TestPutBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")
	fsc, err := New(bucketPath)
	c.Assert(err, IsNil)
	err = fsc.CreateBucket()
	c.Assert(err, IsNil)
}

func (s *MySuite) TestStatBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	bucketPath := filepath.Join(root, "bucket")

	fsc, err := New(bucketPath)
	c.Assert(err, IsNil)
	err = fsc.CreateBucket()
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
	err = fsc.CreateBucket()
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
	binarySum := md5.Sum([]byte(data))
	etag := base64.StdEncoding.EncodeToString(binarySum[:])
	dataLen := len(data)

	writer, err := fsc.CreateObject(etag, uint64(dataLen))
	c.Assert(err, IsNil)

	size, err := io.CopyN(writer, bytes.NewBufferString(data), int64(dataLen))
	c.Assert(err, IsNil)
	c.Assert(size, Equals, int64(dataLen))
}

func (s *MySuite) TestGetObject(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	binarySum := md5.Sum([]byte(data))
	etag := hex.EncodeToString(binarySum[:])
	dataLen := len(data)

	writer, err := fsc.CreateObject(etag, uint64(dataLen))
	c.Assert(err, IsNil)

	_, err = io.CopyN(writer, bytes.NewBufferString(data), int64(dataLen))
	c.Assert(err, IsNil)

	reader, size, md5Sum, err := fsc.GetObject(0, 0)
	c.Assert(err, IsNil)
	var results bytes.Buffer
	c.Assert(etag, Equals, md5Sum)
	_, err = io.CopyN(&results, reader, int64(size))
	c.Assert(err, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

}

func (s *MySuite) TestStat(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "fs-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object")
	fsc, err := New(objectPath)
	c.Assert(err, IsNil)

	data := "hello"
	binarySum := md5.Sum([]byte(data))
	etag := base64.StdEncoding.EncodeToString(binarySum[:])
	dataLen := len(data)

	writer, err := fsc.CreateObject(etag, uint64(dataLen))
	c.Assert(err, IsNil)

	_, err = io.CopyN(writer, bytes.NewBufferString(data), int64(dataLen))
	c.Assert(err, IsNil)

	content, err := fsc.Stat()
	c.Assert(err, IsNil)
	c.Assert(content.Name, Equals, objectPath)
	c.Assert(content.Size, Equals, int64(dataLen))
}
