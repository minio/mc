/*
 * MinIO Client (C) 2015 MinIO, Inc.
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

package cmd

import (
	"os"
	"regexp"

	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestValidSessionID(c *C) {
	validSid := regexp.MustCompile("^[a-zA-Z]+$")
	sid := newRandomID(8)
	c.Assert(len(sid), Equals, 8)
	c.Assert(validSid.MatchString(sid), Equals, true)
}

func (s *TestSuite) TestSession(c *C) {
	err := createSessionDir()
	c.Assert(err, IsNil)
	c.Assert(isSessionDirExists(), Equals, true)

	session := newSessionV8(getHash("cp", []string{"mybucket", "myminio/mybucket"}))
	c.Assert(session.Header.CommandArgs, IsNil)
	c.Assert(len(session.SessionID) >= 8, Equals, true)
	_, e := os.Stat(session.DataFP.Name())
	c.Assert(e, IsNil)

	err = session.Close()
	c.Assert(err, IsNil)
	c.Assert(isSessionExists(session.SessionID), Equals, true)

	savedSession, err := loadSessionV8(session.SessionID)
	c.Assert(err, IsNil)
	c.Assert(session.SessionID, Equals, savedSession.SessionID)

	err = savedSession.Close()
	c.Assert(err, IsNil)

	err = savedSession.Delete()
	c.Assert(err, IsNil)
	c.Assert(isSessionExists(session.SessionID), Equals, false)
	_, e = os.Stat(session.DataFP.Name())
	c.Assert(e, NotNil)
}
