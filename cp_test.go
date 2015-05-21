/*
 * Minio Client (C) 2015 Minio, Inc.
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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	. "github.com/minio/check"
)

var bar barSend

func (s *CmdTestSuite) TestCpTypeA(c *C) {
	globalQuietFlag = true
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	sourcePath := filepath.Join(source, "object1")
	data := "hello"
	dataLen := len(data)
	err = putTarget(sourcePath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)
	targetPath := filepath.Join(target, "newObject1")

	for err := range doCopyCmd([]string{sourcePath}, targetPath, bar) {
		c.Assert(err, IsNil)
	}

	targetURL := server.URL + "/bucket/newObject"
	for err := range doCopyCmd([]string{sourcePath}, targetURL, bar) {
		c.Assert(err, IsNil)
	}
}

func (s *CmdTestSuite) TestCpTypeB(c *C) {
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	sourcePath := filepath.Join(source, "object1")
	data := "hello"
	dataLen := len(data)
	err = putTarget(sourcePath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	for err := range doCopyCmd([]string{sourcePath}, target, bar) {
		c.Assert(err, IsNil)
	}

	targetURL := server.URL + "/bucket"
	for err := range doCopyCmd([]string{sourcePath}, targetURL, bar) {
		c.Assert(err, IsNil)
	}
}

func (s *CmdTestSuite) TestCpTypeC(c *C) {
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(source, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	for err := range doCopyCmd([]string{source + "..."}, target, bar) {
		c.Assert(err, IsNil)
	}

	targetURL := server.URL + "/bucket"
	for err := range doCopyCmd([]string{source + "..."}, targetURL, bar) {
		c.Assert(err, IsNil)
	}
}

func (s *CmdTestSuite) TestCpTypeD(c *C) {
	/// filesystem
	source1, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	source2, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source1)
	defer os.RemoveAll(source2)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(source1, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	for i := 10; i < 20; i++ {
		objectPath := filepath.Join(source2, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	for err := range doCopyCmd([]string{source1 + "...", source2 + "..."}, target, bar) {
		c.Assert(err, IsNil)
	}

	targetURL := server.URL + "/bucket"
	for err := range doCopyCmd([]string{source1 + "...", source2 + "..."}, targetURL, bar) {
		c.Assert(err, IsNil)
	}
}
