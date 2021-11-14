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
	filters []nameFilter

	name string

	excluded bool
}{
	{nil, "testfile", false},
	{[]nameFilter{excludeWildcardFilter{"test*"}}, "testfile", true},
	{[]nameFilter{excludeWildcardFilter{"file*"}}, "file/abc/bcd/def", true},
	{[]nameFilter{excludeWildcardFilter{"*"}}, "file/abc/bcd/def", true},
	{[]nameFilter{excludeWildcardFilter{""}}, "file/abc/bcd/def", false},
	{[]nameFilter{excludeWildcardFilter{"abc*"}}, "file/abc/bcd/def", false},
	{[]nameFilter{excludeWildcardFilter{"abc*"}, excludeWildcardFilter{"*abc/*"}}, "file/abc/bcd/def", true},
	{[]nameFilter{excludeWildcardFilter{"*.txt"}}, "file/abc/bcd/def.txt", true},
	{[]nameFilter{excludeWildcardFilter{".*"}}, ".sys", true},
	{[]nameFilter{excludeWildcardFilter{"*."}}, ".sys.", true},

	// select all files. Filter does not affect.
	{[]nameFilter{includeWildcardFilter{"*.zip"}}, "backup.zip", false},
	{[]nameFilter{includeWildcardFilter{"*.zip"}}, "some.log", false},

	// Exclude all and ignore empty inclusion
	{[]nameFilter{
		excludeWildcardFilter{"*"},
		includeWildcardFilter{""},
	}, "some.log", true},

	// select zips only
	{[]nameFilter{
		excludeWildcardFilter{"*"},
		includeWildcardFilter{"*.zip"},
	}, "some.log", true},
	{[]nameFilter{
		excludeWildcardFilter{"*"},
		includeWildcardFilter{"*.zip"},
	}, "backup.zip", false},

	// select zips except ignored zips
	{[]nameFilter{
		excludeWildcardFilter{"*"},
		includeWildcardFilter{"*.zip"},
		excludeWildcardFilter{"ignore*.zip"},
	}, "backup.zip", false},
	{[]nameFilter{
		excludeWildcardFilter{"*"},
		includeWildcardFilter{"*.zip"},
		excludeWildcardFilter{"ignore*.zip"},
	}, "ignored.zip", true},

	// select all, ignore logs except important.log
	{[]nameFilter{
		excludeWildcardFilter{"*.log"},
		includeWildcardFilter{"important.log"},
	}, "important.log", false},

	// select zips and important.log
	{[]nameFilter{
		excludeWildcardFilter{"*"},
		includeWildcardFilter{"*.zip"},
		includeWildcardFilter{"important.log"},
	}, "important.log", false},

	// select all except zips and logs but select important zips
	{[]nameFilter{
		excludeWildcardFilter{"*.zip"},
		excludeWildcardFilter{"*.log"},
		includeWildcardFilter{"important*.zip"},
	}, "important.zip", false},
}

func TestExcludeOptions(t *testing.T) {
	for _, test := range testCases {
		if shouldExcludeFileByFilters(test.filters, test.name) != test.excluded {
			t.Fatalf("Unexpected result %t, with filters %s and name %s \n",
				!test.excluded, test.filters, test.name)
		}
	}
}
