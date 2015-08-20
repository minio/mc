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
	"os"
	"regexp"

	. "github.com/minio/mc/internal/gopkg.in/check.v1"
	"github.com/minio/mc/pkg/console"
)

func (s *CmdTestSuite) TestValidSessionID(c *C) {
	validSid := regexp.MustCompile("^[a-zA-Z]+$")
	sid := newSID(8)
	c.Assert(len(sid), Equals, 8)
	c.Assert(validSid.MatchString(sid), Equals, true)
}

func (s *CmdTestSuite) TestSession(c *C) {
	perr := createSessionDir()
	c.Assert(perr, IsNil)
	c.Assert(isSessionDirExists(), Equals, true)

	session := newSessionV2()
	c.Assert(session.Header.CommandArgs, IsNil)
	c.Assert(len(session.SessionID), Equals, 8)

	perr = session.Close()
	c.Assert(perr, IsNil)

	savedSession, perr := loadSessionV2(session.SessionID)
	c.Assert(perr, IsNil)
	c.Assert(session.SessionID, Equals, savedSession.SessionID)

	perr = savedSession.Close()
	c.Assert(perr, IsNil)

	perr = savedSession.Delete()
	c.Assert(perr, IsNil)
}

func (s *CmdTestSuite) TestSessionContext(c *C) {
	err := app.Run([]string{os.Args[0], "session", "list"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "session", "clear", "all"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "session", "resume", "invalid"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "session", "clear", "invalid"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false

	perr := createSessionDir()
	c.Assert(perr, IsNil)
	c.Assert(isSessionDirExists(), Equals, true)

	// reset back
	console.IsExited = false

	session := newSessionV2()
	c.Assert(session.Header.CommandArgs, IsNil)
	c.Assert(len(session.SessionID), Equals, 8)

	perr = session.Save()
	c.Assert(perr, IsNil)

	err = app.Run([]string{os.Args[0], "session", "clear", session.SessionID})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)

	// reset back
	console.IsExited = false
}
