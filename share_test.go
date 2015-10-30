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

func (s *TestSuite) TestShareFailure(c *C) {
	objectURL := server.URL + "/bucket/object1"

	// invalid duration format ``1hr``
	err := app.Run([]string{os.Args[0], "share", "download", objectURL, "1hr"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false

	// too high duration 169h, maximum is 168h i.e 7days.
	err = app.Run([]string{os.Args[0], "share", "download", objectURL, "169hr"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false

	// too low duration 0s, minimum required is 1s.
	err = app.Run([]string{os.Args[0], "share", "download", objectURL, "0s"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false
}

func (s *TestSuite) TestShareSuccess(c *C) {
	objectURL := server.URL + "/bucket/object1"

	err := app.Run([]string{os.Args[0], "share", "download", objectURL})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "share", "download", objectURL, "1h"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false
}
