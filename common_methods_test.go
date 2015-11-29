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

func (s *TestSuite) TestCommonMethods(c *C) {
	/// filesystem
	root, e := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object1")

	c.Assert(isTargetURLDir(root), Equals, true)
	c.Assert(isTargetURLDir(objectPath), Equals, false)

	objectPathServer := server.URL + "/bucket/object1"
	data := "hello"
	err := putTarget(objectPath, bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)
	err = putTarget(objectPathServer, bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	c.Assert(isTargetURLDir(objectPathServer), Equals, false)
	c.Assert(isTargetURLDir(server.URL+"/bucket"), Equals, true)

	reader, err := getSource(objectPathServer)
	c.Assert(err, IsNil)
	var results bytes.Buffer
	_, e = io.Copy(&results, reader)
	c.Assert(e, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

	_, content, err := url2Stat(objectPathServer)
	c.Assert(err, IsNil)
	c.Assert(content.Type.IsRegular(), Equals, true)

	_, _, err = url2Stat(objectPath + "invalid")
	c.Assert(err, Not(IsNil))

	_, err = url2Client(objectPathServer)
	c.Assert(err, IsNil)

	_, err = url2Client(objectPathServer)
	c.Assert(err, IsNil)
}
