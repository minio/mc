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
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/hashicorp/go-version"
	"github.com/minio/cli"
	. "gopkg.in/check.v1"
)

var customConfigDir string

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

var server *httptest.Server
var app *cli.App

func (s *TestSuite) SetUpSuite(c *C) {
}

func (s *TestSuite) TearDownSuite(c *C) {
}

func (s *TestSuite) TestValidPERMS(c *C) {
	perms := accessPerms("none")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(string(perms), Equals, "none")
	perms = accessPerms("readwrite")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(string(perms), Equals, "readwrite")
	perms = accessPerms("readonly")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(string(perms), Equals, "readonly")
	perms = accessPerms("writeonly")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(string(perms), Equals, "writeonly")
}

// Tests valid and invalid secret keys.
func (s *TestSuite) TestValidSecretKeys(c *C) {
	c.Assert(isValidSecretKey("password"), Equals, true)
	c.Assert(isValidSecretKey("BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF"), Equals, true)

	c.Assert(isValidSecretKey("aaa"), Equals, false)
	c.Assert(isValidSecretKey("password%%"), Equals, false)
}

// Tests valid and invalid access keys.
func (s *TestSuite) TestValidAccessKeys(c *C) {
	c.Assert(isValidAccessKey("c67W2-r4MAyAYScRl"), Equals, true)
	c.Assert(isValidAccessKey("EXOb76bfeb1234562iu679f11588"), Equals, true)
	c.Assert(isValidAccessKey("BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF"), Equals, true)
	c.Assert(isValidAccessKey("admin"), Equals, true)

	c.Assert(isValidAccessKey("aaa"), Equals, false)
	c.Assert(isValidAccessKey("$$%%%%%3333"), Equals, false)
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
	c.Check(isValidAlias("h0SFD2k24Fdsa"), Equals, true)
	c.Check(isValidAlias("fdslka-4"), Equals, true)
	c.Check(isValidAlias("fdslka-"), Equals, true)
	c.Check(isValidAlias("helloWorld$"), Equals, false)
	c.Check(isValidAlias("h0SFD2k2#Fdsa"), Equals, false)
	c.Check(isValidAlias("0dslka-4"), Equals, false)
	c.Check(isValidAlias("-fdslka"), Equals, false)
}

func (s *TestSuite) TestHumanizedTime(c *C) {
	hTime := timeDurationToHumanizedTime(time.Duration(10) * time.Second)
	c.Assert(hTime.Minutes, Equals, int64(0))
	c.Assert(hTime.Hours, Equals, int64(0))
	c.Assert(hTime.Days, Equals, int64(0))

	hTime = timeDurationToHumanizedTime(time.Duration(10) * time.Minute)
	c.Assert(hTime.Hours, Equals, int64(0))
	c.Assert(hTime.Days, Equals, int64(0))

	hTime = timeDurationToHumanizedTime(time.Duration(10) * time.Hour)
	c.Assert(hTime.Days, Equals, int64(0))

	hTime = timeDurationToHumanizedTime(time.Duration(24) * time.Hour)
	c.Assert(hTime.Days, Not(Equals), int64(0))
}

func (s *TestSuite) TestCommonPrefix(c *C) {
	c.Assert(commonPrefix("/usr", "/usr/local"), Equals, "/usr")
	c.Assert(commonPrefix("/uabbf", "/ursfad/ccc"), Equals, "/u")
	c.Assert(commonPrefix("/usr/local/lib", "/usr/local/test"), Equals, "/usr/local/")
}

func (s *TestSuite) TestVersions(c *C) {
	v1, e := version.NewVersion("1.6")
	c.Assert(e, IsNil)
	v2, e := version.NewConstraint(">= 1.5.0")
	c.Assert(e, IsNil)
	c.Assert(v2.Check(v1), Equals, true)
}
