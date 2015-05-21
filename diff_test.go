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

func (s *CmdTestSuite) TestDiffObjects(c *C) {
	/// filesystem
	root1, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root1)

	root2, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root2)

	objectPath1 := filepath.Join(root1, "object1")
	data := "hello"
	dataLen := len(data)
	err = putTarget(objectPath1, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	objectPath2 := filepath.Join(root2, "object1")
	data = "hello"
	dataLen = len(data)
	err = putTarget(objectPath2, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	for diff := range doDiffCmd(objectPath1, objectPath2, false) {
		c.Assert(diff.err, IsNil)
	}
}

func (s *CmdTestSuite) TestDiffDirs(c *C) {
	/// filesystem
	root1, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root1)

	root2, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root2)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(root1, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(root2, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}
	for diff := range doDiffCmd(root1, root2, false) {
		c.Assert(diff.err, IsNil)
	}
}
