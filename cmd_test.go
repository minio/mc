/*
 * Mini Copy, (C) 2014, 2015 Minio, Inc.
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
package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"sync"
	"time"

	. "github.com/minio-io/check"
	"github.com/minio-io/mc/pkg/client"
	clientMocks "github.com/minio-io/mc/pkg/client/mocks"
)

type CmdTestSuite struct{}

var _ = Suite(&CmdTestSuite{})

func (s *CmdTestSuite) TestCopyToSingleTarget(c *C) {
	manager := &MockclientManager{}
	sourceURL, err := getURL("foo", nil)
	c.Assert(err, IsNil)

	data := "Hello World"
	md5Sum := md5.Sum([]byte(data))
	hexMd5 := hex.EncodeToString(md5Sum[:])
	dataLength := int64(len(data))

	targetURL, err := getURL("bar", nil)
	c.Assert(err, IsNil)
	targetURLs := []string{targetURL}

	sourceReader, sourceWriter := io.Pipe()
	targetReader, targetWriter := io.Pipe()
	var resultBuffer bytes.Buffer
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		io.Copy(sourceWriter, bytes.NewBufferString("Hello World"))
		sourceWriter.Close()
		wg.Done()
	}()
	go func() {
		io.Copy(&resultBuffer, targetReader)
		wg.Done()
	}()
	manager.On("getSourceReader", sourceURL).Return(sourceReader, dataLength, hexMd5, nil).Once()
	manager.On("getTargetWriter", targetURL, hexMd5, dataLength).Return(targetWriter, nil).Once()
	msg, err := doCopyCmd(manager, sourceURL, targetURLs)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)
	wg.Wait()
	c.Assert(err, IsNil)
	c.Assert(resultBuffer.String(), DeepEquals, data)
	manager.AssertExpectations(c)
}

/* - TODO(y4m4) - enable this after figuring out changes with getNewClient behavior in doCmdRecursive()
func (s *CmdTestSuite) TestCopyRecursive(c *C) {
	sourceURL, err := getURL("http://example.com/bucket1/", nil)
	c.Assert(err, IsNil)

	targetURL, err := getURL("http://example.com/bucket2/", nil)
	c.Assert(err, IsNil)
	targetURLs := []string{targetURL}

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}
	cl2 := &clientMocks.Client{}
	cl3 := &clientMocks.Client{}

	wg := &sync.WaitGroup{}

	data1 := "hello1"
	binarySum1 := md5.Sum([]byte(data1))
	etag1 := base64.StdEncoding.EncodeToString(binarySum1[:])
	dataLen1 := int64(len(data1))
	reader1, writer1 := io.Pipe()
	var results1 bytes.Buffer
	var err1 error
	wg.Add(1)
	go func() {
		_, err1 = io.Copy(&results1, reader1)
		wg.Done()
	}()

	data2 := "hello world 2"
	binarySum2 := md5.Sum([]byte(data2))
	etag2 := base64.StdEncoding.EncodeToString(binarySum2[:])
	dataLen2 := int64(len(data2))
	reader2, writer2 := io.Pipe()
	var err2 error
	var results2 bytes.Buffer
	wg.Add(1)
	go func() {
		_, err2 = io.Copy(&results2, reader2)
		wg.Done()
	}()

	items := []*client.Item{
		{Name: "hello1", Time: time.Now(), Size: dataLen1},
		{Name: "hello2", Time: time.Now(), Size: dataLen2},
	}

	manager.On("getNewClient", sourceURL, false).Return(cl1, nil).Once()
	cl1.On("List").Return(items, nil).Once()
	cl1.On("Get").Return(ioutil.NopCloser(bytes.NewBufferString(data1)), dataLen1, etag1, nil).Once()
	manager.On("getNewClient", targetURL+"hello2", false).Return(cl2, nil).Once()
	cl2.On("StatBucket").Return(nil).Once()
	cl2.On("Put", etag1, dataLen1).Return(writer1, nil).Once()
	cl1.On("Get").Return(ioutil.NopCloser(bytes.NewBufferString(data2)), dataLen2, etag2, nil).Once()
	manager.On("getNewClient", targetURL+"hello1", false).Return(cl3, nil).Once()
	cl3.On("StatBucket").Return(nil).Once()
	cl3.On("Put", etag2, dataLen2).Return(writer2, nil).Once()
	doCopyCmdRecursive(manager, sourceURL, targetURLs)

	wg.Wait()
	c.Assert(err1, IsNil)
	c.Assert(results1.String(), Equals, data1)
	c.Assert(err2, IsNil)
	c.Assert(results2.String(), Equals, data2)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
	cl2.AssertExpectations(c)
	cl3.AssertExpectations(c)
}
*/

func (s *CmdTestSuite) TestLsCmdWithBucket(c *C) {
	sourceURL, err := getURL("http://example.com/bucket1/", nil)
	c.Assert(err, IsNil)

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}

	data1 := "hello1"
	dataLen1 := int64(len(data1))

	data2 := "hello world 2"
	dataLen2 := int64(len(data2))

	items := []*client.Item{
		{Name: "hello1", Time: time.Now(), Size: dataLen1},
		{Name: "hello2", Time: time.Now(), Size: dataLen2},
	}

	manager.On("getNewClient", sourceURL, false).Return(cl1, nil).Once()
	cl1.On("List").Return(items, nil).Once()
	msg, err := doListCmd(manager, sourceURL, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestLsCmdWithFilePath(c *C) {
	sourceURL, err := getURL("foo", nil)
	c.Assert(err, IsNil)

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}

	data1 := "hello1"
	dataLen1 := int64(len(data1))

	data2 := "hello world 2"
	dataLen2 := int64(len(data2))

	items := []*client.Item{
		{Name: "hello1", Time: time.Now(), Size: dataLen1},
		{Name: "hello2", Time: time.Now(), Size: dataLen2},
	}

	manager.On("getNewClient", sourceURL, false).Return(cl1, nil).Once()
	cl1.On("List").Return(items, nil).Once()
	msg, err := doListCmd(manager, sourceURL, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestLsCmdListsBuckets(c *C) {
	sourceURL, err := getURL("http://example.com", nil)
	c.Assert(err, IsNil)

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}

	buckets := []*client.Item{
		{Name: "bucket1", Time: time.Now()},
		{Name: "bucket2", Time: time.Now()},
	}

	manager.On("getNewClient", sourceURL, false).Return(cl1, nil).Once()
	cl1.On("List").Return(buckets, nil).Once()
	msg, err := doListCmd(manager, sourceURL, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestMbCmd(c *C) {
	sourceURL, err := getURL("http://example.com/bucket1", nil)
	c.Assert(err, IsNil)

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}

	manager.On("getNewClient", sourceURL, false).Return(cl1, nil).Once()
	cl1.On("PutBucket").Return(nil).Once()
	msg, err := doMakeBucketCmd(manager, sourceURL, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestMbCmdOnFile(c *C) {
	sourceURL, err := getURL("bucket1", nil)
	c.Assert(err, IsNil)

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}

	manager.On("getNewClient", sourceURL, false).Return(cl1, nil).Once()
	cl1.On("PutBucket").Return(nil).Once()
	msg, err := doMakeBucketCmd(manager, sourceURL, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}
