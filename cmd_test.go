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
	"os/user"
	"path"
	"runtime"
	"testing"
	"time"

	"errors"
	"net"

	"github.com/cheggaaa/pb"
	. "github.com/minio-io/check"
	"github.com/minio-io/minio/pkg/iodine"
)

func Test(t *testing.T) { TestingT(t) }

type CmdTestSuite struct{}

var _ = Suite(&CmdTestSuite{})

func mustGetMcConfigDir() string {
	dir, _ := getMcConfigDir()
	return dir
}

func (s *CmdTestSuite) TestGetMcConfigDir(c *C) {
	u, err := user.Current()
	c.Assert(err, IsNil)
	dir, err := getMcConfigDir()
	c.Assert(err, IsNil)
	switch runtime.GOOS {
	case "linux":
		c.Assert(dir, Equals, path.Join(u.HomeDir, ".mc/"))
	case "windows":
		c.Assert(dir, Equals, path.Join(u.HomeDir, "mc/"))
	case "darwin":
		c.Assert(dir, Equals, path.Join(u.HomeDir, ".mc/"))
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

func (s *CmdTestSuite) TestStatusBar(c *C) {
	bar := startBar(1024)
	c.Assert(bar, Not(IsNil))
	c.Assert(bar.Units, Equals, pb.U_BYTES)
	c.Assert(bar.RefreshRate, Equals, time.Millisecond*10)
	c.Assert(bar.NotPrint, Equals, true)
	c.Assert(bar.ShowSpeed, Equals, true)
}

func (s *CmdTestSuite) TestIsValidRetry(c *C) {
	opError := &net.OpError{
		Op:   "read",
		Net:  "net",
		Addr: &testAddr{},
		Err:  errors.New("Op Error"),
	}
	c.Assert(isValidRetry(nil), Equals, false)
	c.Assert(isValidRetry(errors.New("hello")), Equals, false)
	c.Assert(isValidRetry(iodine.New(errors.New("hello"), nil)), Equals, false)
	c.Assert(isValidRetry(&net.DNSError{}), Equals, true)
	c.Assert(isValidRetry(iodine.New(&net.DNSError{}, nil)), Equals, true)
	// op error read
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error write
	opError.Op = "write"
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error dial
	opError.Op = "dial"
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error foo
	opError.Op = "foo"
	c.Assert(isValidRetry(opError), Equals, false)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, false)
}
