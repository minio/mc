/*
 * MinIO Client (C) 2014, 2015 MinIO, Inc.
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

package cmd

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

func (s *TestSuite) SetUpSuite(c *C) {
}

func (s *TestSuite) TearDownSuite(c *C) {
}

func (s *TestSuite) TestValidPERMS(c *C) {
	perms := accessPerms("none")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(string(perms), Equals, "none")
	perms = accessPerms("public")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(string(perms), Equals, "public")
	perms = accessPerms("download")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(string(perms), Equals, "download")
	perms = accessPerms("upload")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(string(perms), Equals, "upload")
}

func (s *TestSuite) TestInvalidPERMS(c *C) {
	perms := accessPerms("invalid")
	c.Assert(perms.isValidAccessPERM(), Equals, false)
}

func (s *TestSuite) TestGetMcConfigDir(c *C) {
	dir, err := getMcConfigDir()
	c.Assert(err, IsNil)
	c.Assert(dir, Not(Equals), "")
	c.Assert(mustGetMcConfigDir(), Equals, dir)
}

func (s *TestSuite) TestGetMcConfigPath(c *C) {
	dir, err := getMcConfigPath()
	c.Assert(err, IsNil)
	switch runtime.GOOS {
	case "linux", "freebsd", "darwin", "solaris":
		c.Assert(dir, Equals, filepath.Join(mustGetMcConfigDir(), "config.json"))
	case "windows":
		c.Assert(dir, Equals, filepath.Join(mustGetMcConfigDir(), "config.json"))
	default:
		c.Fail()
	}
	c.Assert(mustGetMcConfigPath(), Equals, dir)
}

func (s *TestSuite) TestIsvalidAliasName(c *C) {
	c.Check(isValidAlias("helloWorld0"), Equals, true)
	c.Check(isValidAlias("hello_World0"), Equals, true)
	c.Check(isValidAlias("h0SFD2k24Fdsa"), Equals, true)
	c.Check(isValidAlias("fdslka-4"), Equals, true)
	c.Check(isValidAlias("fdslka-"), Equals, true)
	c.Check(isValidAlias("helloWorld$"), Equals, false)
	c.Check(isValidAlias("h0SFD2k2#Fdsa"), Equals, false)
	c.Check(isValidAlias("0dslka-4"), Equals, false)
	c.Check(isValidAlias("-fdslka"), Equals, false)
}

func (s *TestSuite) TestHumanizedTime(c *C) {
	hTime := timeDurationToHumanizedDuration(time.Duration(10) * time.Second)
	c.Assert(hTime.Minutes, Equals, int64(0))
	c.Assert(hTime.Hours, Equals, int64(0))
	c.Assert(hTime.Days, Equals, int64(0))

	hTime = timeDurationToHumanizedDuration(time.Duration(10) * time.Minute)
	c.Assert(hTime.Hours, Equals, int64(0))
	c.Assert(hTime.Days, Equals, int64(0))

	hTime = timeDurationToHumanizedDuration(time.Duration(10) * time.Hour)
	c.Assert(hTime.Days, Equals, int64(0))

	hTime = timeDurationToHumanizedDuration(time.Duration(24) * time.Hour)
	c.Assert(hTime.Days, Not(Equals), int64(0))
}
