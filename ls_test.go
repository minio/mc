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

func (s *CmdTestSuite) TestLSCmd(c *C) {
	/// filesystem
	root, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(root, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err = putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	err = doListCmd(root, false)
	c.Assert(err, IsNil)

	err = doListCmd(root, true)
	c.Assert(err, IsNil)

	for i := 0; i < 10; i++ {
		objectPath := server.URL + "/bucket/object" + strconv.Itoa(i)
		data := "hello"
		dataLen := len(data)
		err := putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}
	err = doListCmd(server.URL+"/bucket", false)
	c.Assert(err, IsNil)

	err = doListCmd(server.URL+"/bucket", true)
	c.Assert(err, IsNil)

}
