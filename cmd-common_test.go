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
	"time"

	"github.com/cheggaaa/pb"

	"errors"
	"net"

	. "github.com/minio-io/check"
	"github.com/minio-io/minio/pkg/iodine"
)

type CommonSuite struct{}

type testAddr struct{}

func (ta *testAddr) Network() string {
	return ta.String()
}
func (ta *testAddr) Error() string {
	return ta.String()
}
func (ta *testAddr) String() string {
	return "testAddr"
}

var _ = Suite(&CommonSuite{})

func (s *CommonSuite) TestStatusBar(c *C) {
	bar := startBar(1024)
	c.Assert(bar, Not(IsNil))
	c.Assert(bar.Units, Equals, pb.U_BYTES)
	c.Assert(bar.RefreshRate, Equals, time.Millisecond*10)
	c.Assert(bar.NotPrint, Equals, true)
	c.Assert(bar.ShowSpeed, Equals, true)
}

func (s *CommonSuite) TestIsValidRetry(c *C) {
	opError := &net.OpError{
		Op:   "read",
		Net:  "net",
		Addr: &testAddr{},
		Err:  errors.New("Op Error"),
	}
	c.Assert(isValidRetry(nil), Equals, false)
	c.Assert(isValidRetry(errors.New("hello")), Equals, false)
	c.Assert(isValidRetry(iodine.New(errors.New("hello"), nil)), Equals, false)
	c.Assert(isValidRetry(&net.DNSError{}), Equals, true)
	c.Assert(isValidRetry(iodine.New(&net.DNSError{}, nil)), Equals, true)
	// op error read
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error write
	opError.Op = "write"
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error dial
	opError.Op = "dial"
	c.Assert(isValidRetry(opError), Equals, true)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, true)
	// op error foo
	opError.Op = "foo"
	c.Assert(isValidRetry(opError), Equals, false)
	c.Assert(isValidRetry(iodine.New(opError, nil)), Equals, false)
}
