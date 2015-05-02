/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"runtime"
	"sync"
	"testing"
	"time"

	"errors"
	"net"

	"github.com/cheggaaa/pb"
	. "github.com/minio-io/check"
	"github.com/minio-io/mc/pkg/client"
	clientMocks "github.com/minio-io/mc/pkg/client/mocks"
	"github.com/minio-io/minio/pkg/iodine"
)

func Test(t *testing.T) { TestingT(t) }

type CmdTestSuite struct{}

var _ = Suite(&CmdTestSuite{})

func (s *CmdTestSuite) TestCopyToSingleTarget(c *C) {
	methods := &MockclientMethods{}
	sourceURL, err := getExpandedURL("foo", nil)
	c.Assert(err, IsNil)

	data := "Hello World"
	md5Sum := md5.Sum([]byte(data))
	hexMd5 := hex.EncodeToString(md5Sum[:])
	dataLength := int64(len(data))

	targetURL, err := getExpandedURL("bar", nil)
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

	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""

	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""

	methods.On("getSourceReader", sourceURL, sourceConfig).Return(sourceReader, dataLength, hexMd5, nil).Once()
	methods.On("getTargetWriter", targetURL, targetConfig, hexMd5, dataLength).Return(targetWriter, nil).Once()
	err = doCopySingleSource(methods, sourceURL, targetURL, sourceConfig, targetConfig)
	c.Assert(err, IsNil)
	wg.Wait()
	c.Assert(err, IsNil)
	c.Assert(resultBuffer.String(), DeepEquals, data)
	methods.AssertExpectations(c)
}

