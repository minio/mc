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

var barCp barSend

func (s *CmdTestSuite) TestCpTypeA(c *C) {
	/// filesystem
	source, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(source)

	cps, err := newSession()
	c.Assert(err, IsNil)

	sourcePath := filepath.Join(source, "object1")
	data := "hello"
	dataLen := len(data)
	err = putTarget(sourcePath, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)
	targetPath := filepath.Join(target, "newObject1")

	cps.URLs = append(cps.URLs, sourcePath)
	cps.URLs = append(cps.URLs, targetPath)
	for err := range doCopyCmdSession(barCp, cps) {
		c.Assert(err, IsNil)
	}

	cps, err = newSession()
	c.Assert(err, IsNil)
	targetURL := server.URL + "/bucket/newObject"

	cps.URLs = append(cps.URLs, sourcePath)
	cps.URLs = append(cps.URLs, targetURL)
	for err := range doCopyCmdSession(barCp, cps) {
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
	err = putTarget(sourcePath, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	cps, err := newSession()
	c.Assert(err, IsNil)

	cps.URLs = append(cps.URLs, sourcePath)
	cps.URLs = append(cps.URLs, target)
	for err := range doCopyCmdSession(barCp, cps) {
		c.Assert(err, IsNil)
	}

	cps, err = newSession()
	c.Assert(err, IsNil)

	targetURL := server.URL + "/bucket"
	cps.URLs = append(cps.URLs, sourcePath)
	cps.URLs = append(cps.URLs, targetURL)
	for err := range doCopyCmdSession(barCp, cps) {
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
		err = putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	cps, err := newSession()
	c.Assert(err, IsNil)

	cps.URLs = append(cps.URLs, source+"...")
	cps.URLs = append(cps.URLs, target)
	for err := range doCopyCmdSession(barCp, cps) {
		c.Assert(err, IsNil)
	}

	cps, err = newSession()
	c.Assert(err, IsNil)

	cps.URLs = append(cps.URLs, source+"...")
	cps.URLs = append(cps.URLs, server.URL+"/bucket")
	for err := range doCopyCmdSession(barCp, cps) {
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
		err = putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	for i := 10; i < 20; i++ {
		objectPath := filepath.Join(source2, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	target, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(target)

	cps, err := newSession()
	c.Assert(err, IsNil)
	cps.URLs = append(cps.URLs, source1+"...")
	cps.URLs = append(cps.URLs, source2+"...")
	cps.URLs = append(cps.URLs, target)
	for err := range doCopyCmdSession(barCp, cps) {
		c.Assert(err, IsNil)
	}

	cps, err = newSession()
	c.Assert(err, IsNil)
	targetURL := server.URL + "/bucket"
	cps.URLs = append(cps.URLs, source1+"...")
	cps.URLs = append(cps.URLs, source2+"...")
	cps.URLs = append(cps.URLs, targetURL)
	for err := range doCopyCmdSession(barCp, cps) {
		c.Assert(err, IsNil)
	}
}
*/
