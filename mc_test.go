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
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	. "gopkg.in/check.v1"
)

var customConfigDir string

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

var server *httptest.Server
var app *cli.App

func (s *TestSuite) SetUpSuite(c *C) {
	objectAPI := objectAPIHandler(objectAPIHandler{lock: &sync.Mutex{}, bucket: "bucket", object: make(map[string][]byte)})
	server = httptest.NewServer(objectAPI)
	console.IsTesting = true

	// do not set it elsewhere, leads to data races since this is a global flag
	globalQuiet = true // quiet is set to turn of progress bar

	tmpDir, e := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(e, IsNil)

	// For windows the path is slightly different.
	if runtime.GOOS == "windows" {
		customConfigDir = filepath.Join(tmpDir, globalMCConfigWindowsDir)
	} else {
		customConfigDir = filepath.Join(tmpDir, globalMCConfigDir)
	}
	setMcConfigDir(customConfigDir)

	err := createMcConfigDir()
	c.Assert(err, IsNil)

	config := newMcConfig()
	config.Hosts[server.URL] = hostConfig{
		AccessKeyID:     "WLGDGYAQYIGI833EV05A",
		SecretAccessKey: "BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF",
		API:             "S3v4",
	}

	err = saveMcConfig(config)
	c.Assert(err, IsNil)

	err = createSessionDir()
	c.Assert(err, IsNil)

	app = registerApp()
}

func (s *TestSuite) TearDownSuite(c *C) {
	os.RemoveAll(customConfigDir)
	if server != nil {
		server.Close()
	}
}

func (s *TestSuite) TestGetNewClient(c *C) {
	_, err := getNewClient("http://example.com/bucket1", hostConfig{})
	c.Assert(err, IsNil)
	_, err = getNewClient("https://example.com/bucket1", hostConfig{})
	c.Assert(err, IsNil)
	_, err = getNewClient("C:\\Users\\Administrator\\MyDocuments", hostConfig{})
	c.Assert(err, IsNil)
	_, err = getNewClient("/usr/bin/pandoc", hostConfig{})
	c.Assert(err, IsNil)
	_, err = getNewClient("pkg/client", hostConfig{})
	c.Assert(err, IsNil)
}

// setMcConfigPath - set a custom minio client config path.
func setMcConfigPath(configPath string) {
	mcCustomConfigPath = configPath
}

