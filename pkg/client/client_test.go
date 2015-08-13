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

package client

import (
	"testing"

	. "github.com/minio/mc/internal/gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestURLParse(c *C) {
	u, err := Parse("http://s3.example.com")
	c.Assert(err, IsNil)
	c.Assert(u.Scheme, Equals, "http")
	c.Assert(u.Host, Equals, "s3.example.com")
	c.Assert(u.Path, Equals, "/")
	c.Assert(u.SchemeSeparator, Equals, "://")

	u, err = Parse("http://s3.example.com/path/new")
	c.Assert(err, IsNil)
	c.Assert(u.Scheme, Equals, "http")
	c.Assert(u.Host, Equals, "s3.example.com")
	c.Assert(u.Path, Equals, "/path/new")
	c.Assert(u.SchemeSeparator, Equals, "://")

	u, err = Parse(":::://s3.example.com/path/new")
	c.Assert(err, IsNil)
	c.Assert(u.Scheme, Equals, "")
	c.Assert(u.Host, Equals, "")
	c.Assert(u.Path, Equals, ":::://s3.example.com/path/new")
	c.Assert(u.SchemeSeparator, Equals, "")
}

func (s *MySuite) TestPathParse(c *C) {
	u, err := Parse("path/test")
	c.Assert(err, IsNil)
	c.Assert(u.Scheme, Equals, "")
	c.Assert(u.Host, Equals, "")
	c.Assert(u.Path, Equals, "path/test")
	c.Assert(u.SchemeSeparator, Equals, "")

	u, err = Parse("/path/test")
	c.Assert(err, IsNil)
	c.Assert(u.Scheme, Equals, "")
	c.Assert(u.Host, Equals, "")
	c.Assert(u.Path, Equals, "/path/test")
	c.Assert(u.SchemeSeparator, Equals, "")
}
