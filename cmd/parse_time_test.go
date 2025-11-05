// Copyright (c) 2015-2022 MinIO, Inc.
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
)

var parseDurationTests = []struct {
	in   string
	ok   bool
	want Duration
}{
	// simple
	{"0", true, 0},
	{"5s", true, 5 * Second},
	{"30s", true, 30 * Second},
	{"1478s", true, 1478 * Second},
	// sign
	{"-5s", true, -5 * Second},
	{"+5s", true, 5 * Second},
	{"-0", true, 0},
	{"+0", true, 0},
	// decimal
	{"5.0s", true, 5 * Second},
	{"5.6s", true, 5*Second + 600*Millisecond},
	{"5.s", true, 5 * Second},
	{".5s", true, 500 * Millisecond},
	{"1.0s", true, 1 * Second},
	{"1.00s", true, 1 * Second},
	{"1.004s", true, 1*Second + 4*Millisecond},
	{"1.0040s", true, 1*Second + 4*Millisecond},
	{"100.00100s", true, 100*Second + 1*Millisecond},
	// different units
	{"10ns", true, 10 * Nanosecond},
	{"11us", true, 11 * Microsecond},
	{"12µs", true, 12 * Microsecond}, // U+00B5
	{"12μs", true, 12 * Microsecond}, // U+03BC
	{"13ms", true, 13 * Millisecond},
	{"14s", true, 14 * Second},
	{"15m", true, 15 * Minute},
	{"16h", true, 16 * Hour},
	{"12d", true, 12 * Day},
	{"3w", true, 3 * Week},
	// composite durations
	{"3h30m", true, 3*Hour + 30*Minute},
	{"10.5s4m", true, 4*Minute + 10*Second + 500*Millisecond},
	{"-2m3.4s", true, -(2*Minute + 3*Second + 400*Millisecond)},
	{"1h2m3s4ms5us6ns", true, 1*Hour + 2*Minute + 3*Second + 4*Millisecond + 5*Microsecond + 6*Nanosecond},
	{"39h9m14.425s", true, 39*Hour + 9*Minute + 14*Second + 425*Millisecond},
	{"2w3d12h", true, 2*Week + 3*Day + 12*Hour},
	// large value
	{"52763797000ns", true, 52763797000 * Nanosecond},
	// more than 9 digits after decimal point, see https://golang.org/issue/6617
	{"0.3333333333333333333h", true, 20 * Minute},
	// 9007199254740993 = 1<<53+1 cannot be stored precisely in a float64
	{"9007199254740993ns", true, (1<<53 + 1) * Nanosecond},
	// largest duration that can be represented by int64 in nanoseconds
	{"9223372036854775807ns", true, (1<<63 - 1) * Nanosecond},
	{"9223372036854775.807us", true, (1<<63 - 1) * Nanosecond},
	{"9223372036s854ms775us807ns", true, (1<<63 - 1) * Nanosecond},
	// large negative value
	{"-9223372036854775807ns", true, -1<<63 + 1*Nanosecond},

	// errors
	{"", false, 0},
	{"3", false, 0},
	{"-", false, 0},
	{"s", false, 0},
	{".", false, 0},
	{"-.", false, 0},
	{".s", false, 0},
	{"+.s", false, 0},
	{"3000000h", false, 0},                  // overflow
	{"9223372036854775808ns", false, 0},     // overflow
	{"9223372036854775.808us", false, 0},    // overflow
	{"9223372036854ms775us808ns", false, 0}, // overflow
	// largest negative value of type int64 in nanoseconds should fail
	// see https://go-review.googlesource.com/#/c/2461/
	{"-9223372036854775808ns", false, 0},
}

func TestParseDuration(t *testing.T) {
	for _, tc := range parseDurationTests {
		t.Run(tc.in, func(t *testing.T) {
			d, err := ParseDuration(tc.in)
			if tc.ok && (err != nil || d != tc.want) {
				t.Errorf("ParseDuration(%q) = %v, %v, want %v, nil", tc.in, d, err, tc.want)
			} else if !tc.ok && err == nil {
				t.Errorf("ParseDuration(%q) = _, nil, want _, non-nil", tc.in)
			}
		})
	}
}

// Test for ParseDurationTime. Validates the returned value
// for given time value in days, hours and minute format.
func TestParseDurationTime(t *testing.T) {
	testCases := []struct {
		timeValue string
		expected  Duration
		err       string
	}{
		// Test 1: empty string as input
		{"", 0, "invalid empty duration"},
		// Test 2: Input string contains 4 day, 10 hour and 3 minutes
		{"4d10h3m", 381780000000000, ""},
		// Test 3: Input string contains 10 day and  3 hours
		{"10d3h", 874800000000000, ""},
		// Test 4: Input string contains minutes and days
		{"3m7d", 604980000000000, ""},
		// Test 5: Input string contains unknown unit
		{"4a3d", 0, "unknown unit a in duration 4a3d"},
		// Test 6: Input string contains fractional day
		{"1.5d", 129600000000000, ""},
		// Test 6: Input string contains fractional day and hour
		{"2.5d1.5h", 221400000000000, ""},
		// Test 7: Input string contains fractional day , hour and minute
		{"2.5d1.5h3.5m", 221610000000000, ""},
	}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			myVal, err := ParseDuration(testCase.timeValue)
			if err != nil && err.Error() != testCase.err {
				t.Error()
			}
			if myVal != testCase.expected {
				t.Errorf("Expected %v, got %v", testCase.expected, myVal)
			}
		})
	}
}
