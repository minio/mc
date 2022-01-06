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
	"errors"
	"reflect"
	"testing"
)

func TestGetDecodedKey(t *testing.T) {
	getDecodeCases := []struct {
		input  string
		output string
		err    error
		status bool
	}{
		// success scenario the key contains non printable (tab) character as key
		{"s3/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=", "s3/documents/=32byteslongsecreabcdefg	givenn21", nil, true},
		// success scenario the key contains non printable (tab character) as key
		{"s3/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=,play/documents/=MzJieXRlc2xvbmdzZWNyZXRrZQltdXN0YmVnaXZlbjE=", "s3/documents/=32byteslongsecreabcdefg	givenn21,play/documents/=32byteslongsecretke	mustbegiven1", nil, true},
		// success scenario using a normal string
		{"s3/documents/=32byteslongsecretkeymustbegiven1", "s3/documents/=32byteslongsecretkeymustbegiven1", nil, true},
		// success scenario using a normal string
		{"s3/documents/=32byteslongsecretkeymustbegiven1,myminio/documents/=32byteslongsecretkeymustbegiven2", "s3/documents/=32byteslongsecretkeymustbegiven1,myminio/documents/=32byteslongsecretkeymustbegiven2", nil, true},
		// success scenario using a mix of normal string and encoded string
		{"s3/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=,play/documents/=32byteslongsecretkeymustbegiven1", "s3/documents/=32byteslongsecreabcdefg	givenn21,play/documents/=32byteslongsecretkeymustbegiven1", nil, true},
		// success scenario using a mix of normal string and encoded string
		{"play/documents/=32byteslongsecretkeymustbegiven1,s3/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=", "play/documents/=32byteslongsecretkeymustbegiven1,s3/documents/=32byteslongsecreabcdefg	givenn21", nil, true},
		// decoded key less than 32 char and conatin non printable (tab) character
		{"s3/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE", "", errors.New("Encryption key should be 32 bytes plain text key or 44 bytes base64 encoded key"), false},
		// normal key less than 32 character
		{"s3/documents/=32byteslongsecretkeymustbegiven", "", errors.New("Encryption key should be 32 bytes plain text key or 44 bytes base64 encoded key"), false},
	}

	for idx, testCase := range getDecodeCases {
		decodedString, errDecode := getDecodedKey(testCase.input)
		if testCase.status == true {
			if errDecode != nil {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, errDecode)
			}
			if !reflect.DeepEqual(decodedString, testCase.output) {
				t.Fatalf("Test %d: generated key not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, decodedString)
			}
		}

		if testCase.status == false {
			if !reflect.DeepEqual(decodedString, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, errDecode)
			}
			if errDecode.Cause.Error() != testCase.err.Error() {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, errDecode)
			}
		}
	}
}
