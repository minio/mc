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
	"context"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Tests match find function with all supported inputs on
// file pattern, size and time.
func TestMatchFind(t *testing.T) {
	// List of various contexts used in each tests,
	// tests are run in the same order as this list.
	listFindContexts := []*findContext{
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			ignorePattern: "*.go",
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			namePattern: "console",
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			pathPattern: "*console*",
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			regexPattern: regexp.MustCompile(`^(\d+\.){3}\d+$`),
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			olderThan: "1d",
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			newerThan: "32000d",
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			largerSize: 1024 * 1024,
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			smallerSize: 1024,
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
			ignorePattern: "*.txt",
		},
		{
			clnt: &S3Client{
				targetURL: &ClientURL{},
			},
		},
	}

	testCases := []struct {
		content       contentMessage
		expectedMatch bool
	}{
		// Matches ignore pattern, so match will be false - Test 1.
		{
			content: contentMessage{
				Key: "pkg/console/console.go",
			},
			expectedMatch: false,
		},
		// Matches name pattern - Test 2.
		{
			content: contentMessage{
				Key: "pkg/console/console.go",
			},
			expectedMatch: true,
		},
		// Matches path pattern - Test 3.
		{
			content: contentMessage{
				Key: "pkg/console/console.go",
			},
			expectedMatch: true,
		},
		// Matches regex pattern - Test 4.
		{
			content: contentMessage{
				Key: "192.168.1.1",
			},
			expectedMatch: true,
		},
		// Matches older than time - Test 5.
		{
			content: contentMessage{
				Time: time.Unix(11999, 0).UTC(),
			},
			expectedMatch: true,
		},
		// Matches newer than time - Test 6.
		{
			content: contentMessage{
				Time: time.Unix(12001, 0).UTC(),
			},
			expectedMatch: true,
		},
		// Matches size larger - Test 7.
		{
			content: contentMessage{
				Size: 1024 * 1024 * 2,
			},
			expectedMatch: true,
		},
		// Matches size smaller - Test 8.
		{
			content: contentMessage{
				Size: 1023,
			},
			expectedMatch: true,
		},
		// Does not match ignore pattern, so match will be true - Test 9.
		{
			content: contentMessage{
				Key: "pkg/console/console.go",
			},
			expectedMatch: true,
		},
		// No matching inputs were provided, so nothing to match return value is true - Test 10.
		{
			content:       contentMessage{},
			expectedMatch: true,
		},
	}

	// Runs all the test cases and validate the expected conditions.
	for i, testCase := range testCases {
		gotMatch := matchFind(listFindContexts[i], testCase.content)
		if testCase.expectedMatch != gotMatch {
			t.Errorf("Test: %d, expected match %t, got %t", i+1, testCase.expectedMatch, gotMatch)
		}
	}
}

// Tests suffix strings trimmed off correctly at maxdepth.
func TestSuffixTrimmingAtMaxDepth(t *testing.T) {
	testCases := []struct {
		startPrefix     string
		path            string
		separator       string
		maxDepth        uint
		expectedNewPath string
	}{
		// Tests at max depth 0.
		{
			startPrefix:     "./",
			path:            ".git/refs/remotes",
			separator:       "/",
			maxDepth:        0,
			expectedNewPath: ".git/refs/remotes",
		},
		// Tests at max depth 1.
		{
			startPrefix:     "./",
			path:            ".git/refs/remotes",
			separator:       "/",
			maxDepth:        1,
			expectedNewPath: "./.git/",
		},
		// Tests at max depth 2.
		{
			startPrefix:     "./",
			path:            ".git/refs/remotes",
			separator:       "/",
			maxDepth:        2,
			expectedNewPath: "./.git/refs/",
		},
		// Tests at max depth 3.
		{
			startPrefix:     "./",
			path:            ".git/refs/remotes",
			separator:       "/",
			maxDepth:        3,
			expectedNewPath: "./.git/refs/remotes",
		},
		// Tests with startPrefix empty.
		{
			startPrefix:     "",
			path:            ".git/refs/remotes",
			separator:       "/",
			maxDepth:        2,
			expectedNewPath: ".git/refs/",
		},
		// Tests with separator empty.
		{
			startPrefix:     "",
			path:            ".git/refs/remotes",
			separator:       "",
			maxDepth:        2,
			expectedNewPath: ".g",
		},
		// Tests with nested startPrefix paths - 1.
		{
			startPrefix:     ".git/refs/",
			path:            ".git/refs/remotes",
			separator:       "/",
			maxDepth:        1,
			expectedNewPath: ".git/refs/remotes",
		},
		// Tests with nested startPrefix paths - 2.
		{
			startPrefix:     ".git/refs",
			path:            ".git/refs/remotes",
			separator:       "/",
			maxDepth:        1,
			expectedNewPath: ".git/refs/",
		},
	}

	// Run all the test cases and validate for returned new path.
	for i, testCase := range testCases {
		gotNewPath := trimSuffixAtMaxDepth(testCase.startPrefix, testCase.path, testCase.separator, testCase.maxDepth)
		if testCase.expectedNewPath != gotNewPath {
			t.Errorf("Test: %d, expected path %s, got %s", i+1, testCase.expectedNewPath, gotNewPath)
		}
	}
}

