// Copyright (c) 2015-2021 MinIO, Inc.
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
