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

func (s *CmdTestSuite) TestCopyURLType(c *C) {
	sourceURLs := []string{server.URL + "/bucket/object1"}
	targetURL := server.URL + "/bucket/test"
	c.Assert(guessCopyURLType(sourceURLs, targetURL), Equals, copyURLsTypeA)

	sourceURLs = []string{server.URL + "/bucket/object1"}
	targetURL = server.URL + "/bucket"
	c.Assert(guessCopyURLType(sourceURLs, targetURL), Equals, copyURLsTypeB)

	sourceURLs = []string{server.URL + "/bucket/..."}
	targetURL = server.URL + "/bucket"
	c.Assert(guessCopyURLType(sourceURLs, targetURL), Equals, copyURLsTypeC)

	sourceURLs = []string{server.URL + "/bucket/...", server.URL + "/bucket/..."}
	targetURL = server.URL + "/bucket/test"
	c.Assert(guessCopyURLType(sourceURLs, targetURL), Equals, copyURLsTypeD)
}

func (s *CmdTestSuite) TestCastURLType(c *C) {
	sourceURL := server.URL + "/bucket"
	targetURLs := []string{}
	c.Assert(guessCastURLType(sourceURL, targetURLs), Equals, castURLsTypeInvalid)

	sourceURL = server.URL + "/bucket"
	targetURLs = nil
	c.Assert(guessCastURLType(sourceURL, targetURLs), Equals, castURLsTypeInvalid)

	sourceURL = server.URL + "/bucket..."
	targetURLs = []string{server.URL + "/bucket"}
	c.Assert(guessCastURLType(sourceURL, targetURLs), Equals, castURLsTypeC)

	sourceURL = server.URL + "/bucket/object1"
	targetURLs = []string{server.URL + "/bucket"}
	c.Assert(guessCastURLType(sourceURL, targetURLs), Equals, castURLsTypeB)

	sourceURL = server.URL + "/bucket/object1"
	targetURLs = []string{server.URL + "/bucket/object_new"}
	c.Assert(guessCastURLType(sourceURL, targetURLs), Equals, castURLsTypeA)
}
