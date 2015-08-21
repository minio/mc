/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

	. "github.com/minio/mc/internal/gopkg.in/check.v1"
	"github.com/minio/mc/pkg/console"
)

func (s *CmdTestSuite) TestConfigContext(c *C) {
	err := app.Run([]string{os.Args[0], "config", "alias", "test", "htt://test.io"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false

	err = app.Run([]string{os.Args[0], "config", "alias", "new", "http://test.io"})
	c.Assert(err, IsNil)
	c.Assert(console.IsExited, Equals, false)
}
