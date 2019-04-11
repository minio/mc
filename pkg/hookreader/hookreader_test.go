/*
 * MinIO Client (C) 2016 MinIO, Inc.
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

package hookreader

import (
	"bytes"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

// customReader - implements custom progress reader.
type customReader struct {
	readBytes int
}

func (c *customReader) Read(b []byte) (n int, err error) {
	c.readBytes += len(b)
	return len(b), nil
}

// Tests hook reader implementation.
func (s *MySuite) TestHookReader(c *C) {
	var buffer bytes.Buffer
	writer := &buffer
	_, err := writer.Write([]byte("Hello"))
	c.Assert(err, IsNil)
	progress := &customReader{}
	reader := NewHook(&buffer, progress)
	b := make([]byte, 3)
	n, err := reader.Read(b)
	c.Assert(err, IsNil)
	c.Assert(n, Equals, 3)
	c.Assert(progress.readBytes, Equals, 3)
}
