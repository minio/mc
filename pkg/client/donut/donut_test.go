/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
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

package donut

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/minio-io/mc/pkg/client"
)

const (
	globalMaxKeys = 1000 // Maximum number of keys to fetch per request
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func setupNodeDiskMap(c *C) map[string][]string {
	var disks []string
	for i := 0; i < 16; i++ {
		root, err := ioutil.TempDir(os.TempDir(), "donut-")
		c.Assert(err, IsNil)
		disks = append(disks, root)
	}
	nodeDiskMap := make(map[string][]string)
	nodeDiskMap["localhost"] = disks
	return nodeDiskMap
}

func removeDisks(c *C, disks []string) {
	for _, disk := range disks {
		err := os.RemoveAll(disk)
		c.Assert(err, IsNil)
	}
}

func (s *MySuite) TestEmptyBucket(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	// check buckets are empty
	buckets, err := donut.ListBuckets()
	c.Assert(err, IsNil)
	c.Assert(len(buckets), Equals, 0)
}

func (s *MySuite) TestBucketWithoutNameFails(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	// fail to create new bucket without a name
	err = donut.PutBucket("")
	c.Assert(err, Not(IsNil))

	err = donut.PutBucket(" ")
	c.Assert(err, Not(IsNil))
}

func (s *MySuite) TestCreateBucketAndList(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	// make bucket
	err = donut.PutBucket("foo")
	c.Assert(err, IsNil)

	// check bucket exists
	buckets, err := donut.ListBuckets()
	c.Assert(err, IsNil)
	c.Assert(len(buckets), Equals, 1)
	c.Assert(buckets[0].Name, Equals, "foo")
}

func (s *MySuite) TestCreateBucketWithSameNameFails(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	// make bucket
	err = donut.PutBucket("foo")
	c.Assert(err, IsNil)

	// make bucket fail
	err = donut.PutBucket("foo")
	c.Assert(err, Not(IsNil))
}

func getBucketNames(buckets []*client.Bucket) []string {
	var bucketNames []string
	for _, bucket := range buckets {
		bucketNames = append(bucketNames, bucket.Name)
	}
	return bucketNames
}

func (s *MySuite) TestCreateMultipleBucketsAndList(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	err = donut.PutBucket("foo")
	c.Assert(err, IsNil)

	err = donut.PutBucket("bar")
	c.Assert(err, IsNil)

	buckets, err := donut.ListBuckets()
	c.Assert(err, IsNil)

	bucketNamesReceived := getBucketNames(buckets)
	bucketNamesProvided := []string{"bar", "foo"}
	c.Assert(bucketNamesReceived, DeepEquals, bucketNamesProvided)

	err = donut.PutBucket("foobar")
	c.Assert(err, IsNil)
	bucketNamesProvided = append(bucketNamesProvided, "foobar")

	buckets, err = donut.ListBuckets()
	c.Assert(err, IsNil)
	bucketNamesReceived = getBucketNames(buckets)
	c.Assert(bucketNamesReceived, DeepEquals, bucketNamesProvided)
}

func (s *MySuite) TestNewObjectFailsWithoutBucket(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	_, _, _, err = donut.Get("foo", "obj")
	c.Assert(err, Not(IsNil))
}

func (s *MySuite) TestNewObjectFailsWithEmptyName(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	_, _, _, err = donut.Get("foo", "")
	c.Assert(err, Not(IsNil))

	_, _, _, err = donut.Get("foo", " ")
	c.Assert(err, Not(IsNil))
}

func (s *MySuite) TestNewObjectCanBeWritten(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	err = donut.PutBucket("foo")
	c.Assert(err, IsNil)

	m := md5.New()
	data := "Hello World"
	io.WriteString(m, data)
	err = donut.Put("foo", "obj", hex.EncodeToString(m.Sum(nil)), int64(len(data)), bytes.NewBuffer([]byte(data)))
	c.Assert(err, IsNil)

	reader, size, _, err := donut.Get("foo", "obj")
	c.Assert(err, IsNil)

	var actualData bytes.Buffer
	_, err = io.Copy(&actualData, reader)
	c.Assert(err, IsNil)
	c.Assert(actualData.Bytes(), DeepEquals, []byte(data))
	c.Assert(int64(actualData.Len()), Equals, size)
}

func getObjectNames(objects []*client.Item) []string {
	var objectNames []string
	for _, object := range objects {
		objectNames = append(objectNames, object.Key)
	}
	return objectNames
}

func (s *MySuite) TestMultipleNewObjects(c *C) {
	nodeDiskMap := setupNodeDiskMap(c)
	donut, err := GetNewClient("testemptydonut", nodeDiskMap)
	defer removeDisks(c, nodeDiskMap["localhost"])
	c.Assert(err, IsNil)

	c.Assert(donut.PutBucket("foo"), IsNil)

	m1 := md5.New()
	m2 := md5.New()
	io.WriteString(m1, "one")
	io.WriteString(m2, "two")

	err = donut.Put("foo", "obj1",
		hex.EncodeToString(m1.Sum(nil)), int64(len([]byte("one"))), bytes.NewBuffer([]byte("one")))
	c.Assert(err, IsNil)

	err = donut.Put("foo", "obj2",
		hex.EncodeToString(m2.Sum(nil)), int64(len([]byte("two"))), bytes.NewBuffer([]byte("two")))
	c.Assert(err, IsNil)

	reader, _, _, err := donut.Get("foo", "obj1")
	c.Assert(err, IsNil)

	var readerBuffer1 bytes.Buffer
	_, err = io.Copy(&readerBuffer1, reader)
	c.Assert(err, IsNil)
	c.Assert(readerBuffer1.Bytes(), DeepEquals, []byte("one"))

	reader, _, _, err = donut.Get("foo", "obj2")
	c.Assert(err, IsNil)

	var readerBuffer2 bytes.Buffer
	_, err = io.Copy(&readerBuffer2, reader)
	c.Assert(err, IsNil)
	c.Assert(readerBuffer2.Bytes(), DeepEquals, []byte("two"))

	// test list objects
	listObjects, _, err := donut.ListObjects("foo", "", "", "", globalMaxKeys)
	c.Assert(err, IsNil)

	receivedObjectNames := getObjectNames(listObjects)
	c.Assert(receivedObjectNames, DeepEquals, []string{"obj1", "obj2"})
}
