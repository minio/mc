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

	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestCopyURLType(c *C) {
	// Valid Types.
	sourceURLs := []string{server.URL + "/bucket/object1"}
	targetURL := server.URL + "/bucket/test"
	isRecursive := false
	c.Assert(guessCopyURLType(sourceURLs, targetURL, isRecursive), Equals, copyURLsTypeA)

	sourceURLs = []string{server.URL + "/bucket/object1"}
	targetURL = server.URL + "/bucket"
	isRecursive = false
	c.Assert(guessCopyURLType(sourceURLs, targetURL, isRecursive), Equals, copyURLsTypeB)

	sourceURLs = []string{server.URL + "/bucket/"}
	targetURL = server.URL + "/bucket"
	isRecursive = true
	c.Assert(guessCopyURLType(sourceURLs, targetURL, isRecursive), Equals, copyURLsTypeC)

	sourceURLs = []string{server.URL + "/bucket/test1.txt", server.URL + "/bucket/test2.txt"}
	targetURL = server.URL + "/bucket"
	isRecursive = false
	c.Assert(guessCopyURLType(sourceURLs, targetURL, isRecursive), Equals, copyURLsTypeD)

	//   * INVALID RULES
	//   =========================
	//   copy(d, f)
	//   copy(d..., f)
	//   copy([]f, f)

	sourceURLs = []string{"/test/"}
	targetURL = server.URL + "/bucket/test.txt"
	isRecursive = false
	c.Assert(guessCopyURLType(sourceURLs, targetURL, isRecursive), Equals, copyURLsTypeInvalid)

	sourceURLs = []string{"/test/"}
	targetURL = server.URL + "/bucket/test.txt"
	isRecursive = true
	c.Assert(guessCopyURLType(sourceURLs, targetURL, isRecursive), Equals, copyURLsTypeInvalid)

	sourceURLs = []string{"/test/test1.txt", "/test/test2.txt"}
	targetURL = server.URL + "/bucket/test.txt"
	isRecursive = false
	c.Assert(guessCopyURLType(sourceURLs, targetURL, isRecursive), Equals, copyURLsTypeInvalid)

	sourceURLs = []string{server.URL + "/bucket/", server.URL + "/bucket/"}
	targetURL = ""
	isRecursive = false
	c.Assert(guessCopyURLType(sourceURLs, targetURL, isRecursive), Equals, copyURLsTypeInvalid)
}

func (s *TestSuite) TestMirror(c *C) {
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(source, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err := putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "mirror", source, target})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false
}

func (s *TestSuite) TestCopy(c *C) {
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(source, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err := putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	// reset back
	console.IsExited = false

	target := filepath.Join(source, "random")
	defer os.RemoveAll(target)

	err = app.Run([]string{os.Args[0], "cp", "--recursive", source, target})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsError = false
	console.IsExited = false
}
