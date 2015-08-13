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

	"github.com/minio/mc/internal/github.com/minio/cli"
	. "github.com/minio/mc/internal/gopkg.in/check.v1"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/quick"
)

var customConfigDir string

func Test(t *testing.T) { TestingT(t) }

type CmdTestSuite struct{}

var _ = Suite(&CmdTestSuite{})

var server *httptest.Server
var app *cli.App

func (s *CmdTestSuite) SetUpSuite(c *C) {
	objectAPI := objectAPIHandler(objectAPIHandler{lock: &sync.Mutex{}, bucket: "bucket", object: make(map[string][]byte)})
	server = httptest.NewServer(objectAPI)
	console.IsTesting = true

	// do not set it elsewhere, leads to data races since this is a global flag
	globalQuietFlag = true

	tmpDir, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)

	// For windows the path is slightly different.
	if runtime.GOOS == "windows" {
		customConfigDir = filepath.Join(tmpDir, mcConfigWindowsDir)
	} else {
		customConfigDir = filepath.Join(tmpDir, mcConfigDir)
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

	perr = doConfig("invalid", nil)
	c.Assert(perr, Not(IsNil))

	perr = doConfig("alias", []string{"test", "https://test.io"})
	c.Assert(perr, IsNil)

	perr = doConfig("alias", []string{"test", "https://test.io"})
	c.Assert(perr, Not(IsNil))

	perr = doConfig("alias", []string{"test", "htt://test.io"})
	c.Assert(perr, Not(IsNil))

	perr = doConfig("alias", []string{"readonly", "https://new.test.io"})
	c.Assert(perr, Not(IsNil))

	app = registerApp()
}

func (s *CmdTestSuite) TearDownSuite(c *C) {
	os.RemoveAll(customConfigDir)
	if server != nil {
		server.Close()
	}
}

func (s *CmdTestSuite) TestGetNewClient(c *C) {
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

func (s *CmdTestSuite) TestNewConfigV1(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "mc-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	conf, perr := newConfig()
	c.Assert(perr, IsNil)
	configFile := filepath.Join(root, "config.json")
	perr = conf.Save(configFile)
	c.Assert(perr, IsNil)

	confNew := newConfigV101()
	config, perr := quick.New(confNew)
	c.Assert(perr, IsNil)
	perr = config.Load(configFile)
	c.Assert(perr, IsNil)
	data := config.Data().(*configV1)

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

func (s *CmdTestSuite) TestRecursiveURL(c *C) {
	c.Assert(isURLRecursive("url..."), Equals, true)
	c.Assert(isURLRecursive("url.."), Equals, false)
	c.Assert(stripRecursiveURL("url..."), Equals, "url")
	c.Assert(stripRecursiveURL("url.."), Equals, "url..")
	c.Assert(stripRecursiveURL("..."), Equals, ".")
	c.Assert(stripRecursiveURL("...url"), Equals, "...url")
}

func (s *CmdTestSuite) TestHostConfig(c *C) {
	hostcfg, err := getHostConfig("https://s3.amazonaws.com")
	c.Assert(err, IsNil)
	c.Assert(hostcfg.AccessKeyID, Equals, globalAccessKeyID)
	c.Assert(hostcfg.SecretAccessKey, Equals, globalSecretAccessKey)

	_, err = getHostConfig("http://test.minio.io")
	c.Assert(err, Not(IsNil))
}

func (s *CmdTestSuite) TestArgs2URL(c *C) {
	URLs := []string{"localhost:", "s3:", "play:", "https://s3-us-west-2.amazonaws.com"}
	expandedURLs, err := args2URLs(URLs)
	c.Assert(err, IsNil)
	c.Assert(expandedURLs[0], Equals, "http://localhost:9000/")
	c.Assert(expandedURLs[1], Equals, "https://s3.amazonaws.com/")
	c.Assert(expandedURLs[2], Equals, "https://play.minio.io:9000/")
	c.Assert(expandedURLs[3], Equals, "https://s3-us-west-2.amazonaws.com")
}

func (s *CmdTestSuite) TestValidACL(c *C) {
	acl := bucketACL("private")
	c.Assert(acl.isValidBucketACL(), Equals, true)
	c.Assert(acl.String(), Equals, "private")
	acl = bucketACL("public")
	c.Assert(acl.isValidBucketACL(), Equals, true)
	c.Assert(acl.String(), Equals, "public-read-write")
	acl = bucketACL("readonly")
	c.Assert(acl.isValidBucketACL(), Equals, true)
	c.Assert(acl.String(), Equals, "public-read")
	acl = bucketACL("authenticated")
	c.Assert(acl.isValidBucketACL(), Equals, true)
	c.Assert(acl.String(), Equals, "authenticated-read")
}

func (s *CmdTestSuite) TestInvalidACL(c *C) {
	acl := bucketACL("invalid")
	c.Assert(acl.isValidBucketACL(), Equals, false)
}

func (s *CmdTestSuite) TestGetMcConfigDir(c *C) {
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

func (s *CmdTestSuite) TestGetMcConfigPath(c *C) {
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

func (s *CmdTestSuite) TestIsvalidAliasName(c *C) {
	c.Check(isValidAliasName("helloWorld0"), Equals, true)
	c.Check(isValidAliasName("h0SFD2k24Fdsa"), Equals, true)
	c.Check(isValidAliasName("fdslka-4"), Equals, true)
	c.Check(isValidAliasName("fdslka-"), Equals, true)
	c.Check(isValidAliasName("helloWorld$"), Equals, false)
	c.Check(isValidAliasName("h0SFD2k2#Fdsa"), Equals, false)
	c.Check(isValidAliasName("0dslka-4"), Equals, false)
	c.Check(isValidAliasName("-fdslka"), Equals, false)
	c.Check(isAliasReserved("help"), Equals, true)
	c.Check(isAliasReserved("private"), Equals, true) // reserved names
}

func (s *CmdTestSuite) TestEmptyExpansions(c *C) {
	url, err := aliasExpand("hello", nil)
	c.Assert(url, Equals, "hello")
	c.Assert(err, IsNil)

	url, err = aliasExpand("minio://hello", nil)
	c.Assert(url, Equals, "minio://hello")
	c.Assert(err, IsNil)

	url, err = aliasExpand("$#\\", nil)
	c.Assert(url, Equals, "$#\\")
	c.Assert(err, IsNil)

	url, err = aliasExpand("foo:bar", map[string]string{"foo": "http://foo"})
	c.Assert(url, Equals, "http://foo/bar")
	c.Assert(err, IsNil)

	url, err = aliasExpand("myfoo:bar", nil)
	c.Assert(url, Equals, "myfoo:bar")
	c.Assert(err, IsNil)

	url, err = aliasExpand("", nil)
	c.Assert(url, Equals, "")
	c.Assert(err, IsNil)

	url, err = aliasExpand("hello", nil)
	c.Assert(url, Equals, "hello")
	c.Assert(err, IsNil)
}

func (s *CmdTestSuite) TestApp(c *C) {
	err := app.Run([]string{""})
	c.Assert(err, IsNil)
}
