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
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/minio/check"
)

func (s *CmdTestSuite) TestMbAndAccessCmd(c *C) {
	/// filesystem
	root, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	_, err = doMakeBucketCmd(filepath.Join(root, "bucket"), &hostConfig{}, false)
	c.Assert(err, IsNil)

	_, err = doUpdateAccessCmd(filepath.Join(root, "bucket"), "public-read-write", &hostConfig{}, false)
	c.Assert(err, IsNil)

	_, err = doUpdateAccessCmd(filepath.Join(root, "bucket"), "invalid", &hostConfig{}, false)
	c.Assert(err, Not(IsNil))

	_, err = doMakeBucketCmd(server.URL+"/bucket", &hostConfig{}, false)
	c.Assert(err, IsNil)

	_, err = doUpdateAccessCmd(server.URL+"/bucket", "public-read-write", &hostConfig{}, false)
	c.Assert(err, IsNil)

	_, err = doUpdateAccessCmd(server.URL+"/bucket", "invalid", &hostConfig{}, false)
	c.Assert(err, Not(IsNil))

}
