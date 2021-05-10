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

package cmd

import (
	"testing"

	"maze.io/x/duration"
)

// Test for ParseDurationTime. Validates the returned value
// for given time value in days, hours and minute format.
func TestParseDurationTime(t *testing.T) {
	testCases := []struct {
		timeValue string
		expected  duration.Duration
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
		myVal, err := duration.ParseDuration(testCase.timeValue)
		if err != nil && err.Error() != testCase.err {
			t.Error()
		}
		if myVal != testCase.expected {
			t.Error()
		}
	}
}
