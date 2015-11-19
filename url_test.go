/*
 * Minio Client (C) 2015 Minio, Inc.
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

import . "gopkg.in/check.v1"

func (s *TestSuite) TestURLJoinPath(c *C) {
	// Join two URLs
	url1 := "http://s3.mycompany.io/dev"
	url2 := "http://s3.aws.amazon.com/mybucket/bin/zgrep"
	url := urlJoinPath(url1, url2)
	c.Assert(url, Equals, "http://s3.mycompany.io/dev/mybucket/bin/zgrep")

	// Join URL and a path
	url1 = "http://s3.mycompany.io/dev"
	url2 = "mybucket/bin/zgrep"
	url = urlJoinPath(url1, url2)
	c.Assert(url, Equals, "http://s3.mycompany.io/dev/mybucket/bin/zgrep")

	// Check if it strips URL2's tailing ‘/’
	url1 = "http://s3.mycompany.io/dev"
	url2 = "mybucket/bin/"
	url = urlJoinPath(url1, url2)
	c.Assert(url, Equals, "http://s3.mycompany.io/dev/mybucket/bin/")
}

func (s *TestSuite) TestURL2DirContent(c *C) {
	_, err := url2DirContent(".")
	c.Assert(err, IsNil)
}