// Tests matching functions for name, path and regex.
func TestFindMatch(t *testing.T) {
	// testFind is the structure used to contain params pertinent to find related tests
	type testFind struct {
		pattern, filePath, flagName string
		match                       bool
	}

	basicTests := []testFind{
		// Name match tests - success cases.
		{"*.jpg", "carter.jpg", "name", true},
		{"console", "pkg/console/console.go", "name", true},
		{"console.go", "pkg/console/console.go", "name", true},
		{"*XA==", "I/enjoy/morning/walks/XA==", "name ", true},
		{"*parser", "/This/might/mess up./the/parser", "name", true},
		{"*LTIxNDc0ODM2NDgvLTE=", "What/A/Naughty/String/LTIxNDc0ODM2NDgvLTE=", "name", true},
		{"*", "/bla/bla/bla/ ", "name", true},

		// Name match tests - failure cases.
		{"*.jpg", "carter.jpeg", "name", false},
		{"*/test/*", "/test/bob/likes/cake", "name", false},
		{"*test/*", "bob/test/likes/cake", "name", false},
		{"*test/*", "bob/likes/test/cake", "name", false},
		{"*/test/*", "bob/likes/cake/test", "name", false},
		{"*.jpg", ".jpg/elves/are/evil", "name", false},
		{
			"wq3YgNiB2ILYg9iE2IXYnNud3I/hoI7igIvigIzigI3igI7igI/igKrigKvigKzigK3igK7igaDi",
			"An/Even/Bigger/String/wq3YgNiB2ILYg9iE2IXYnNud3I/hoI7igIvigIzigI3igI7igI/igKrigKvigKzigK3igK7igaDi", "name", false,
		},
		{"𝕿𝖍𝖊", "well/this/isAN/odd/font/THE", "name", false},
		{"𝕿𝖍𝖊", "well/this/isAN/odd/font/The", "name", false},
		{"𝕿𝖍𝖊", "well/this/isAN/odd/font/𝓣𝓱𝓮", "name", false},
		{"𝕿𝖍𝖊", "what/a/strange/turn/of/events/𝓣he", "name", false},
		{"𝕿𝖍𝖊", "well/this/isAN/odd/font/𝕿𝖍𝖊", "name", true},

		// Path match tests - success cases.
		{"*/test/*", "bob/test/likes/cake", "path", true},
		{"*/test/*", "/test/bob/likes/cake", "path", true},

		// Path match tests - failure cases.
		{"*.jpg", ".jpg/elves/are/evil", "path", false},
		{"*/test/*", "test1/test2/test3/test", "path", false},
		{"*/ test /*", "test/test1/test2/test3/test", "path", false},
		{"*/test/*", " test /I/have/Really/Long/hair", "path", false},
		{"*XA==", "XA==/Height/is/a/social/construct", "path", false},
		{"*W", "/Word//this/is a/trickyTest", "path", false},
		{"LTIxNDc0ODM2NDgvLTE=", "LTIxNDc0ODM2NDgvLTE=/I/Am/One/Baaaaad/String", "path", false},
		{"/", "funky/path/name", "path", false},
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
		}
	}
}

// Tests string substitution function.
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
			expectedStr: `"0 B"`,
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
			expectedStr: `1.0 MiB`,
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
		gotStr := stringsReplace(context.Background(), testCase.str, testCase.content)
		if gotStr != testCase.expectedStr {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedStr, gotStr)
		}
	}
}

// Tests exit status, getExitStatus() function
func TestGetExitStatus(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping on non-linux")
		return
	}
	testCases := []struct {
		command            string
		expectedExitStatus int
	}{
		// Tests "No such file or directory", exit status code 2
		{
			command:            "ls asdf",
			expectedExitStatus: 2,
		},
		{
			command:            "cp x x",
			expectedExitStatus: 1,
		},
		// expectedExitStatus for "command not found" case is 127,
		// but exec command cannot capture anything since a process
		// for the command could not be started at all,
		// so the expectedExitStatus is 1
		{
			command:            "asdf",
			expectedExitStatus: 1,
		},
	}
	for i, testCase := range testCases {
		commandArgs := strings.Split(testCase.command, " ")
		cmd := exec.Command(commandArgs[0], commandArgs[1:]...)
		// Return exit status of the command run
		exitStatus := getExitStatus(cmd.Run())
		if exitStatus != testCase.expectedExitStatus {
			t.Errorf("Test %d: Expected error status code for command \"%v\" is %v, got %v",
				i+1, testCase.command, testCase.expectedExitStatus, exitStatus)
		}
	}
}
