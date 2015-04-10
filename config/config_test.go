/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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

package config

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/minio-io/check"
)

type MySuite struct{}

var _ = Suite(&MySuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *MySuite) TestConfig(c *C) {
	configPath, _ := ioutil.TempDir("/tmp", "minio-test-")
	ctx, err := New(configPath, "config.json")
	c.Assert(err, IsNil)
	defer os.RemoveAll(configPath)

	ctx.Config = new(Config)

	err = ctx.Config.AddAuth("http*://s3*.amazonaws.com", "YOUR-ACCESS-KEY-ID-HERE", "YOUR-SECRET-ACCESS-KEY-HERE")
	c.Assert(err, Not(IsNil))

	err = ctx.Config.AddAuth("http*://s3*.amazonaws.com", "AC5NH40NQLTL4D2W92PM", "0nAMx5oJbWx5IgCmOJJneXM8w/ohTz2b0QAb2xvN")
	c.Assert(err, IsNil)

	err = ctx.Config.AddAlias("s3", "https://s3.amazonaws.com")
	c.Assert(err, IsNil)

	err = ctx.SaveConfig()
	c.Assert(err, IsNil)

	err = ctx.LoadConfig()
	c.Assert(err, IsNil)
}
