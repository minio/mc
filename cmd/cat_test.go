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
	"bytes"
	"io"
	"testing"
)

func TestPrettyStdout(t *testing.T) {
	testCases := []struct {
		originText string
		prettyText string
	}{
		{"", ""},
		{"text", "text"},
		// Check with new lines
		{"text\r\n", "text\r\n"},
		{"\ttext\n", "\ttext\n"},
		// Print some unicode characters and check if it is not altered
		{"добро пожаловать.", "добро пожаловать."},
		// Print colored text
		{"\x1b\x5b\x33\x31\x6d\x66\x61\x69\x6c", "^?[31mfail"},
		// Print clear screen
		{"\x1b\x63", "^?c"},
		// Random data
		{"\x3d\xef\xd2\xb5", "=�ҵ"},
	}

	for i, testCase := range testCases {
		reader := bytes.NewReader([]byte(testCase.originText))
		fakeStdout := bytes.NewBuffer([]byte(""))
		n, err := io.Copy(newPrettyStdout(fakeStdout), reader)
		if err != nil {
			t.Fatalf("Test %d: %v\n", i+1, err)
		}
		if int(n) != len(testCase.originText) {
			t.Fatalf("Test %d: copy error\n", i+1)
		}
		prettyText, err := io.ReadAll(fakeStdout)
		if err != nil {
			t.Fatalf("Test %d: %v", i+1, err)
		}
		if string(prettyText) != testCase.prettyText {
			t.Fatalf("Test %d: expected output `%s`, found output `%s`", i+1, testCase.prettyText, string(prettyText))
		}
	}
}
