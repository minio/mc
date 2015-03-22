package donut

import (
	"testing"

	"bytes"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"os"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestEmptyBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)

	// check buckets are empty
	buckets, err := donut.ListBuckets()
	c.Assert(err, IsNil)
	c.Assert(buckets, IsNil)
}

func (s *MySuite) TestBucketWithoutNameFails(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)
	// fail to create new bucket without a name
	err = donut.PutBucket("")
	c.Assert(err, Not(IsNil))

	err = donut.PutBucket(" ")
	c.Assert(err, Not(IsNil))
}

func (s *MySuite) TestPutBucketAndList(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)
	// create bucket
	err = donut.PutBucket("foo")
	c.Assert(err, IsNil)

	// check bucket exists
	buckets, err := donut.ListBuckets()
	c.Assert(err, IsNil)
	c.Assert(buckets, DeepEquals, []string{"foo"})
}

func (s *MySuite) TestPutBucketWithSameNameFails(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)
	err = donut.PutBucket("foo")
	c.Assert(err, IsNil)

	err = donut.PutBucket("foo")
	c.Assert(err, Not(IsNil))
}

func (s *MySuite) TestCreateMultipleBucketsAndList(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)
	// add a second bucket
	err = donut.PutBucket("foo")
	c.Assert(err, IsNil)

	err = donut.PutBucket("bar")
	c.Assert(err, IsNil)

	buckets, err := donut.ListBuckets()
	c.Assert(err, IsNil)
	c.Assert(buckets, DeepEquals, []string{"bar", "foo"})

	err = donut.PutBucket("foobar")
	c.Assert(err, IsNil)

	buckets, err = donut.ListBuckets()
	c.Assert(err, IsNil)
	c.Assert(buckets, DeepEquals, []string{"bar", "foo", "foobar"})
}

func (s *MySuite) TestNewObjectFailsWithoutBucket(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)

	writer, err := donut.Put("foo", "obj")
	c.Assert(err, Not(IsNil))
	c.Assert(writer, IsNil)
}

func (s *MySuite) TestNewObjectFailsWithEmptyName(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)

	writer, err := donut.Put("foo", "")
	c.Assert(err, Not(IsNil))
	c.Assert(writer, IsNil)

	writer, err = donut.Put("foo", " ")
	c.Assert(err, Not(IsNil))
	c.Assert(writer, IsNil)
}

func (s *MySuite) TestNewObjectCanBeWritten(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)

	err = donut.PutBucket("foo")
	c.Assert(err, IsNil)

	writer, err := donut.Put("foo", "obj")
	c.Assert(err, IsNil)

	data := "Hello World"
	length, err := writer.Write([]byte(data))
	c.Assert(length, Equals, len(data))

	expectedMetadata := map[string]string{
		"foo":     "bar",
		"created": "one",
		"hello":   "world",
	}

	err = writer.SetMetadata(expectedMetadata)
	c.Assert(err, IsNil)

	err = writer.Close()
	c.Assert(err, IsNil)

	actualWriterMetadata, err := writer.GetMetadata()
	c.Assert(err, IsNil)
	c.Assert(actualWriterMetadata, DeepEquals, expectedMetadata)

	c.Assert(err, IsNil)

	reader, _, err := donut.Get("foo", "obj")
	c.Assert(err, IsNil)

	var actualData bytes.Buffer
	_, err = io.Copy(&actualData, reader)
	c.Assert(err, IsNil)
	c.Assert(actualData.Bytes(), DeepEquals, []byte(data))

	actualMetadata, err := donut.Stat("foo", "obj")
	c.Assert(err, IsNil)
	c.Assert(actualMetadata, DeepEquals, expectedMetadata)
}

func (s *MySuite) TestMultipleNewObjects(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "donut-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	donut := NewDriver(root)

	c.Assert(donut.PutBucket("foo"), IsNil)
	writer, err := donut.Put("foo", "obj1")
	c.Assert(err, IsNil)
	writer.Write([]byte("one"))
	writer.Close()

	writer, err = donut.Put("foo", "obj2")
	c.Assert(err, IsNil)
	writer.Write([]byte("two"))
	writer.Close()

	//	c.Skip("not complete")

	reader, _, err := donut.Get("foo", "obj1")
	c.Assert(err, IsNil)
	var readerBuffer1 bytes.Buffer
	_, err = io.Copy(&readerBuffer1, reader)
	c.Assert(err, IsNil)
	//	c.Skip("Not Implemented")
	c.Assert(readerBuffer1.Bytes(), DeepEquals, []byte("one"))

	reader, _, err = donut.Get("foo", "obj2")
	c.Assert(err, IsNil)
	var readerBuffer2 bytes.Buffer
	_, err = io.Copy(&readerBuffer2, reader)
	c.Assert(err, IsNil)
	c.Assert(readerBuffer2.Bytes(), DeepEquals, []byte("two"))

	// test list objects
	listObjects, err := donut.ListObjects("foo")
	c.Assert(err, IsNil)
	c.Assert(listObjects, DeepEquals, []string{"obj1", "obj2"})
}
