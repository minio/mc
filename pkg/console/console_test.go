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

package console

import (
	"runtime"
	"testing"

	. "github.com/minio-io/check"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestSetTheme(c *C) {
	err := SetTheme("nocolor")
	c.Assert(err, IsNil)
	c.Assert(GetThemeName(), Equals, "nocolor")
	err = SetTheme("unknown")
	c.Assert(err, Not(IsNil))
	c.Assert(GetThemeName(), Equals, "nocolor")
}

func (s *MySuite) TestDefaultTheme(c *C) {
	if runtime.GOOS == "windows" {
		c.Assert(GetDefaultThemeName(), Equals, "nocolor")
	} else {
		c.Assert(GetDefaultThemeName(), Equals, "minimal")
	}
}
