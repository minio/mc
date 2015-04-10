package main

import (
	. "github.com/minio-io/check"
	"log"
	"testing"
)

type MySuite struct{}

func Test(t *testing.T) { TestingT(t) }

var _ = Suite(&MySuite{})

func (s *MySuite) TestIsvalidAliasName(c *C) {
	c.Check(isValidAliasName("helloWorld0"), Equals, true)
	c.Check(isValidAliasName("h0SFD2k24Fdsa"), Equals, true)
	c.Check(isValidAliasName("fdslka-4"), Equals, true)
	c.Check(isValidAliasName("fdslka-"), Equals, true)
	c.Check(isValidAliasName("helloWorld$"), Equals, false)
	c.Check(isValidAliasName("h0SFD2k2#Fdsa"), Equals, false)
	c.Check(isValidAliasName("0dslka-4"), Equals, false)
	c.Check(isValidAliasName("-fdslka"), Equals, false)
}

func (s *MySuite) TestInvalidUrlInAliasExpand(c *C) {
	c.Skip("Test still being written")
	invalidUrl := "foohello"
	url, err := aliasExpand(invalidUrl, nil)
	c.Assert(err, Not(IsNil))
	log.Println(url)
	log.Println(err)
}
