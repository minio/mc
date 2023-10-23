// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	checkv1 "gopkg.in/check.v1"
)

func Test(t *testing.T) { checkv1.TestingT(t) }

type TestSuite struct{}

var _ = checkv1.Suite(&TestSuite{})

func (s *TestSuite) SetUpSuite(_ *checkv1.C) {
}

func (s *TestSuite) TearDownSuite(_ *checkv1.C) {
}

func (s *TestSuite) TestValidPERMS(c *checkv1.C) {
	perms := accessPerms("none")
	c.Assert(perms.isValidAccessPERM(), checkv1.Equals, true)
	c.Assert(string(perms), checkv1.Equals, "none")
	perms = accessPerms("public")
	c.Assert(perms.isValidAccessPERM(), checkv1.Equals, true)
	c.Assert(string(perms), checkv1.Equals, "public")
	perms = accessPerms("private")
	c.Assert(perms.isValidAccessPERM(), checkv1.Equals, true)
	c.Assert(string(perms), checkv1.Equals, "private")
	perms = accessPerms("download")
	c.Assert(perms.isValidAccessPERM(), checkv1.Equals, true)
	c.Assert(string(perms), checkv1.Equals, "download")
	perms = accessPerms("upload")
	c.Assert(perms.isValidAccessPERM(), checkv1.Equals, true)
	c.Assert(string(perms), checkv1.Equals, "upload")
}

func (s *TestSuite) TestInvalidPERMS(c *checkv1.C) {
	perms := accessPerms("invalid")
	c.Assert(perms.isValidAccessPERM(), checkv1.Equals, false)
}

func (s *TestSuite) TestGetMcConfigDir(c *checkv1.C) {
	dir, err := getMcConfigDir()
	c.Assert(err, checkv1.IsNil)
	c.Assert(dir, checkv1.Not(checkv1.Equals), "")
	c.Assert(mustGetMcConfigDir(), checkv1.Equals, dir)
}

func (s *TestSuite) TestGetMcConfigPath(c *checkv1.C) {
	dir, err := getMcConfigPath()
	c.Assert(err, checkv1.IsNil)
	switch runtime.GOOS {
	case "linux", "freebsd", "darwin", "solaris":
		c.Assert(dir, checkv1.Equals, filepath.Join(mustGetMcConfigDir(), "config.json"))
	case "windows":
		c.Assert(dir, checkv1.Equals, filepath.Join(mustGetMcConfigDir(), "config.json"))
	default:
		c.Fail()
	}
	c.Assert(mustGetMcConfigPath(), checkv1.Equals, dir)
}

func (s *TestSuite) TestIsvalidAliasName(c *checkv1.C) {
	c.Check(isValidAlias("helloWorld0"), checkv1.Equals, true)
	c.Check(isValidAlias("hello_World0"), checkv1.Equals, true)
	c.Check(isValidAlias("h0SFD2k24Fdsa"), checkv1.Equals, true)
	c.Check(isValidAlias("fdslka-4"), checkv1.Equals, true)
	c.Check(isValidAlias("fdslka-"), checkv1.Equals, true)
	c.Check(isValidAlias("helloWorld$"), checkv1.Equals, false)
	c.Check(isValidAlias("h0SFD2k2#Fdsa"), checkv1.Equals, false)
	c.Check(isValidAlias("0dslka-4"), checkv1.Equals, false)
	c.Check(isValidAlias("-fdslka"), checkv1.Equals, false)
}

func (s *TestSuite) TestHumanizedTime(c *checkv1.C) {
	hTime := timeDurationToHumanizedDuration(time.Duration(10) * time.Second)
	c.Assert(hTime.Minutes, checkv1.Equals, int64(0))
	c.Assert(hTime.Hours, checkv1.Equals, int64(0))
	c.Assert(hTime.Days, checkv1.Equals, int64(0))

	hTime = timeDurationToHumanizedDuration(time.Duration(10) * time.Minute)
	c.Assert(hTime.Hours, checkv1.Equals, int64(0))
	c.Assert(hTime.Days, checkv1.Equals, int64(0))

	hTime = timeDurationToHumanizedDuration(time.Duration(10) * time.Hour)
	c.Assert(hTime.Days, checkv1.Equals, int64(0))

	hTime = timeDurationToHumanizedDuration(time.Duration(24) * time.Hour)
	c.Assert(hTime.Days, checkv1.Not(checkv1.Equals), int64(0))
}
