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
	"reflect"
	"testing"
)

func TestParseMetaData(t *testing.T) {
	metaDataCases := []struct {
		input  string
		output map[string]string
		err    error
		status bool
	}{
		// success scenerio
		{"key1=value1,key2=value2", map[string]string{"key1": "value1", "key2": "value2"}, nil, true},
		// using different delimitter, other than ',' between multiple meta data
		{"key1=value1;key2=value2", nil, ErrInvalidMetadata, false},
		// using different delimitter, other than '=' between key value
		{"key1:value1,key2:value2", nil, ErrInvalidMetadata, false},
		// using no delimitter
		{"key1:value1:key2:value2", nil, ErrInvalidMetadata, false},
	}

	for idx, testCase := range metaDataCases {
		metaDatamap, errMeta := getMetaDataEntry(testCase.input)
		if testCase.status == true {
			if errMeta != nil {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, errMeta)
			}
			if !reflect.DeepEqual(metaDatamap, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, metaDatamap)
			}
		}

		if testCase.status == false {
			if !reflect.DeepEqual(metaDatamap, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, metaDatamap)
			}
			if errMeta.Cause.Error() != testCase.err.Error() {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, errMeta)
			}
		}
	}
}
