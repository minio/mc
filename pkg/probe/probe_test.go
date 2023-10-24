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

package probe_test

import (
	"os"
	"testing"

	"github.com/minio/mc/pkg/probe"
	check "gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type MySuite struct{}

var _ = check.Suite(&MySuite{})

func testDummy0() *probe.Error {
	_, e := os.Stat("this-file-cannot-exit")
	return probe.NewError(e)
}

func testDummy1() *probe.Error {
	return testDummy0().Trace("DummyTag1")
}

func testDummy2() *probe.Error {
	return testDummy1().Trace("DummyTag2")
}

func (s *MySuite) TestProbe(c *check.C) {
	probe.Init() // Set project's root source path.
	probe.SetAppInfo("Commit-ID", "7390cc957239")
	es := testDummy2().Trace("TopOfStack")
	// Uncomment the following Println to visually test probe call trace.
	// fmt.Println("Expecting a simulated error here.", es)
	c.Assert(es, check.Not(check.Equals), nil)

	newES := es.Trace()
	c.Assert(newES, check.Not(check.Equals), nil)
}

func (s *MySuite) TestWrappedError(c *check.C) {
	_, e := os.Stat("this-file-cannot-exit")
	es := probe.NewError(e) // *probe.Error
	e = probe.WrapError(es) // *probe.WrappedError
	_, ok := probe.UnwrapError(e)
	c.Assert(ok, check.Equals, true)
}
