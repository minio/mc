/*
 * MinIO Cloud Storage, (C) 2016 MinIO, Inc.
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

package ioutils_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/minio/mc/pkg/ioutils"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestIoutils(c *C) {
	path, err := ioutil.TempDir(os.TempDir(), "minio-ioutils_test")
	c.Assert(err, IsNil)
	defer os.RemoveAll(path)

	status, err := ioutils.IsDirEmpty(path)
	c.Assert(err, IsNil)
	c.Assert(status, Equals, true)
}

// Test for ParseDurationTime. Validates the returned value
// for given time value in days, hours and minute format.
func TestParseDurationTime(t *testing.T) {

	testCases := []struct {
		timeValue string
		expected  time.Duration
		err       string
	}{
		// Test 1: empty string as input
		{"", 0, "time: invalid duration "},
		// Test 2: Input string contains 4 day, 10 hour and 3 minutes
		{"4d10h3m", 381780000000000, ""},
		// Test 3: Input string contains 10 day and  3 hours
		{"10d3h", 874800000000000, ""},
		// Test 4: Input string contains minutes and days
		{"3m7d", 604980000000000, ""},
		// Test 5: Input string contains unknown unit
		{"4a3d", 0, "time: unknown unit a in duration 4a3d"},
		// Test 6: Input string contains fractional day
		{"1.5d", 129600000000000, ""},
		// Test 6: Input string contains fractional day and hour
		{"2.5d1.5h", 221400000000000, ""},
		// Test 7: Input string contains fractional day , hour and minute
		{"2.5d1.5h3.5m", 221610000000000, ""},
	}

	for _, testCase := range testCases {
		myVal, err := ioutils.ParseDurationTime(testCase.timeValue)
		if err != nil && err.Error() != testCase.err {
			t.Error()
		}
		if myVal != testCase.expected {
			t.Error()
		}
	}
}
