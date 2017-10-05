/*
 * Minio Client (C) 2017 Minio, Inc.
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

package cmd

import (
	"testing"
	"time"
)

// Tests matching functions for name, path and regex.
func TestFindMatch(t *testing.T) {
	// testFind is the structure used to contain params pertinent to find related tests
	type testFind struct {
		pattern, filePath, flagName string
		match                       bool
	}

	var basicTests = []testFind{
		// Basic name and path tests
		{"*.jpg", "carter.jpg", "name", true},
		{"*.jpg", "carter.jpeg", "name", false},
		{"*/test/*", "/test/bob/likes/cake", "name", false},
		{"*/test/*", "/test/bob/likes/cake", "path", true},
		{"*test/*", "bob/test/likes/cake", "name", false},
		{"*/test/*", "bob/test/likes/cake", "path", true},
		{"*test/*", "bob/likes/test/cake", "name", false},

		// More advanced name and path tests
		{"*/test/*", "bob/likes/cake/test", "name", false},
		{"*.jpg", ".jpg/elves/are/evil", "name", false},
		{"*.jpg", ".jpg/elves/are/evil", "path", false},
		{"*/test/*", "test1/test2/test3/test", "path", false},
		{"*/ test /*", "test/test1/test2/test3/test", "path", false},
		{"*/test/*", " test /I/have/Really/Long/hair", "path", false},
		{"*XA==", "I/enjoy/morning/walks/XA==", "name ", true},
		{"*XA==", "XA==/Height/is/a/social/construct", "path", false},
		{"*W", "/Word//this/is a/trickyTest", "path", false},
		{"*parser", "/This/might/mess up./the/parser", "name", true},
		{"*", "/bla/bla/bla/ ", "name", true},
		{"*LTIxNDc0ODM2NDgvLTE=", "What/A/Naughty/String/LTIxNDc0ODM2NDgvLTE=", "name", true},
		{"LTIxNDc0ODM2NDgvLTE=", "LTIxNDc0ODM2NDgvLTE=/I/Am/One/Baaaaad/String", "path", false},
		{"wq3YgNiB2ILYg9iE2IXYnNud3I/hoI7igIvigIzigI3igI7igI/igKrigKvigKzigK3igK7igaDi",
			"An/Even/Bigger/String/wq3YgNiB2ILYg9iE2IXYnNud3I/hoI7igIvigIzigI3igI7igI/igKrigKvigKzigK3igK7igaDi", "name", false},
		{"/", "funky/path/name", "path", false},
		{"𝕿𝖍𝖊", "well/this/isAN/odd/font/THE", "name", false},
		{"𝕿𝖍𝖊", "well/this/isAN/odd/font/The", "name", false},
		{"𝕿𝖍𝖊", "well/this/isAN/odd/font/𝓣𝓱𝓮", "name", false},
		{"𝕿𝖍𝖊", "what/a/strange/turn/of/events/𝓣he", "name", false},
		{"𝕿𝖍𝖊", "well/this/isAN/odd/font/𝕿𝖍𝖊", "name", true},

		// Regexp based.
		{"^[a-zA-Z][a-zA-Z0-9\\-]+[a-zA-Z0-9]$", "testbucket-1", "regex", true},
		{"^[a-zA-Z][a-zA-Z0-9\\-]+[a-zA-Z0-9]$", "testbucket.", "regex", false},
		{`^(\d+\.){3}\d+$`, "192.168.1.1", "regex", true},
		{`^(\d+\.){3}\d+$`, "192.168.x.x", "regex", false},
	}

	for _, test := range basicTests {
		switch test.flagName {
		case "name":
			testMatch := nameMatch(test.pattern, test.filePath)
			if testMatch != test.match {
				t.Fatalf("Unexpected result %t, with pattern %s, flag %s  and filepath %s \n",
					!test.match, test.pattern, test.flagName, test.filePath)
			}
		case "path":
			testMatch := pathMatch(test.pattern, test.filePath)
			if testMatch != test.match {
				t.Fatalf("Unexpected result %t, with pattern %s, flag %s and filepath %s \n",
					!test.match, test.pattern, test.flagName, test.filePath)
			}
		case "regex":
			testMatch := regexMatch(test.pattern, test.filePath)
			if testMatch != test.match {
				t.Fatalf("Unexpected result %t, with pattern %s, flag %s and filepath %s \n",
					!test.match, test.pattern, test.flagName, test.filePath)
			}
		}
	}
}

// Tests for parsing time layout.
func TestParseTime(t *testing.T) {
	testCases := []struct {
		value   string
		success bool
	}{
		// Parses 1 day successfully.
		{
			value:   "1d",
			success: true,
		},
		// Parses 1 week successfully.
		{
			value:   "1w",
			success: true,
		},
		// Parses 1 year successfully.
		{
			value:   "1y",
			success: true,
		},
		// Parses 2 months successfully.
		{
			value:   "2m",
			success: true,
		},
		// Failure to parse "xd".
		{
			value:   "xd",
			success: false,
		},
		// Failure to parse empty string.
		{
			value:   "",
			success: false,
		},
	}
	for i, testCase := range testCases {
		pt, err := parseTime(testCase.value)
		if err != nil && testCase.success {
			t.Errorf("Test: %d, Expected to be successful, but found error %s, for time value %s", i+1, err, testCase.value)
		}
		if pt.IsZero() && testCase.success {
			t.Errorf("Test: %d, Expected time to be non zero, but found zero time for time value %s", i+1, testCase.value)
		}
		if err == nil && !testCase.success {
			t.Errorf("Test: %d, Expected error but found to be successful for time value %s", i+1, testCase.value)
		}
	}
}

// Tests string substritution function.
func TestStringReplace(t *testing.T) {
	testCases := []struct {
		str         string
		expectedStr string
		content     contentMessage
	}{
		// Tests string replace {} without quotes.
		{
			str:         "{}",
			expectedStr: "path/1",
			content:     contentMessage{Key: "path/1"},
		},
		// Tests string replace {} with quotes.
		{
			str:         `{""}`,
			expectedStr: `"path/1"`,
			content:     contentMessage{Key: "path/1"},
		},
		// Tests string replace {base}
		{
			str:         "{base}",
			expectedStr: "1",
			content:     contentMessage{Key: "path/1"},
		},
		// Tests string replace {"base"} with quotes.
		{
			str:         `{"base"}`,
			expectedStr: `"1"`,
			content:     contentMessage{Key: "path/1"},
		},
		// Tests string replace {dir}
		{
			str:         `{dir}`,
			expectedStr: `path`,
			content:     contentMessage{Key: "path/1"},
		},
		// Tests string replace {"dir"} with quotes.
		{
			str:         `{"dir"}`,
			expectedStr: `"path"`,
			content:     contentMessage{Key: "path/1"},
		},
		// Tests string replace {"size"} with quotes.
		{
			str:         `{"size"}`,
			expectedStr: `"0B"`,
			content:     contentMessage{Size: 0},
		},
		// Tests string replace {"time"} with quotes.
		{
			str:         `{"time"}`,
			expectedStr: `"2038-01-19 03:14:07 UTC"`,
			content: contentMessage{
				Time: time.Unix(2147483647, 0).UTC(),
			},
		},
		// Tests string replace {size}
		{
			str:         `{size}`,
			expectedStr: `1.0MiB`,
			content:     contentMessage{Size: 1024 * 1024},
		},
		// Tests string replace {time}
		{
			str:         `{time}`,
			expectedStr: `2038-01-19 03:14:07 UTC`,
			content: contentMessage{
				Time: time.Unix(2147483647, 0).UTC(),
			},
		},
	}
	for i, testCase := range testCases {
		gotStr := stringsReplace(testCase.str, testCase.content)
		if gotStr != testCase.expectedStr {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedStr, gotStr)
		}
	}
}
