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

	"net/http/httptest"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/quick"
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
	globalQuietFlag = true // quiet is set to turn of progress bar

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)

	// For windows the path is slightly different.
	if runtime.GOOS == "windows" {
		customConfigDir = filepath.Join(tmpDir, globalMCConfigWindowsDir)
	} else {
		customConfigDir = filepath.Join(tmpDir, globalMCConfigDir)
	}
	setMcConfigDir(customConfigDir)

	perr := createMcConfigDir()
	c.Assert(perr, IsNil)

	config, perr := newConfig()
	c.Assert(perr, IsNil)

	perr = writeConfig(config)
	c.Assert(perr, IsNil)

	perr = createSessionDir()
	c.Assert(perr, IsNil)

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

func (s *TestSuite) TestNewConfigV4(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "mc-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	conf, perr := newConfig()
	c.Assert(perr, IsNil)
	configFile := filepath.Join(root, "config.json")
	perr = conf.Save(configFile)
	c.Assert(perr, IsNil)

	confNew := newConfigV5()
	config, perr := quick.New(confNew)
	c.Assert(perr, IsNil)
	perr = config.Load(configFile)
	c.Assert(perr, IsNil)
	data := config.Data().(*configV5)

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
			"localhost",
			"http://localhost:9000",
		},
	}
	for _, alias := range wantAliases {
		url, ok := data.Aliases[alias.name]
		c.Assert(ok, Equals, true)
		c.Assert(url, Equals, alias.url)
	}

	wantHosts := []string{
		"localhost:*",
		"127.0.0.1:*",
		"play.minio.io:9000",
		"dl.minio.io:9000",
		"s3*.amazonaws.com",
	}
	for _, host := range wantHosts {
		_, ok := data.Hosts[host]
		c.Assert(ok, Equals, true)
	}
}

func (s *TestSuite) TestRecursiveURL(c *C) {
	c.Assert(isURLRecursive("url..."), Equals, true)
	c.Assert(isURLRecursive("url.."), Equals, false)
	c.Assert(stripRecursiveURL("url..."), Equals, "url")
	c.Assert(stripRecursiveURL("url.."), Equals, "url..")
	c.Assert(stripRecursiveURL("..."), Equals, ".")
	c.Assert(stripRecursiveURL("...url"), Equals, "...url")
}

func (s *TestSuite) TestHostConfig(c *C) {
	hostcfg, err := getHostConfig("https://s3.amazonaws.com")
	c.Assert(err, IsNil)
	c.Assert(hostcfg.AccessKeyID, Equals, globalAccessKeyID)
	c.Assert(hostcfg.SecretAccessKey, Equals, globalSecretAccessKey)
	c.Assert(hostcfg.API, Equals, "S3v4")

	_, err = getHostConfig("http://test.minio.io")
	c.Assert(err, Not(IsNil))
}

func (s *TestSuite) TestArgs2URL(c *C) {
	URLs := []string{"localhost", "s3", "play", "playgo", "play.go", "https://s3-us-west-2.amazonaws.com"}
	expandedURLs, err := args2URLs(URLs)
	c.Assert(err, IsNil)
	c.Assert(expandedURLs[0], Equals, "http://localhost:9000")
	c.Assert(expandedURLs[1], Equals, "https://s3.amazonaws.com")
	c.Assert(expandedURLs[2], Equals, "https://play.minio.io:9000")
	c.Assert(expandedURLs[3], Equals, "playgo")  // Has no corresponding alias. So expect same value.
	c.Assert(expandedURLs[4], Equals, "play.go") // Has no corresponding alias. So expect same value.
	c.Assert(expandedURLs[5], Equals, "https://s3-us-west-2.amazonaws.com")
}

func (s *TestSuite) TestValidPERMS(c *C) {
	perms := bucketPerms("private")
	c.Assert(perms.isValidBucketPERM(), Equals, true)
	c.Assert(perms.String(), Equals, "private")
	perms = bucketPerms("public")
	c.Assert(perms.isValidBucketPERM(), Equals, true)
	c.Assert(perms.String(), Equals, "public-read-write")
	perms = bucketPerms("readonly")
	c.Assert(perms.isValidBucketPERM(), Equals, true)
	c.Assert(perms.String(), Equals, "public-read")
	perms = bucketPerms("authorized")
	c.Assert(perms.isValidBucketPERM(), Equals, true)
	c.Assert(perms.String(), Equals, "authenticated-read")
}

func (s *TestSuite) TestInvalidPERMS(c *C) {
	perms := bucketPerms("invalid")
	c.Assert(perms.isValidBucketPERM(), Equals, false)
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
	url := getAliasURL("hello", nil)
	c.Assert(url, Equals, "hello")

	url = getAliasURL("minio://hello", nil)
	c.Assert(url, Equals, "minio://hello")

	url = getAliasURL("$#\\", nil)
	c.Assert(url, Equals, "$#\\")

	url = getAliasURL("foo/bar", map[string]string{"foo": "http://foo"})
	c.Assert(url, Equals, "http://foo/bar")

	url = getAliasURL("myfoo/bar", nil)
	c.Assert(url, Equals, "myfoo/bar")

	url = getAliasURL("", nil)
	c.Assert(url, Equals, "")

	url = getAliasURL("hello", nil)
	c.Assert(url, Equals, "hello")
}

func (s *TestSuite) TestApp(c *C) {
	err := app.Run([]string{""})
	c.Assert(err, IsNil)
}
