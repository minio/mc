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

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestDiffObjects(c *C) {
	/// filesystem
	root1, e := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root1)

	root2, e := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root2)

	objectPath1 := filepath.Join(root1, "object1")
	data := "hello"
	dataLen := len(data)
	err := putTarget(objectPath1, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	objectPath2 := filepath.Join(root2, "object1")
	data = "hello"
	dataLen = len(data)
	err = putTarget(objectPath2, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	for diff := range doDiffMain(objectPath1, objectPath2, false) {
		c.Assert(diff.Error, IsNil)
	}
}

func (s *TestSuite) TestDiffDirs(c *C) {
	/// filesystem
	root1, e := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(e, IsNil)
	subDir1 := filepath.Join(root1, "subDir")
	e = os.Mkdir(subDir1, 0755)
	c.Assert(e, IsNil)
	defer os.RemoveAll(root1)

	root2, e := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(e, IsNil)
	subDir2 := filepath.Join(root2, "subDir")
	e = os.Mkdir(subDir2, 0755)
	defer os.RemoveAll(root2)

	var err *probe.Error
	for i := 0; i < 10; i++ {
		objectPath1 := filepath.Join(root1, "object"+strconv.Itoa(i))
		objectPath2 := filepath.Join(subDir1, "object"+strconv.Itoa(i))

		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath1, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
		err = putTarget(objectPath2, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	for i := 0; i < 10; i++ {
		objectPath1 := filepath.Join(root2, "object"+strconv.Itoa(i))
		objectPath2 := filepath.Join(subDir2, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath1, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
		err = putTarget(objectPath2, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	// non-recursive
	for diff := range doDiffMain(root1, root2, false) {
		c.Assert(diff.Error, IsNil)
	}
	// recursive
	for diff := range doDiffMain(root1, root2, true) {
		c.Assert(diff.Error, IsNil)
	}
}

func (s *TestSuite) TestDiffContext(c *C) {
	err := app.Run([]string{os.Args[0], "diff", server.URL + "/bucket", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "diff", server.URL + "/bucket...", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "diff", server.URL + "/invalid", server.URL + "/invalid..."})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false
}
