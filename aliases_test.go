package main

import (
	. "github.com/minio-io/check"
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

func (s *MySuite) TestEmptyExpansions(c *C) {
	//	c.Skip("Test still being written")
	url, err := aliasExpand("hello", nil)
	c.Assert(url, Equals, "hello")
	c.Assert(err, IsNil)

	url, err = aliasExpand("minio://hello", nil)
	c.Assert(url, Equals, "minio://hello")
	c.Assert(err, IsNil)

	url, err = aliasExpand("$#\\", nil)
	c.Assert(url, Equals, "$#\\")
	c.Assert(err, IsNil)

	url, err = aliasExpand("foo:bar", map[string]string{"foo": "http://foo/"})
	c.Assert(url, Equals, "http://foo/bar")
	c.Assert(err, IsNil)

	url, err = aliasExpand("myfoo:bar", map[string]string{"foo": "http://foo/"})
	c.Assert(url, Equals, "myfoo:bar")
	c.Assert(err, IsNil)

	url, err = aliasExpand("", map[string]string{"foo": "http://foo/"})
	c.Assert(url, Equals, "")
	c.Assert(err, IsNil)

	url, err = aliasExpand("hello", nil)
	c.Assert(url, Equals, "hello")
	c.Assert(err, IsNil)
}
