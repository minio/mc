package main

import (
	"github.com/cheggaaa/pb"
	. "github.com/minio-io/check"
	"strings"
	"time"
)

var _ = Suite(&MySuite{})

func (s *MySuite) TestStatusBar(c *C) {
	bar := startBar(1024)
	c.Assert(bar, Not(IsNil))
	c.Assert(bar.Units, Equals, pb.U_BYTES)
	c.Assert(bar.RefreshRate, Equals, time.Millisecond*10)
	c.Assert(bar.NotPrint, Equals, true)
	c.Assert(bar.ShowSpeed, Equals, true)
}

func (s *MySuite) TestBashCompletionFilename(c *C) {
	file, err := getMcBashCompletionFilename()
	c.Assert(err, IsNil)
	c.Assert(file, Not(Equals), "mc.bash_completion")
	c.Assert(strings.HasSuffix(file, "mc.bash_completion"), Equals, true)

	file2 := mustGetMcBashCompletionFilename()
	c.Assert(file, Equals, file2)
}
