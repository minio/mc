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
	"math/rand"
	"os"
	"regexp"

	checkv1 "gopkg.in/check.v1"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// newRandomID generates a random id of regular lower case and uppercase english characters.
func newRandomID(n int) string {
	sid := make([]rune, n)
	for i := range sid {
		sid[i] = letters[rand.Intn(len(letters))]
	}
	return string(sid)
}

func (s *TestSuite) TestValidSessionID(c *checkv1.C) {
	validSid := regexp.MustCompile("^[a-zA-Z]+$")
	sid := newRandomID(8)
	c.Assert(len(sid), checkv1.Equals, 8)
	c.Assert(validSid.MatchString(sid), checkv1.Equals, true)
}

func (s *TestSuite) TestSession(c *checkv1.C) {
	err := createSessionDir()
	c.Assert(err, checkv1.IsNil)
	c.Assert(isSessionDirExists(), checkv1.Equals, true)

	session := newSessionV8(getHash("cp", []string{"mybucket", "myminio/mybucket"}))
	c.Assert(session.Header.CommandArgs, checkv1.IsNil)
	c.Assert(len(session.SessionID) >= 8, checkv1.Equals, true)
	_, e := os.Stat(session.DataFP.Name())
	c.Assert(e, checkv1.IsNil)

	err = session.Close()
	c.Assert(err, checkv1.IsNil)
	c.Assert(isSessionExists(session.SessionID), checkv1.Equals, true)

	savedSession, err := loadSessionV8(session.SessionID)
	c.Assert(err, checkv1.IsNil)
	c.Assert(session.SessionID, checkv1.Equals, savedSession.SessionID)

	err = savedSession.Close()
	c.Assert(err, checkv1.IsNil)

	err = savedSession.Delete()
	c.Assert(err, checkv1.IsNil)
	c.Assert(isSessionExists(session.SessionID), checkv1.Equals, false)
	_, e = os.Stat(session.DataFP.Name())
	c.Assert(e, checkv1.NotNil)
}
