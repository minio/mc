/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

func (s *TestSuite) TestConfigVersionContext(c *C) {
	console.IsExited = false
	err := app.Run([]string{os.Args[0], "config", "version"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)
	console.IsExited = false
}

func (s *TestSuite) TestConfigAliasContext(c *C) {
	console.IsExited = false

	err := app.Run([]string{os.Args[0], "config", "alias", "add", "test", "htt://test.io"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "config", "alias", "add", "new", "http://test.io"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false
}

func (s *TestSuite) TestConfigHostContext(c *C) {
	console.IsExited = false

	err := app.Run([]string{os.Args[0],
		"config",
		"host",
		"add",
		"*test.io",
		"AKIKJAA5BMMU2RHO6IBB",
		"V7f1CCwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr9",
	})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0],
		"config",
		"host",
		"add",
		"http://my-example.com",
		"AKIKJAA5BMMU2RHO6IBB",
		"V7f1CCwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr9",
		"S3v2",
	})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "config", "host", "remove", "http://my-example.com"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "config", "host", "remove", "http://dl.minio.io:9000"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false
}
