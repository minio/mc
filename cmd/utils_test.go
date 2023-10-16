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
	"reflect"
	"testing"

	"github.com/minio/minio-go/v7/pkg/encrypt"
)

func TestParseEncryptionKeys(t *testing.T) {
	sseKey1, err := encrypt.NewSSEC([]byte("32byteslongsecretkeymustbegiven2"))
	if err != nil {
		t.Fatal(err)
	}
	sseKey2, err := encrypt.NewSSEC([]byte("32byteslongsecretkeymustbegiven1"))
	if err != nil {
		t.Fatal(err)
	}
	sseSpaceKey1, err := encrypt.NewSSEC([]byte("32byteslongsecret   mustbegiven1"))
	if err != nil {
		t.Fatal(err)
	}
	sseCommaKey1, err := encrypt.NewSSEC([]byte("32byteslongsecretkey,ustbegiven1"))
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		encryptionKey  string
		expectedEncMap map[string][]prefixSSEPair
		success        bool
	}{
		{
			encryptionKey: "myminio1/test2=32byteslongsecretkeymustbegiven2",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": {{
				Prefix: "myminio1/test2",
				SSE:    sseKey1,
			}}},
			success: true,
		},
		{
			encryptionKey:  "myminio1/test2=32byteslongsecretkeymustbegiven",
			expectedEncMap: nil,
			success:        false,
		},
		{
			encryptionKey: "myminio1/test2=32byteslongsecretkey,ustbegiven1",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": {{
				Prefix: "myminio1/test2",
				SSE:    sseCommaKey1,
			}}},
			success: true,
		},
		{
			encryptionKey: "myminio1/test2=32byteslongsecret   mustbegiven1",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": {{
				Prefix: "myminio1/test2",
				SSE:    sseSpaceKey1,
			}}},
			success: true,
		},
		{
			encryptionKey: "myminio1/test2=32byteslongsecretkeymustbegiven2,myminio1/test1/a=32byteslongsecretkeymustbegiven1",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": {{
				Prefix: "myminio1/test1/a",
				SSE:    sseKey2,
			}, {
				Prefix: "myminio1/test2",
				SSE:    sseKey1,
			}}},
			success: true,
		},
	}
	for i, testCase := range testCases {
		encMap, err := parseEncryptionKeys(testCase.encryptionKey)
		if err != nil && testCase.success {
			t.Fatalf("Test %d: Expected success, got %s", i+1, err)
		}
		if err == nil && !testCase.success {
			t.Fatalf("Test %d: Expected error, got success", i+1)
		}
		if testCase.success && !reflect.DeepEqual(encMap, testCase.expectedEncMap) {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedEncMap, encMap)
		}
	}
}

func TestParseAttribute(t *testing.T) {
	metaDataCases := []struct {
		input  string
		output map[string]string
		err    error
		status bool
	}{
		// // When blank value is passed.
		{"", map[string]string{}, ErrInvalidFileSystemAttribute, false},
		//  When space is passed.
		{"  ", map[string]string{}, ErrInvalidFileSystemAttribute, false},
		// When / is passed.
		{"/", map[string]string{}, ErrInvalidFileSystemAttribute, false},
		// When "atime:" is passed.
		{"atime:/", map[string]string{"atime": ""}, ErrInvalidFileSystemAttribute, false},
		// When "atime:" is passed.
		{"atime", map[string]string{"atime": ""}, nil, true},
		//  When "atime:" is passed.
		{"atime:", map[string]string{"atime": ""}, nil, true},
		// Passing a valid value
		{
			"atime:1/gid:1/gname:a/md:/mode:3/mtime:1/uid:1/uname:a",
			map[string]string{
				"atime": "1",
				"gid":   "1",
				"gname": "a",
				"md":    "",
				"mode":  "3",
				"mtime": "1",
				"uid":   "1",
				"uname": "a",
			},
			nil, true,
		},
	}

	for idx, testCase := range metaDataCases {
		meta, err := parseAttribute(map[string]string{
			metadataKey: testCase.input,
		})
		if testCase.status == true {
			if err != nil {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, err)
			}
			if !reflect.DeepEqual(meta, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, meta)
			}
		}
		if testCase.status == false {
			if !reflect.DeepEqual(meta, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, meta)
			}
			if err != testCase.err {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, err)
			}
		}

	}
}

