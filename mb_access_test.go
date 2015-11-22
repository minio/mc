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

func (s *TestSuite) TestMbAndAccess(c *C) {
	// Instantiate client for URL.
	clnt, err := url2Client(server.URL + "/bucket")
	c.Assert(err, IsNil)

	// Make bucket.
	err = clnt.MakeBucket()
	c.Assert(err, IsNil)

	err = doSetAccess(server.URL+"/bucket", "public-read-write")
	c.Assert(err, IsNil)

	err = doSetAccess(server.URL+"/bucket", "invalid")
	c.Assert(err, Not(IsNil))

	perm, err := doGetAccess(server.URL + "/bucket")
	c.Assert(err, IsNil)
	c.Assert(perm.isPrivate(), Equals, true)
}
