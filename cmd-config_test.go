/*
 * Mini Copy (C) 2014, 2015 Minio, Inc.
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
	"os/user"
	"path"
	"runtime"

	. "github.com/minio-io/check"
)

func mustGetMcConfigDir() string {
	dir, _ := getMcConfigDir()
	return dir
}

func (s *MySuite) TestGetMcConfigDir(c *C) {
	u, err := user.Current()
	c.Assert(err, IsNil)
	dir, err := getMcConfigDir()
	c.Assert(err, IsNil)
	if runtime.GOOS == "linux" {
		c.Assert(dir, Equals, path.Join(u.HomeDir, ".mc/"))
	} else if runtime.GOOS == "windows" {
		c.Assert(dir, Equals, path.Join(u.HomeDir, "mc/"))
	} else {
		c.Fail()
	}
	c.Assert(mustGetMcConfigDir(), Equals, dir)
}

func (s *MySuite) TestGetMcConfigPath(c *C) {
	dir, err := getMcConfigPath()
	c.Assert(err, IsNil)
	if runtime.GOOS == "linux" {
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	} else if runtime.GOOS == "windows" {
		c.Assert(dir, Equals, path.Join(mustGetMcConfigDir(), "config.json"))
	} else {
		c.Fail()
	}
	c.Assert(mustGetMcConfigPath(), Equals, dir)
}