func (s *CmdTestSuite) TestCopyRecursive(c *C) {
	sourceURL, err := getExpandedURL("http://example.com/bucket1/", nil)
	c.Assert(err, IsNil)

	targetURL, err := getExpandedURL("http://example.com/bucket2/", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	wg := &sync.WaitGroup{}

	data1 := "hello world 1"
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

	methods.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	itemCh := make(chan client.ItemOnChannel)
	go func() {
		defer close(itemCh)
		for _, item := range items {
			itemCh <- client.ItemOnChannel{
				Item: item,
				Err:  nil,
			}
		}
	}()
	cl1.On("ListRecursive").Return(itemCh).Once()
	sourceReader1 := ioutil.NopCloser(bytes.NewBufferString(data1))
	sourceReader2 := ioutil.NopCloser(bytes.NewBufferString(data2))
	methods.On("getSourceReader", sourceURL+"hello1", sourceConfig).Return(sourceReader1, dataLen1, etag1, nil).Once()
	methods.On("getTargetWriter", targetURL+"hello1", targetConfig, etag1, dataLen1).Return(writer1, nil).Once()
	methods.On("getSourceReader", sourceURL+"hello2", sourceConfig).Return(sourceReader2, dataLen2, etag2, nil).Once()
	methods.On("getTargetWriter", targetURL+"hello2", targetConfig, etag2, dataLen2).Return(writer2, nil).Once()

	err = doCopySingleSourceRecursive(methods, sourceURL, targetURL, sourceConfig, targetConfig)
	c.Assert(err, IsNil)

	wg.Wait()
	c.Assert(err1, IsNil)
	c.Assert(results1.String(), Equals, data1)
	c.Assert(err2, IsNil)
	c.Assert(results2.String(), Equals, data2)

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

type failClose struct{}

func (c *failClose) Write(p []byte) (int, error) {
	return len(p), nil
}
func (c *failClose) Close() error {
	return errors.New("Expected Close Error")
}

func (s *CmdTestSuite) TestCopyCmdFailures(c *C) {
	methods := &MockclientMethods{}
	sourceURL, err := getExpandedURL("foo", nil)
	c.Assert(err, IsNil)

	targetURL, err := getExpandedURL("bar", nil)
	c.Assert(err, IsNil)

	var nilReadCloser io.ReadCloser

	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""

	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""

	methods.On("getSourceReader", sourceURL, sourceConfig).Return(nilReadCloser, int64(0), "", errors.New("Expected Error")).Once()
	err = doCopySingleSource(methods, sourceURL, targetURL, sourceConfig, targetConfig)
	c.Assert(err, Not(IsNil))
	methods.AssertExpectations(c)

	// source fails
	wg := &sync.WaitGroup{}
	data1 := "hello1"
	binarySum1 := md5.Sum([]byte(data1))
	etag1 := base64.StdEncoding.EncodeToString(binarySum1[:])
	dataLen1 := int64(len(data1))
	reader1, writer1 := io.Pipe()

	methods.On("getSourceReader", sourceURL, sourceConfig).Return(reader1, dataLen1, etag1, nil).Once()
	methods.On("getTargetWriter", targetURL, targetConfig, etag1, dataLen1).Return(nil, errors.New("Expected Error")).Once()
	err = doCopySingleSource(methods, sourceURL, targetURL, sourceConfig, targetConfig)
	writer1.Close()
	wg.Wait()
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
	methods.On("getSourceReader", sourceURL, sourceConfig).Return(reader2, dataLen1, etag1, nil).Once()
	methods.On("getTargetWriter", targetURL, targetConfig, etag1, dataLen1).Return(writer3, nil).Once()
	err = doCopySingleSource(methods, sourceURL, targetURL, sourceConfig, targetConfig)
	wg.Wait()
	c.Assert(err, Not(IsNil))
	c.Assert(n3, Equals, int64(3))

	// target close fails
	reader4, writer4 := io.Pipe()
	wg.Add(1)
	go func() {
		io.Copy(writer4, bytes.NewBufferString(data1))
		wg.Done()
	}()
	var failclose io.WriteCloser
	failclose = &failClose{}
	methods.On("getSourceReader", sourceURL, sourceConfig).Return(reader4, dataLen1, etag1, nil).Once()
	methods.On("getTargetWriter", targetURL, targetConfig, etag1, dataLen1).Return(failclose, nil).Once()
	err = doCopySingleSource(methods, sourceURL, targetURL, sourceConfig, targetConfig)
	wg.Wait()
	c.Assert(err, Not(IsNil))
}

func (s *CmdTestSuite) TestLsCmdWithBucket(c *C) {
	sourceURL, err := getExpandedURL("http://example.com/bucket1/", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	data1 := "hello world 1"
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

	methods.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	itemCh := make(chan client.ItemOnChannel)
	go func() {
		defer close(itemCh)
		for _, item := range items {
			itemCh <- client.ItemOnChannel{
				Item: item,
				Err:  nil,
			}
		}
	}()
	cl1.On("ListRecursive").Return(itemCh).Once()
	err = doListRecursiveCmd(methods, sourceURL, sourceConfig, false)
	c.Assert(err, IsNil)

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestLsCmdWithFilePath(c *C) {
	sourceURL, err := getExpandedURL("foo", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	data1 := "hello world 1"
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

	methods.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()

	itemCh := make(chan client.ItemOnChannel)
	go func() {
		defer close(itemCh)
		for _, item := range items {
			itemCh <- client.ItemOnChannel{
				Item: item,
				Err:  nil,
			}
		}
	}()
	cl1.On("ListRecursive").Return(itemCh).Once()
	err = doListRecursiveCmd(methods, sourceURL, sourceConfig, false)
	c.Assert(err, IsNil)

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestLsCmdListsBuckets(c *C) {
	sourceURL, err := getExpandedURL("http://example.com", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
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

	methods.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	itemCh := make(chan client.ItemOnChannel)
	go func() {
		defer close(itemCh)
		for _, bucket := range buckets {
			itemCh <- client.ItemOnChannel{
				Item: bucket,
				Err:  nil,
			}
		}
	}()
	cl1.On("ListRecursive").Return(itemCh).Once()
	err = doListRecursiveCmd(methods, sourceURL, sourceConfig, false)
	c.Assert(err, IsNil)

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestMbCmd(c *C) {
	targetURL, err := getExpandedURL("http://example.com/bucket1", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "").Return(nil).Once()
	msg, err := doMakeBucketCmd(methods, targetURL, targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestAccessCmd(c *C) {
	targetURL, err := getExpandedURL("http://example.com/bucket1", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "private").Return(nil).Once()
	msg, err := doUpdateAccessCmd(methods, targetURL, "private", targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "public").Return(nil).Once()
	msg, err = doUpdateAccessCmd(methods, targetURL, "public", targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "readonly").Return(nil).Once()
	msg, err = doUpdateAccessCmd(methods, targetURL, "readonly", targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestAccessCmdFailures(c *C) {
	targetURL, err := getExpandedURL("http://example.com/bucket1", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	methods.On("getNewClient", targetURL, targetConfig, false).Return(nil, errors.New("Expected Failure")).Once()
	msg, err := doMakeBucketCmd(methods, targetURL, targetConfig, false)
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "private").Return(&net.DNSError{}).Once()
	cl1.On("PutBucket", "private").Return(nil).Once()
	msg, err = doUpdateAccessCmd(methods, targetURL, "private", targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	// we use <= rather than < since the original doesn't count in the retry
	retries := globalMaxRetryFlag
	globalMaxRetryFlag = 1
	for i := 0; i <= globalMaxRetryFlag; i++ {
		err := new(net.OpError)
		err.Op = "dial"
		err.Net = "tcp"
		err.Err = errors.New("Another expected error")
		cl1.On("PutBucket", "private").Return(err).Once()
	}
	msg, err = doUpdateAccessCmd(methods, targetURL, "private", targetConfig, false)
	globalMaxRetryFlag = retries
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestMbCmdFailures(c *C) {
	targetURL, err := getExpandedURL("http://example.com/bucket1", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	methods.On("getNewClient", targetURL, targetConfig, false).Return(nil, errors.New("Expected Failure")).Once()
	msg, err := doMakeBucketCmd(methods, targetURL, targetConfig, false)
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "").Return(&net.DNSError{}).Once()
	cl1.On("PutBucket", "").Return(nil).Once()
	msg, err = doMakeBucketCmd(methods, targetURL, targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	// we use <= rather than < since the original doesn't count in the retry
	retries := globalMaxRetryFlag
	globalMaxRetryFlag = 1
	for i := 0; i <= globalMaxRetryFlag; i++ {
		err := new(net.OpError)
		err.Op = "dial"
		err.Net = "tcp"
		err.Err = errors.New("Another expected error")
		cl1.On("PutBucket", "").Return(err).Once()
	}
	msg, err = doMakeBucketCmd(methods, targetURL, targetConfig, false)
	globalMaxRetryFlag = retries
	c.Assert(len(msg) > 0, Equals, true)
	c.Assert(err, Not(IsNil))

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestAccessCmdOnFile(c *C) {
	targetURL, err := getExpandedURL("bucket1", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "private").Return(nil).Once()
	msg, err := doUpdateAccessCmd(methods, targetURL, "private", targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "public").Return(nil).Once()
	msg, err = doUpdateAccessCmd(methods, targetURL, "public", targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "readonly").Return(nil).Once()
	msg, err = doUpdateAccessCmd(methods, targetURL, "readonly", targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestMbCmdOnFile(c *C) {
	targetURL, err := getExpandedURL("bucket1", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}

	targetURLConfigMap := make(map[string]*hostConfig)
	targetConfig := new(hostConfig)
	targetConfig.AccessKeyID = ""
	targetConfig.SecretAccessKey = ""
	targetURLConfigMap[targetURL] = targetConfig

	methods.On("getNewClient", targetURL, targetConfig, false).Return(cl1, nil).Once()
	cl1.On("PutBucket", "").Return(nil).Once()
	msg, err := doMakeBucketCmd(methods, targetURL, targetConfig, false)
	c.Assert(msg, Equals, "")
	c.Assert(err, IsNil)

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
}

func (s *CmdTestSuite) TestCatCmdObject(c *C) {
	sourceURL, err := getExpandedURL("http://example.com/bucket1/object1", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}
	cl2 := &clientMocks.Client{}

	data1 := "hello world 1"
	binarySum1 := md5.Sum([]byte(data1))
	etag1 := hex.EncodeToString(binarySum1[:])
	dataLen1 := int64(len(data1))

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	sourceReader, sourceWriter := io.Pipe()
	var resultBuffer bytes.Buffer
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		io.Copy(sourceWriter, bytes.NewBufferString(data1))
		sourceWriter.Close()
		wg.Done()
	}()
	methods.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("Get").Return(sourceReader, dataLen1, etag1, nil)
	hasher := md5.New()
	mw := io.MultiWriter(os.Stdout, hasher)
	_, err = io.CopyN(mw, sourceReader, dataLen1)
	c.Assert(err, IsNil)
	msg, err := doCatCmd(methods, sourceURLConfigMap, false)
	c.Assert(msg, Not(Equals), "")
	c.Assert(err, Not(IsNil))

	// without this there will be data races
	wg.Wait()
	c.Assert(data1, Not(Equals), resultBuffer.String())

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
	cl2.AssertExpectations(c)
}

func (s *CmdTestSuite) TestCatCmdFile(c *C) {
	sourceURL, err := getExpandedURL("object1", nil)
	c.Assert(err, IsNil)

	methods := &MockclientMethods{}
	cl1 := &clientMocks.Client{}
	cl2 := &clientMocks.Client{}

	data1 := "hello world 1"
	binarySum1 := md5.Sum([]byte(data1))
	etag1 := hex.EncodeToString(binarySum1[:])
	dataLen1 := int64(len(data1))

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig := new(hostConfig)
	sourceConfig.AccessKeyID = ""
	sourceConfig.SecretAccessKey = ""
	sourceURLConfigMap[sourceURL] = sourceConfig

	sourceReader, sourceWriter := io.Pipe()
	var resultBuffer bytes.Buffer
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		io.Copy(sourceWriter, bytes.NewBufferString(data1))
		sourceWriter.Close()
		wg.Done()
	}()

	methods.On("getNewClient", sourceURL, sourceConfig, false).Return(cl1, nil).Once()
	cl1.On("Get").Return(sourceReader, dataLen1, etag1, nil)
	hasher := md5.New()
	mw := io.MultiWriter(os.Stdout, hasher)
	_, err = io.CopyN(mw, sourceReader, dataLen1)
	c.Assert(err, IsNil)
	msg, err := doCatCmd(methods, sourceURLConfigMap, false)
	c.Assert(msg, Not(Equals), "")
	c.Assert(err, Not(IsNil))

	// with this there will be data races
	wg.Wait()
	c.Assert(data1, Not(Equals), resultBuffer.String())

	methods.AssertExpectations(c)
	cl1.AssertExpectations(c)
	cl2.AssertExpectations(c)
}

func mustGetMcConfigDir() string {
	dir, _ := getMcConfigDir()
	return dir
}

func (s *CmdTestSuite) TestGetMcConfigDir(c *C) {
	u, err := user.Current()
	c.Assert(err, IsNil)
	dir, err := getMcConfigDir()
	c.Assert(err, IsNil)
	switch runtime.GOOS {
	case "linux":
		c.Assert(dir, Equals, path.Join(u.HomeDir, ".mc/"))
	case "windows":
		c.Assert(dir, Equals, path.Join(u.HomeDir, "mc/"))
	case "darwin":
		c.Assert(dir, Equals, path.Join(u.HomeDir, ".mc/"))
	default:
		c.Fail()
	}
	c.Assert(mustGetMcConfigDir(), Equals, dir)
}

func (s *CmdTestSuite) TestGetMcConfigPath(c *C) {
	dir, err := getMcConfigPath()
	c.Assert(err, IsNil)
	switch runtime.GOOS {
	case "linux":
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	case "windows":
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	case "darwin":
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	default:
		c.Fail()
	}
	c.Assert(mustGetMcConfigPath(), Equals, dir)
}

func (s *CmdTestSuite) TestIsvalidAliasName(c *C) {
	c.Check(isValidAliasName("helloWorld0"), Equals, true)
	c.Check(isValidAliasName("h0SFD2k24Fdsa"), Equals, true)
	c.Check(isValidAliasName("fdslka-4"), Equals, true)
	c.Check(isValidAliasName("fdslka-"), Equals, true)
	c.Check(isValidAliasName("helloWorld$"), Equals, false)
	c.Check(isValidAliasName("h0SFD2k2#Fdsa"), Equals, false)
	c.Check(isValidAliasName("0dslka-4"), Equals, false)
	c.Check(isValidAliasName("-fdslka"), Equals, false)
	c.Check(isValidAliasName("help"), Equals, false)
}

func (s *CmdTestSuite) TestEmptyExpansions(c *C) {
	url, err := aliasExpand("hello", nil)
	c.Assert(url, Equals, "hello")
	c.Assert(err, IsNil)

	url, err = aliasExpand("minio://hello", nil)
	c.Assert(url, Equals, "minio://hello")
	c.Assert(err, IsNil)

	url, err = aliasExpand("$#\\", nil)
	c.Assert(url, Equals, "$#\\")
	c.Assert(err, IsNil)

	url, err = aliasExpand("foo:bar", map[string]string{"foo": "http://foo"})
	c.Assert(url, Equals, "http://foo/bar")
	c.Assert(err, IsNil)

	url, err = aliasExpand("myfoo:bar", nil)
	c.Assert(url, Equals, "myfoo:bar")
	c.Assert(err, IsNil)

	url, err = aliasExpand("", nil)
	c.Assert(url, Equals, "")
	c.Assert(err, IsNil)

	url, err = aliasExpand("hello", nil)
	c.Assert(url, Equals, "hello")
	c.Assert(err, IsNil)
}

type testAddr struct{}

func (ta *testAddr) Network() string {
	return ta.String()
}
func (ta *testAddr) Error() string {
	return ta.String()
}
func (ta *testAddr) String() string {
	return "testAddr"
}

func (s *CmdTestSuite) TestStatusBar(c *C) {
	bar := startBar(1024)
	c.Assert(bar, Not(IsNil))
	c.Assert(bar.Units, Equals, pb.U_BYTES)
	c.Assert(bar.RefreshRate, Equals, time.Millisecond*10)
	c.Assert(bar.NotPrint, Equals, true)
	c.Assert(bar.ShowSpeed, Equals, true)
}

func (s *CmdTestSuite) TestIsValidRetry(c *C) {
	opError := &net.OpError{
		Op:   "read",
		Net:  "net",
		Addr: &testAddr{},
		Err:  errors.New("Op Error"),
	}
	c.Assert(isValidRetry(nil), Equals, false)
	c.Assert(isValidRetry(errors.New("hello")), Equals, false)
	c.Assert(isValidRetry(iodine.New(errors.New("hello"), nil)), Equals, false)
	c.Assert(isValidRetry(&net.DNSError{}), Equals, true)
	c.Assert(isValidRetry(iodine.New(&net.DNSError{}, nil)), Equals, true)
	// op error read
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error write
	opError.Op = "write"
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error dial
	opError.Op = "dial"
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error foo
	opError.Op = "foo"
	c.Assert(isValidRetry(opError), Equals, false)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, false)
}
