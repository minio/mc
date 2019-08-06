/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
		//success scenario the key contains non printable (tab) character as key
		{"s3/documents/=MzJieXRlc2xvbmdzZWNyZWFiY2RlZmcJZ2l2ZW5uMjE=", "s3/documents/=32byteslongsecreabcdefg	givenn21", nil, true},
		//success scenario the key contains non printable (tab character) as key
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
