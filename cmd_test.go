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

	"errors"
	. "github.com/minio-io/check"

	"github.com/minio-io/mc/pkg/client"
	clientMocks "github.com/minio-io/mc/pkg/client/mocks"
)

type CmdTestSuite struct{}

var _ = Suite(&CmdTestSuite{})

func (s *CmdTestSuite) TestCopyToSingleTarget(c *C) {
	manager := &MockclientManager{}
	sourceURL, err := parseURL("foo", nil)
	c.Assert(err, IsNil)

	data := "Hello World"
	md5Sum := md5.Sum([]byte(data))
	hexMd5 := hex.EncodeToString(md5Sum[:])
	dataLength := int64(len(data))

	targetURL, err := parseURL("bar", nil)
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
	doCopyCmd(manager, sourceURL, targetURLs)
	wg.Wait()
	c.Assert(err, IsNil)
	c.Assert(resultBuffer.String(), DeepEquals, data)
}

func (s *CmdTestSuite) TestCopyRecursive(c *C) {
	c.Skip("Incomplete")
	sourceURL, err := parseURL("foo", nil)
	c.Assert(err, IsNil)

	targetURL, err := parseURL("bar", nil)
	c.Assert(err, IsNil)
	targetURLs := []string{targetURL}

	var mockClient client.Client
	manager := &MockclientManager{}
	mockClient = &clientMocks.Client{}

	manager.On("getNewClient", sourceURL, false).Return(mockClient, errors.New("foo")).Once()
	doCopyCmdRecursive(manager, sourceURL, targetURLs)
}
