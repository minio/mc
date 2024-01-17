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

var testCases = []struct {
	pattern []string

	srcSuffix string

	match bool

	typ ClientURLType
}{
	{nil, "testfile", false, objectStorage},
	{[]string{"test*"}, "testfile", true, objectStorage},
	{[]string{"file*"}, "file/abc/bcd/def", true, objectStorage},
	{[]string{"*"}, "file/abc/bcd/def", true, objectStorage},
	{[]string{""}, "file/abc/bcd/def", false, objectStorage},
	{[]string{"abc*"}, "file/abc/bcd/def", false, objectStorage},
	{[]string{"abc*", "*abc/*"}, "file/abc/bcd/def", true, objectStorage},
	{[]string{"*.txt"}, "file/abc/bcd/def.txt", true, objectStorage},
	{[]string{".*"}, ".sys", true, objectStorage},
	{[]string{"*."}, ".sys.", true, objectStorage},
	{nil, "testfile", false, fileSystem},
	{[]string{"test*"}, "testfile", true, fileSystem},
	{[]string{"file*"}, "file/abc/bcd/def", true, fileSystem},
	{[]string{"*"}, "file/abc/bcd/def", true, fileSystem},
	{[]string{""}, "file/abc/bcd/def", false, fileSystem},
	{[]string{"abc*"}, "file/abc/bcd/def", false, fileSystem},
	{[]string{"abc*", "*abc/*"}, "file/abc/bcd/def", true, fileSystem},
	{[]string{"abc*", "*abc/*"}, "/file/abc/bcd/def", true, fileSystem},
	{[]string{"*.txt"}, "file/abc/bcd/def.txt", true, fileSystem},
	{[]string{"*.txt"}, "/file/abc/bcd/def.txt", true, fileSystem},
	{[]string{".*"}, ".sys", true, fileSystem},
	{[]string{"*."}, ".sys.", true, fileSystem},
}

func TestExcludeOptions(t *testing.T) {
	for _, test := range testCases {
		if matchExcludeOptions(test.pattern, test.srcSuffix, test.typ) != test.match {
			t.Fatalf("Unexpected result %t, with pattern %s and srcSuffix %s \n", !test.match, test.pattern, test.srcSuffix)
		}
	}
}
