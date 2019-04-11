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
	"strings"
	"time"

	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestParseStat(c *C) {
	localTime := time.Unix(12001, 0).UTC()
	testCases := []struct {
		content     clientContent
		targetAlias string
	}{
		{clientContent{URL: *newClientURL("https://play.min.io:9000/abc"), Size: 0, Time: localTime, Type: os.ModeDir, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}, EncryptionHeaders: map[string]string{}, Expires: time.Now()},
			"play"},
		{clientContent{URL: *newClientURL("https://play.min.io:9000/testbucket"), Size: 500, Time: localTime, Type: os.ModeDir, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}, EncryptionHeaders: map[string]string{}, Expires: time.Unix(0, 0).UTC()},
			"play"},
		{clientContent{URL: *newClientURL("https://s3.amazonaws.com/yrdy"), Size: 0, Time: localTime, Type: 0644, ETag: "abcdefasaas", Metadata: map[string]string{}, EncryptionHeaders: map[string]string{}},
			"s3"},
		{clientContent{URL: *newClientURL("https://play.min.io:9000/yrdy"), Size: 10000, Time: localTime, Type: 0644, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}, EncryptionHeaders: map[string]string{"X-Amz-Iv": "test", "X-Amz-Matdesc": "abcd"}},
			"play"},
	}
	for _, testCase := range testCases {
		statMsg := parseStat(&testCase.content)
		c.Assert(testCase.content.Metadata, DeepEquals, statMsg.Metadata)
		c.Assert(testCase.content.EncryptionHeaders, DeepEquals, statMsg.EncryptionHeaders)
		c.Assert(testCase.content.Size, Equals, statMsg.Size)
		c.Assert(testCase.content.Expires, Equals, statMsg.Expires)
		c.Log(statMsg.Type)
		if testCase.content.Type.IsRegular() {
			c.Assert(statMsg.Type, Equals, "file")
		} else {
			c.Assert(statMsg.Type, Equals, "folder")
		}
		etag := strings.TrimPrefix(testCase.content.ETag, "\"")
		etag = strings.TrimSuffix(etag, "\"")
		c.Assert(etag, Equals, statMsg.ETag)
	}
}
