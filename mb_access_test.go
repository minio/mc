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

	"github.com/minio/mc/pkg/console"
	. "gopkg.in/check.v1"
)

func (s *CmdTestSuite) TestMbAndAccessCmd(c *C) {
	/// filesystem
	root, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)
	{
		err := doMakeBucketCmd(filepath.Join(root, "bucket"))
		c.Assert(err, IsNil)

		err = doUpdateAccessCmd(filepath.Join(root, "bucket"), "public-read-write")
		c.Assert(err, IsNil)

		err = doUpdateAccessCmd(filepath.Join(root, "bucket"), "invalid")
		c.Assert(err, Not(IsNil))

		err = doMakeBucketCmd(server.URL + "/bucket")
		c.Assert(err, IsNil)

		err = doUpdateAccessCmd(server.URL+"/bucket", "public-read-write")
		c.Assert(err, IsNil)

		err = doUpdateAccessCmd(server.URL+"/bucket", "invalid")
		c.Assert(err, Not(IsNil))
	}

}

func (s *CmdTestSuite) TestMBContext(c *C) {
	err := app.Run([]string{os.Args[0], "mb", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "mb", server.URL + "/$.bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false
}

func (s *CmdTestSuite) TestAccessContext(c *C) {
	err := app.Run([]string{os.Args[0], "access", "private", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "public", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "readonly", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "authenticated", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "invalid", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)
	// reset back
	console.IsExited = false
}