func (s *TestSuite) TestNewConfigV6(c *C) {
	root, e := ioutil.TempDir(os.TempDir(), "mc-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	conf := newMcConfig()
	configFile := filepath.Join(root, globalMCConfigFile)
	setMcConfigPath(configFile)

	err := saveMcConfig(conf)
	c.Assert(err, IsNil)

	data, err := loadMcConfig()
	c.Assert(err, IsNil)

	setMcConfigPath("")

	type aliases struct {
		name string
		url  string
	}

	wantAliases := []aliases{
		{
			"s3",
			"https://s3.amazonaws.com",
		},
		{
			"play",
			"https://play.minio.io:9000",
		},
		{
			"local",
			"http://localhost:9000",
		},
	}
	for _, alias := range wantAliases {
		url, ok := data.Aliases[alias.name]
		c.Assert(ok, Equals, true)
		c.Assert(url, Equals, alias.url)
	}

	wantHosts := []string{
		"http://localhost:9000",
		"https://play.minio.io:9000",
		"https://dl.minio.io:9000",
		"https://s3.amazonaws.com",
		"https://storage.googleapis.com",
	}
	for _, host := range wantHosts {
		_, ok := data.Hosts[host]
		c.Assert(ok, Equals, true)
	}
}

func (s *TestSuite) TestValidPERMS(c *C) {
	perms := accessPerms("private")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(perms.String(), Equals, "private")
	perms = accessPerms("public")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(perms.String(), Equals, "public-read-write")
	perms = accessPerms("readonly")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(perms.String(), Equals, "public-read")
	perms = accessPerms("authorized")
	c.Assert(perms.isValidAccessPERM(), Equals, true)
	c.Assert(perms.String(), Equals, "authenticated-read")
}

func (s *TestSuite) TestInvalidPERMS(c *C) {
	perms := accessPerms("invalid")
	c.Assert(perms.isValidAccessPERM(), Equals, false)
}

func (s *TestSuite) TestGetMcConfigDir(c *C) {
	dir, err := getMcConfigDir()
	c.Assert(err, IsNil)
	switch runtime.GOOS {
	case "linux":
		fallthrough
	case "freebsd":
		fallthrough
	case "darwin":
		c.Assert(dir, Equals, customConfigDir)
	case "windows":
		c.Assert(dir, Equals, customConfigDir)
	default:
		c.Fail()
	}
	c.Assert(mustGetMcConfigDir(), Equals, dir)
}

func (s *TestSuite) TestGetMcConfigPath(c *C) {
	dir, err := getMcConfigPath()
	c.Assert(err, IsNil)
	switch runtime.GOOS {
	case "linux":
		fallthrough
	case "freebsd":
		fallthrough
	case "darwin":
		c.Assert(dir, Equals, filepath.Join(mustGetMcConfigDir(), "config.json"))
	case "windows":
		c.Assert(dir, Equals, filepath.Join(mustGetMcConfigDir(), "config.json"))
	default:
		c.Fail()
	}
	c.Assert(mustGetMcConfigPath(), Equals, dir)
}

func (s *TestSuite) TestIsvalidAliasName(c *C) {
	c.Check(isValidAliasName("helloWorld0"), Equals, true)
	c.Check(isValidAliasName("h0SFD2k24Fdsa"), Equals, true)
	c.Check(isValidAliasName("fdslka-4"), Equals, true)
	c.Check(isValidAliasName("fdslka-"), Equals, true)
	c.Check(isValidAliasName("helloWorld$"), Equals, false)
	c.Check(isValidAliasName("h0SFD2k2#Fdsa"), Equals, false)
	c.Check(isValidAliasName("0dslka-4"), Equals, false)
	c.Check(isValidAliasName("-fdslka"), Equals, false)
}

func (s *TestSuite) TestEmptyExpansions(c *C) {
	url, err := getAliasURL("hello")
	c.Assert(err, IsNil)
	c.Assert(url, Equals, "hello")

	url, err = getAliasURL("minio://hello")
	c.Assert(err, IsNil)
	c.Assert(url, Equals, "minio://hello")

	url, err = getAliasURL("$#\\")
	c.Assert(err, IsNil)
	c.Assert(url, Equals, "$#\\")

	url, err = getAliasURL("myfoo/bar")
	c.Assert(err, IsNil)
	c.Assert(url, Equals, "myfoo/bar")

	url, err = getAliasURL("")
	c.Assert(err, IsNil)
	c.Assert(url, Equals, "")
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

func (s *TestSuite) TestVersionContext(c *C) {
	console.IsExited = false

	err := app.Run([]string{os.Args[0], "version"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false
}

func (s *TestSuite) TestCommonPrefix(c *C) {
	c.Assert(commonPrefix("/usr", "/usr/local"), Equals, "/usr")
	c.Assert(commonPrefix("/uabbf", "/ursfad/ccc"), Equals, "/u")
	c.Assert(commonPrefix("/usr/local/lib", "/usr/local/test"), Equals, "/usr/local/")
}

func (s *TestSuite) TestVersions(c *C) {
	v1 := newVersion("1.5.1")
	v2 := newVersion("1.5.0")
	c.Assert(v2.LessThan(v1), Equals, true)
	c.Assert(v1.LessThan(v2), Equals, false)
}

func (s *TestSuite) TestApp(c *C) {
	err := app.Run([]string{""})
	c.Assert(err, IsNil)
}
