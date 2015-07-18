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

import (
	"regexp"

	. "gopkg.in/check.v1"
)

func (s *CmdTestSuite) TestValidSessionID(c *C) {
	validSid := regexp.MustCompile("^[a-zA-Z]+$")
	sid := newSID(8)
	c.Assert(len(sid), Equals, 8)
	c.Assert(validSid.MatchString(sid), Equals, true)
}

func (s *CmdTestSuite) TestSession(c *C) {
	err := createSessionDir()
	c.Assert(err, IsNil)
	c.Assert(isSessionDirExists(), Equals, true)

	session := newSessionV2()
	c.Assert(session.Header.CommandArgs, IsNil)
	c.Assert(len(session.SessionID), Equals, 8)

	err = session.Save()
	c.Assert(err, IsNil)

	savedSession, err := loadSessionV2(session.SessionID)
	c.Assert(err, IsNil)
	c.Assert(session.SessionID, Equals, savedSession.SessionID)

	err = session.Close()
	c.Assert(err, IsNil)
}
