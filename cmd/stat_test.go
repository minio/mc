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
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseStat(t *testing.T) {
	localTime := time.Unix(12001, 0).UTC()
	testCases := []struct {
		content     ClientContent
		targetAlias string
	}{
		{ClientContent{URL: *newClientURL("https://play.min.io/abc"), Size: 0, Time: localTime, Type: os.ModeDir, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}, Expires: time.Now()}, "play"},
		{ClientContent{URL: *newClientURL("https://play.min.io/testbucket"), Size: 500, Time: localTime, Type: os.ModeDir, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}, Expires: time.Unix(0, 0).UTC()}, "play"},
		{ClientContent{URL: *newClientURL("https://s3.amazonaws.com/yrdy"), Size: 0, Time: localTime, Type: 0644, ETag: "abcdefasaas", Metadata: map[string]string{}}, "s3"},
		{ClientContent{URL: *newClientURL("https://play.min.io/yrdy"), Size: 10000, Time: localTime, Type: 0644, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}}, "play"},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run("", func(t *testing.T) {
			statMsg := parseStat(&testCase.content)
			if !reflect.DeepEqual(testCase.content.Metadata, statMsg.Metadata) {
				t.Errorf("Expecting %s, got %s", testCase.content.Metadata, statMsg.Metadata)
			}
			if testCase.content.Size != statMsg.Size {
				t.Errorf("Expecting %d, got %d", testCase.content.Size, statMsg.Size)
			}
			if testCase.content.Expires != statMsg.Expires {
				t.Errorf("Expecting %s, got %s", testCase.content.Expires, statMsg.Expires)
			}
			if testCase.content.Type.IsRegular() {
				if statMsg.Type != "file" {
					t.Errorf("Expecting file, got %s", statMsg.Type)
				}
			} else {
				if statMsg.Type != "folder" {
					t.Errorf("Expecting folder, got %s", statMsg.Type)
				}
			}
			etag := strings.TrimPrefix(testCase.content.ETag, "\"")
			etag = strings.TrimSuffix(etag, "\"")
			if etag != statMsg.ETag {
				t.Errorf("Expecting %s, got %s", etag, statMsg.ETag)
			}
		})
	}
}
