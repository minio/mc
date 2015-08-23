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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

func (s *CmdTestSuite) TestCommonMethods(c *C) {
	/// filesystem
	root, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object1")

	c.Assert(isTargetURLDir(root), Equals, true)
	c.Assert(isTargetURLDir(objectPath), Equals, false)

	objectPathServer := server.URL + "/bucket/object1"
	data := "hello"
	dataLen := len(data)
	perr := putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(perr, IsNil)
	perr = putTarget(objectPathServer, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(perr, IsNil)

	c.Assert(isTargetURLDir(objectPathServer), Equals, false)
	c.Assert(isTargetURLDir(server.URL+"/bucket"), Equals, true)

	reader, size, perr := getSource(objectPathServer)
	c.Assert(perr, IsNil)
	c.Assert(size, Not(Equals), 0)
	var results bytes.Buffer
	_, err = io.CopyN(&results, reader, int64(size))
	c.Assert(err, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

	_, content, perr := url2Stat(objectPathServer)
	c.Assert(perr, IsNil)
	c.Assert(content.Name, Equals, "object1")
	c.Assert(content.Type.IsRegular(), Equals, true)

	_, _, perr = getSource(objectPathServer + "invalid")
	c.Assert(perr, Not(IsNil))

	_, _, perr = url2Stat(objectPath + "invalid")
	c.Assert(perr, Not(IsNil))

	_, perr = source2Client(objectPathServer)
	c.Assert(perr, IsNil)

	_, perr = target2Client(objectPathServer)
	c.Assert(perr, IsNil)

	_, perr = source2Client("http://test.minio.io" + "/bucket/fail")
	c.Assert(perr, Not(IsNil))

	_, perr = target2Client("http://test.minio.io" + "/bucket/fail")
	c.Assert(perr, Not(IsNil))
}
