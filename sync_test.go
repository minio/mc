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

/*
import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	. "github.com/minio/check"
)

var barSync barSend

func (s *CmdTestSuite) TestSyncTypeA(c *C) {
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	sourcePath := filepath.Join(source, "object1")
	data := "hello"
	dataLen := len(data)
	err = putTarget(sourcePath, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)
	targetPath := filepath.Join(target, "newObject1")

	ss, err := newSession()
	c.Assert(err, IsNil)
	ss.URLs = append(ss.URLs, sourcePath)
	ss.URLs = append(ss.URLs, targetPath)
	for err := range doSyncCmdSession(barSync, ss) {
		c.Assert(err, IsNil)
	}

	ss, err = newSession()
	c.Assert(err, IsNil)
	targetURL := server.URL + "/bucket/newObject"
	ss.URLs = append(ss.URLs, sourcePath)
	ss.URLs = append(ss.URLs, targetURL)
	for err := range doSyncCmdSession(barSync, ss) {
		c.Assert(err, IsNil)
	}
}

func (s *CmdTestSuite) TestSyncTypeB(c *C) {
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	sourcePath := filepath.Join(source, "object1")
	data := "hello"
	dataLen := len(data)
	err = putTarget(sourcePath, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	ss, err := newSession()
	c.Assert(err, IsNil)
	ss.URLs = append(ss.URLs, sourcePath)
	ss.URLs = append(ss.URLs, target)
	for err := range doSyncCmdSession(barSync, ss) {
		c.Assert(err, IsNil)
	}

	ss, err = newSession()
	c.Assert(err, IsNil)
	targetURL := server.URL + "/bucket"
	ss.URLs = append(ss.URLs, sourcePath)
	ss.URLs = append(ss.URLs, targetURL)
	for err := range doSyncCmdSession(barSync, ss) {
		c.Assert(err, IsNil)
	}
}

func (s *CmdTestSuite) TestSyncTypeC(c *C) {
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(source, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	ss, err := newSession()
	c.Assert(err, IsNil)
	ss.URLs = append(ss.URLs, source+"...")
	ss.URLs = append(ss.URLs, target)
	for err := range doSyncCmdSession(barSync, ss) {
		c.Assert(err, IsNil)
	}

	targetURL := server.URL + "/bucket"
	ss, err = newSession()
	c.Assert(err, IsNil)
	ss.URLs = append(ss.URLs, source+"...")
	ss.URLs = append(ss.URLs, targetURL)
	for err := range doSyncCmdSession(barSync, ss) {
		c.Assert(err, IsNil)
	}

	target1, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target1)

	target2, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target2)

	ss, err = newSession()
	c.Assert(err, IsNil)
	ss.URLs = append(ss.URLs, source+"...")
	ss.URLs = append(ss.URLs, target1)
	ss.URLs = append(ss.URLs, target2)
	for err := range doSyncCmdSession(barSync, ss) {
		c.Assert(err, IsNil)
	}
}
*/
