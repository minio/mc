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

package hookreader

import (
	"bytes"
	"testing"

	check "gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type MySuite struct{}

var _ = check.Suite(&MySuite{})

// customReader - implements custom progress reader.
type customReader struct {
	readBytes int
}

func (c *customReader) Read(b []byte) (n int, err error) {
	c.readBytes += len(b)
	return len(b), nil
}

// Tests hook reader implementation.
func (s *MySuite) TestHookReader(c *check.C) {
	var buffer bytes.Buffer
	writer := &buffer
	_, err := writer.Write([]byte("Hello"))
	c.Assert(err, check.IsNil)
	progress := &customReader{}
	reader := NewHook(&buffer, progress)
	b := make([]byte, 3)
	n, err := reader.Read(b)
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, 3)
	c.Assert(progress.readBytes, check.Equals, 3)
}
