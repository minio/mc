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

import "testing"

// TestPrettyTable - testing the behavior of the pretty table module
func TestPrettyTable(t *testing.T) {
	testCases := []struct {
		sep         string
		tf          []Field
		contents    []string
		expectedRow string
	}{
		// Test 1: empty parameter, empty table
		{"", []Field{}, []string{}, ""},
		// Test 2: one field, without any specific customization
		{"", []Field{{"", -1}}, []string{"abcd"}, "abcd"},
		// Test 3: one field, without 5 chars len
		{"", []Field{{"", 5}}, []string{"my-long-field"}, "my..."},
		// Test 4: one separator, one content
		{" | ", []Field{{"", -1}}, []string{"abcd"}, "abcd"},
		// Test 5: one separtor, multiple contents
		{" | ", []Field{{"", -1}, {"", -1}, {"", -1}}, []string{"column1", "column2", "column3"}, "column1 | column2 | column3"},
		// Test 6: multiple fields
		{" | ", []Field{{"", 5}, {"", -1}}, []string{"144550032", "my long content that should not be cut"}, "14... | my long content that should not be cut"},
	}

	for idx, testCase := range testCases {
		tb := newPrettyTable(testCase.sep, testCase.tf...)
		row := tb.buildRow(testCase.contents...)
		if row != testCase.expectedRow {
			t.Fatalf("Test %d: generated row not matching, expected = `%s`, found = `%s`", idx+1, testCase.expectedRow, row)
		}
	}
}
