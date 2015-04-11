package main

import (
	. "github.com/minio-io/check"
	"os/user"
	"path"
	"runtime"
)

func (s *MySuite) TestGetMcConfigDir(c *C) {
	u, err := user.Current()
	c.Assert(err, IsNil)
	dir, err := getMcConfigDir()
	c.Assert(err, IsNil)
	if runtime.GOOS == "linux" {
		c.Assert(dir, Equals, path.Join(u.HomeDir, ".mc/"))
	} else if runtime.GOOS == "windows" {
		c.Assert(dir, Equals, path.Join(u.HomeDir, "mc/"))
	} else {
		c.Fail()
	}
	c.Assert(mustGetMcConfigDir(), Equals, dir)
}

func (s *MySuite) TestGetMcConfigPath(c *C) {
	dir, err := getMcConfigPath()
	c.Assert(err, IsNil)
	if runtime.GOOS == "linux" {
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	} else if runtime.GOOS == "windows" {
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	} else {
		c.Fail()
	}
	c.Assert(mustGetMcConfigPath(), Equals, dir)
}
