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
)

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
