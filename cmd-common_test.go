/*
 * Mini Copy, (C) 2014, 2015 Minio, Inc.
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
	"time"

	"github.com/cheggaaa/pb"

	. "github.com/minio-io/check"
)

type StatusBarSuite struct{}

var _ = Suite(&StatusBarSuite{})

func (s *StatusBarSuite) TestStatusBar(c *C) {
	bar := startBar(1024)
	c.Assert(bar, Not(IsNil))
	c.Assert(bar.Units, Equals, pb.U_BYTES)
	c.Assert(bar.RefreshRate, Equals, time.Millisecond*10)
	c.Assert(bar.NotPrint, Equals, true)
	c.Assert(bar.ShowSpeed, Equals, true)
}
