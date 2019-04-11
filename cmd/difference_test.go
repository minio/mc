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
