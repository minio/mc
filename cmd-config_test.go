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
}
