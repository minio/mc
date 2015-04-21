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
	"encoding/base64"
	"encoding/hex"
	"io"
	"sync"
	"time"

	"errors"
	"net"

	. "github.com/minio-io/check"
	"github.com/minio-io/mc/pkg/client"
	clientMocks "github.com/minio-io/mc/pkg/client/mocks"
	"io/ioutil"
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

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	manager.On("getSourceReader", sourceURL, sourceConfig).Return(sourceReader, dataLength, hexMd5, nil).Once()
	manager.On("getTargetWriter", targetURL, targetConfig, hexMd5, dataLength).Return(targetWriter, nil).Once()
	msg, err := doCopyCmd(manager, sourceURLConfigMap, targetURLConfigMap)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)
	wg.Wait()
	c.Assert(err, IsNil)
	c.Assert(resultBuffer.String(), DeepEquals, data)
	manager.AssertExpectations(c)
}

func (s *CmdTestSuite) TestCopyRecursive(c *C) {
	sourceURL, err := getURL("http://example.com/bucket1/", nil)
	c.Assert(err, IsNil)

	targetURL, err := getURL("http://example.com/bucket2/", nil)
	c.Assert(err, IsNil)

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}

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

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("List").Return(items, nil).Once()
	manager.On("getSourceReader", sourceURL+"hello1", sourceConfig).Return(ioutil.NopCloser(bytes.NewBufferString(data1)), dataLen1, etag1, nil).Once()
	manager.On("getTargetWriter", targetURL+"hello1", targetConfig, etag1, dataLen1).Return(writer1, nil).Once()
	manager.On("getSourceReader", sourceURL+"hello2", sourceConfig).Return(ioutil.NopCloser(bytes.NewBufferString(data2)), dataLen2, etag2, nil).Once()
	manager.On("getTargetWriter", targetURL+"hello2", targetConfig, etag2, dataLen2).Return(writer2, nil).Once()

	msg, err := doCopyCmdRecursive(manager, sourceURLConfigMap, targetURLConfigMap)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	wg.Wait()
	c.Assert(err1, IsNil)
	c.Assert(results1.String(), Equals, data1)
	c.Assert(err2, IsNil)
	c.Assert(results2.String(), Equals, data2)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestCopyCmdFailures(c *C) {
	manager := &MockclientManager{}
	sourceURL, err := getURL("foo", nil)
	c.Assert(err, IsNil)

	targetURL, err := getURL("bar", nil)
	c.Assert(err, IsNil)

	var nilReadCloser io.ReadCloser

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	manager.On("getSourceReader", sourceURL, sourceConfig).Return(nilReadCloser, int64(0), "", errors.New("Expected Error")).Once()
	msg, err := doCopyCmd(manager, sourceURLConfigMap, targetURLConfigMap)
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))
	manager.AssertExpectations(c)

	// source fails
	wg := &sync.WaitGroup{}
	data1 := "hello1"
	binarySum1 := md5.Sum([]byte(data1))
	etag1 := base64.StdEncoding.EncodeToString(binarySum1[:])
	dataLen1 := int64(len(data1))
	reader1, writer1 := io.Pipe()

	manager.On("getSourceReader", sourceURL, sourceConfig).Return(reader1, dataLen1, etag1, nil).Once()
	manager.On("getTargetWriter", targetURL, targetConfig, etag1, dataLen1).Return(nil, errors.New("Expected Error")).Once()
	msg, err = doCopyCmd(manager, sourceURLConfigMap, targetURLConfigMap)
	writer1.Close()
	wg.Wait()
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))

	// target write fails
	reader2, writer2 := io.Pipe()
	wg.Add(1)
	go func() {
		io.CopyN(writer2, bytes.NewBufferString("hel"), 3)
		writer2.CloseWithError(errors.New("Expected Error"))
		wg.Done()
	}()
	reader3, writer3 := io.Pipe()
	var results3 bytes.Buffer
	var n3 int64
	var err3 error
	wg.Add(1)
	go func() {
		n3, err3 = io.Copy(&results3, reader3)
		wg.Done()
	}()
	manager.On("getSourceReader", sourceURL, sourceConfig).Return(reader2, dataLen1, etag1, nil).Once()
	manager.On("getTargetWriter", targetURL, targetConfig, etag1, dataLen1).Return(writer3, nil).Once()
	msg, err = doCopyCmd(manager, sourceURLConfigMap, targetURLConfigMap)
	wg.Wait()
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))
	c.Assert(n3, Equals, int64(3))

	// target close fails
	reader4, writer4 := io.Pipe()
	wg.Add(1)
	go func() {
		io.Copy(writer4, bytes.NewBufferString(data1))
		wg.Done()
	}()
	var failClose io.WriteCloser
	failClose = &FailClose{}
	manager.On("getSourceReader", sourceURL, sourceConfig).Return(reader4, dataLen1, etag1, nil).Once()
	manager.On("getTargetWriter", targetURL, targetConfig, etag1, dataLen1).Return(failClose, nil).Once()
	msg, err = doCopyCmd(manager, sourceURLConfigMap, targetURLConfigMap)
	wg.Wait()
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))
}

type FailClose struct{}

func (c *FailClose) Write(p []byte) (int, error) {
	return len(p), nil
}
func (c *FailClose) Close() error {
	return errors.New("Expected Error")
}

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

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("List").Return(items, nil).Once()
	msg, err := doListCmd(manager, sourceURLConfigMap, false)
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

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("List").Return(items, nil).Once()
	msg, err := doListCmd(manager, sourceURLConfigMap, false)
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

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("List").Return(buckets, nil).Once()
	msg, err := doListCmd(manager, sourceURLConfigMap, false)
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

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket").Return(nil).Once()
	msg, err := doMakeBucketCmd(manager, sourceURLConfigMap, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestMbCmdFailures(c *C) {
	sourceURL, err := getURL("http://example.com/bucket1", nil)
	c.Assert(err, IsNil)

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(nil, errors.New("Expected Failure")).Once()
	msg, err := doMakeBucketCmd(manager, sourceURLConfigMap, false)
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket").Return(&net.DNSError{}).Once()
	cl1.On("PutBucket").Return(nil).Once()
	msg, err = doMakeBucketCmd(manager, sourceURLConfigMap, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	// we use <= rather than < since the original doesn't count in the retry
	retries := globalMaxRetryFlag
	globalMaxRetryFlag = 1
	for i := 0; i <= globalMaxRetryFlag; i++ {
		cl1.On("PutBucket").Return(errors.New("Another Expected Error")).Once()
	}
	msg, err = doMakeBucketCmd(manager, sourceURLConfigMap, false)
	globalMaxRetryFlag = retries
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestMbCmdOnFile(c *C) {
	sourceURL, err := getURL("bucket1", nil)
	c.Assert(err, IsNil)

	manager := &MockclientManager{}
	cl1 := &clientMocks.Client{}

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	manager.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket").Return(nil).Once()
	msg, err := doMakeBucketCmd(manager, sourceURLConfigMap, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	manager.AssertExpectations(c)
	cl1.AssertExpectations(c)
}