func TestRemoveOverlappingPrefixes(t *testing.T) {
	testMap := make(map[string]struct {
		In  []string
		Out []string
	}, 0)

	testMap["no-overlapping"] = struct {
		In  []string
		Out []string
	}{
		In: []string{
			"alias1/bucket1/prefix",
		},
		Out: []string{"alias1/bucket1/prefix"},
	}

	testMap["no-overlapping-2x-sources"] = struct {
		In  []string
		Out []string
	}{
		In: []string{
			"alias1/bucket1/prefix",
			"alias2/bucket1/prefix",
		},
		Out: []string{
			"alias1/bucket1/prefix",
			"alias2/bucket1/prefix",
		},
	}

	testMap["basic-overlapping"] = struct {
		In  []string
		Out []string
	}{
		In: []string{
			"alias1/bucket1/prefix",
			"alias1/bucket1/prefix-1",
			"alias1/bucket1/prefix-2",
		},
		Out: []string{"alias1/bucket1/prefix"},
	}

	testMap["basic-overlapping-2x-prefix"] = struct {
		In  []string
		Out []string
	}{
		In: []string{
			"alias1/bucket1/prefix",
			"alias1/bucket1/prefix-1",
			"alias1/bucket1/prefix-2",
			"alias1/bucket1/another",
		},
		Out: []string{
			"alias1/bucket1/prefix",
			"alias1/bucket1/another",
		},
	}

	testMap["same-prefix-twice"] = struct {
		In  []string
		Out []string
	}{
		In: []string{
			"alias1/bucket1/prefix",
			"alias1/bucket1/prefix",
			"alias1/bucket1/prefix-2",
			"alias1/bucket1/prefix-2",
		},
		Out: []string{"alias1/bucket1/prefix"},
	}

	testMap["basic-overlapping-different-order"] = struct {
		In  []string
		Out []string
	}{
		In: []string{
			"alias1/bucket1/prefix-3",
			"alias1/bucket1/prefix-2",
			"alias1/bucket1/prefix-1",
			"alias1/bucket1/prefix",
		},
		Out: []string{"alias1/bucket1/prefix"},
	}

	testMap["same-prefix-X-times-different-order"] = struct {
		In  []string
		Out []string
	}{
		In: []string{
			"alias1/bucket1/prefix-3",
			"alias1/bucket1/prefix-2",
			"alias1/bucket1/prefix-1",
			"alias1/bucket1/prefix",
			"alias1/bucket1/prefix",
			"alias1/bucket1/prefix",
		},
		Out: []string{"alias1/bucket1/prefix"},
	}

	testMap["various-buckets-and-prefixes"] = struct {
		In  []string
		Out []string
	}{
		In: []string{
			"alias1/bucket1/prefix",
			"alias1/bucket1/prefix-1",
			"alias1/bucket1/prefix-2",
			"alias1/bucket1/another",
			"alias1/bucket1/another-1",
			"alias1/bucket1/another-2",

			"alias2/bucket1/prefix",
			"alias2/bucket1/prefix-1",
			"alias2/bucket1/another",
			"alias2/bucket1/another-1",
			"alias2/bucket1/another-2",
			"alias2/bucket1/another-3",

			"alias3/bucket1/prefix",
			"alias3/bucket1/prefix-1",
			"alias3/bucket1/prefix-2",
			"alias3/bucket1/prefix-3",
			"alias3/bucket1/another",
			"alias3/bucket1/another-1",

			"alias3/bucket2/prefix",
			"alias3/bucket2/prefix-1",
			"alias3/bucket2/another",
			"alias3/bucket2/another-1",
			"alias3/bucket2/another-2",
			"alias3/bucket2/another-3",
			"alias3/bucket2/another-4",
		},
		Out: []string{
			"alias1/bucket1/prefix",
			"alias1/bucket1/another",

			"alias2/bucket1/prefix",
			"alias2/bucket1/another",

			"alias3/bucket1/prefix",
			"alias3/bucket1/another",

			"alias3/bucket2/prefix",
			"alias3/bucket2/another",
		},
	}

	for i, v := range testMap {
		v.In = removeOverlappingPrefixes(v.In)
		if !reflect.DeepEqual(v.In, v.Out) {
			t.Fatal("Test", i, "failed.. expected:", v.Out, "but got:", v.In)
		}
	}
}
