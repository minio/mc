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
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	. "github.com/minio/check"
)

func (s *CmdTestSuite) TestLSNonRecursive(c *C) {
	configDir, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(configDir)
	switch runtime.GOOS {
	case "windows":
		mcConfigWindowsDir = configDir
	default:
		mcConfigDir = configDir
	}
	_, err = doConfig("generate", nil)
	c.Assert(err, IsNil)

	/// filesystem
	root, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(root, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	clnt, err := getNewClient(root, &hostConfig{}, false)
	c.Assert(err, IsNil)
	err = doList(clnt, root, false)
	c.Assert(err, IsNil)
}

func (s *CmdTestSuite) TestLSRecursive(c *C) {
	configDir, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(configDir)
	switch runtime.GOOS {
	case "windows":
		mcConfigWindowsDir = configDir
	default:
		mcConfigDir = configDir
	}
	_, err = doConfig("generate", nil)
	c.Assert(err, IsNil)

	/// filesystem
	root, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(root, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, &hostConfig{}, uint64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	clnt, err := getNewClient(root, &hostConfig{}, true)
	c.Assert(err, IsNil)
	err = doList(clnt, root, false)
	c.Assert(err, IsNil)
}
