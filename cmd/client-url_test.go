// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"testing"

	checkv1 "gopkg.in/check.v1"
)

// TestURL - tests url parsing and fields.
func (s *TestSuite) TestURL(c *checkv1.C) {
	urlStr := "foo?.go"
	url := newClientURL(urlStr)
	c.Assert(url.Path, checkv1.Equals, "foo?.go")

	urlStr = "https://s3.amazonaws.com/mybucket/foo?.go"
	url = newClientURL(urlStr)
	c.Assert(url.Scheme, checkv1.Equals, "https")
	c.Assert(url.Host, checkv1.Equals, "s3.amazonaws.com")
	c.Assert(url.Path, checkv1.Equals, "/mybucket/foo?.go")
}

// TestURLJoinPath - tests joining two different urls.
func (s *TestSuite) TestURLJoinPath(c *checkv1.C) {
	// Join two URLs
	url1 := "http://s3.mycompany.io/dev"
	url2 := "http://s3.aws.amazon.com/mybucket/bin/zgrep"
	url := urlJoinPath(url1, url2)
	c.Assert(url, checkv1.Equals, "http://s3.mycompany.io/dev/mybucket/bin/zgrep")

	// Join URL and a path
	url1 = "http://s3.mycompany.io/dev"
	url2 = "mybucket/bin/zgrep"
	url = urlJoinPath(url1, url2)
	c.Assert(url, checkv1.Equals, "http://s3.mycompany.io/dev/mybucket/bin/zgrep")

	// Check if it strips URL2's tailing `/`
	url1 = "http://s3.mycompany.io/dev"
	url2 = "mybucket/bin/"
	url = urlJoinPath(url1, url2)
	c.Assert(url, checkv1.Equals, "http://s3.mycompany.io/dev/mybucket/bin/")
}

func Test_isURLPrefix(t *testing.T) {
	type args struct {
		src  string
		dest string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"test1", args{"s3/test", "s3/test/test"}, true},
		{"test2", args{"s3/test/", "s3/test/test"}, true},
		{"test3", args{"s3/test/test", "s3/test/"}, true},
		{"test4", args{"s3/test/test", "s3/test/test.123"}, false},
		{"test5", args{"s3/test/", "s3/test/test/test/test"}, true},
		{"test6", args{"s3/test/*", "s3/test/test/"}, true},
		{"test7", args{"s3/test/*", "s3/test1/test/"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isURLPrefix(tt.args.src, tt.args.dest); got != tt.want {
				t.Errorf("isURLPrefix() = %v, want %v", got, tt.want)
			}
			if got := isURLPrefix(tt.args.dest, tt.args.src); got != tt.want {
				t.Errorf("isURLPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}
