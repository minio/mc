/*
 * MinIO Client (C) 2017 MinIO, Inc.
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
