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
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"net/http/httptest"

	. "github.com/minio/check"
	"github.com/minio/mc/pkg/quick"
)

func Test(t *testing.T) { TestingT(t) }

type CmdTestSuite struct{}

var _ = Suite(&CmdTestSuite{})

func mustGetMcConfigDir() string {
	dir, _ := getMcConfigDir()
	return dir
}

var server *httptest.Server

func (s *CmdTestSuite) SetUpSuite(c *C) {
	configDir, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	customConfigDir = configDir

	_, err = doConfig("generate", nil)
	c.Assert(err, IsNil)

	objectAPI := objectAPIHandler(objectAPIHandler{bucket: "bucket", object: make(map[string][]byte)})
	server = httptest.NewServer(objectAPI)
}

func (s *CmdTestSuite) TearDownSuite(c *C) {
	os.RemoveAll(customConfigDir)
	server.Close()
}

func (s *CmdTestSuite) TestGetNewClient(c *C) {
	_, err := getNewClient("http://example.com/bucket1", &hostConfig{})
	c.Assert(err, IsNil)
	_, err = getNewClient("C:\\Users\\Administrator\\MyDocuments", &hostConfig{})
	c.Assert(err, IsNil)
	_, err = getNewClient("/usr/bin/pandoc", &hostConfig{})
	c.Assert(err, IsNil)
	_, err = getNewClient("pkg/client", &hostConfig{})
	c.Assert(err, IsNil)
}

func (s *CmdTestSuite) TestNewConfigV1(c *C) {
	root, err := ioutil.TempDir(os.TempDir(), "mc-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	conf, err := newConfig()
	c.Assert(err, IsNil)
	configFile := path.Join(root, "config.json")
	err = conf.Save(configFile)
	c.Assert(err, IsNil)

	confNew := newConfigV1()
	config, err := quick.New(confNew)
	c.Assert(err, IsNil)
	err = config.Load(configFile)
	c.Assert(err, IsNil)
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

func (s *CmdTestSuite) TestValidACL(c *C) {
	acl := bucketACL("private")
	c.Assert(acl.isValidBucketACL(), Equals, true)
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
		c.Assert(dir, Equals, path.Join(customConfigDir, ".mc/"))
	case "windows":
		c.Assert(dir, Equals, path.Join(customConfigDir, "mc/"))
	case "darwin":
		c.Assert(dir, Equals, path.Join(customConfigDir, ".mc/"))
	case "freebsd":
		c.Assert(dir, Equals, path.Join(customConfigDir, ".mc/"))
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
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	case "windows":
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	case "darwin":
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	case "freebsd":
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
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
	c.Check(isValidAliasName("help"), Equals, false)
	c.Check(isValidAliasName("private"), Equals, false) // reserved names
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

type testAddr struct{}

func (ta *testAddr) Network() string {
	return ta.String()
}
func (ta *testAddr) Error() string {
	return ta.String()
}
func (ta *testAddr) String() string {
	return "testAddr"
}

func (s *CmdTestSuite) TestCommonMethods(c *C) {
	/// filesystem
	root, err := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(root)

	objectPath := filepath.Join(root, "object1")
	data := "hello"
	dataLen := len(data)

	err = putTarget(objectPath, &hostConfig{}, int64(dataLen), bytes.NewReader([]byte(data)))
	c.Assert(err, IsNil)

	reader, size, err := getSource(objectPath, &hostConfig{})
	c.Assert(err, IsNil)
	c.Assert(size, Not(Equals), 0)
	var results bytes.Buffer
	_, err = io.CopyN(&results, reader, int64(size))
	c.Assert(err, IsNil)
	c.Assert([]byte(data), DeepEquals, results.Bytes())

	_, content, err := url2Stat(objectPath)
	c.Assert(err, IsNil)
	c.Assert(content.Name, Equals, filepath.Join(root, "object1"))
	c.Assert(content.Type.IsRegular(), Equals, true)

	_, _, err = url2Stat(objectPath + "invalid")
	c.Assert(err, Not(IsNil))

}
