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
)

var testCases = []struct {
	pattern []string

	object string

	match bool
}{
	{nil, "testfile", false},
	{[]string{"test*"}, "testfile", true},
	{[]string{"file*"}, "file/abc/bcd/def", true},
	{[]string{"*"}, "file/abc/bcd/def", true},
	{[]string{""}, "file/abc/bcd/def", false},
	{[]string{"abc*"}, "file/abc/bcd/def", false},
	{[]string{"abc*", "*abc/*"}, "file/abc/bcd/def", true},
	{[]string{"*.txt"}, "file/abc/bcd/def.txt", true},
	{[]string{".*"}, ".sys", true},
	{[]string{"*."}, ".sys.", true},
}

func TestExcludeOptions(t *testing.T) {
	for _, test := range testCases {
		if matchExcludeOptions(test.pattern, test.object) != test.match {
			t.Fatalf("Unexpected result %t, with pattern %s and object %s \n", !test.match, test.pattern, test.object)
		}
	}
}
