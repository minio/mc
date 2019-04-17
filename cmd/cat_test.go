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

import (
	"bytes"
	"io"
	"io/ioutil"
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
		prettyText, err := ioutil.ReadAll(fakeStdout)
		if err != nil {
			t.Fatalf("Test %d: %v", i+1, err)
		}
		if string(prettyText) != testCase.prettyText {
			t.Fatalf("Test %d: expected output `%s`, found output `%s`", i+1, testCase.prettyText, string(prettyText))
		}
	}
}
