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
	"os"

	"github.com/minio/mc/pkg/console"
	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestMbAndAccess(c *C) {
	perr := doMakeBucket(server.URL + "/bucket")
	c.Assert(perr, IsNil)

	perr = doSetAccess(server.URL+"/bucket", "public-read-write")
	c.Assert(perr, IsNil)

	perr = doSetAccess(server.URL+"/bucket", "invalid")
	c.Assert(perr, Not(IsNil))

	perm, perr := doGetAccess(server.URL + "/bucket")
	c.Assert(perr, IsNil)
	c.Assert(perm.isPrivate(), Equals, true)
}

func (s *TestSuite) TestMBContext(c *C) {
	console.IsExited = false

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

func (s *TestSuite) TestAccessContext(c *C) {
	console.IsExited = false

	err := app.Run([]string{os.Args[0], "access", "set", "private", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "set", "public", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "set", "readonly", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "set", "authorized", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "set", "invalid", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)
	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "get", server.URL + "/bucket"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)
	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "access", "get", server.URL + "/invalid"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)
	// reset back
	console.IsExited = false
}
