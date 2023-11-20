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
    "os"
    "fmt"
    "reflect"
    "strings"
    "testing"
    "time"
    "github.com/dustin/go-humanize"
)

func TestParseStat(t *testing.T) {
    localTime := time.Unix(12001, 0).UTC()
    testCases := []struct {
        content              ClientContent
        targetAlias          string
        expectedHumanizedSize string
}{
		{
        content: ClientContent{
            URL: *newClientURL("https://play.min.io/abc"), 
            Size: 1000, 
            Time: localTime, 
            Type: os.ModeDir, 
            ETag: "blahblah", 
            Metadata: map[string]string{"custom-key": "custom-value"}, 
            Expires: time.Now(),
        }, 
        targetAlias: "play", 
        expectedHumanizedSize: "1,000",
    },
    {
        content: ClientContent{
            URL: *newClientURL("https://play.min.io/abc"), 
            Size: 0, 
            Time: localTime, 
            Type: os.ModeDir, 
            ETag: "blahblah", 
            Metadata: map[string]string{"custom-key": "custom-value"}, 
            Expires: time.Now(),
        }, 
        targetAlias: "play", 
        expectedHumanizedSize: "0",
    },
    {
        content: ClientContent{
            URL: *newClientURL("https://play.min.io/testbucket"), 
            Size: 500, 
            Time: localTime, 
            Type: os.ModeDir, 
            ETag: "blahblah", 
            Metadata: map[string]string{"custom-key": "custom-value"}, 
            Expires: time.Unix(0, 0).UTC(),
        }, 
        targetAlias: "play", 
        expectedHumanizedSize: "500",
    },
    {
        content: ClientContent{
            URL: *newClientURL("https://s3.amazonaws.com/yrdy"), 
            Size: 0, 
            Time: localTime, 
            Type: 0o644, 
            ETag: "abcdefasaas", 
            Metadata: map[string]string{},
        }, 
        targetAlias: "s3", 
        expectedHumanizedSize: "0",
    },
    {
        content: ClientContent{
            URL: *newClientURL("https://play.min.io/yrdy"), 
            Size: 10000, 
            Time: localTime, 
            Type: 0o644, 
            ETag: "blahblah", 
            Metadata: map[string]string{"custom-key": "custom-value"},
        }, 
        targetAlias: "play", 
        expectedHumanizedSize: "10,000",
    },
}

			
	// 		}{
	// 	{content: ClientContent{URL: *newClientURL("https://play.min.io/abc"), Size: 1000, Time: localTime, Type: os.ModeDir, ETag: "blahblah", Metadata: map[string]string{"custom-key": "custom-value"}, Expires: time.Now()}, "play", "1,000"},
	// 	{content: ClientContent{URL: *newClientURL("https://play.min.io/abc"), Size: 0, Time: localTime, Type: os.ModeDir, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}, Expires: time.Now()}, "play"},
	// 	{content: ClientContent{URL: *newClientURL("https://play.min.io/testbucket"), Size: 500, Time: localTime, Type: os.ModeDir, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}, Expires: time.Unix(0, 0).UTC()}, "play"},
	// 	{content: ClientContent{URL: *newClientURL("https://s3.amazonaws.com/yrdy"), Size: 0, Time: localTime, Type: 0o644, ETag: "abcdefasaas", Metadata: map[string]string{}}, "s3"},
	// 	{content: ClientContent{URL: *newClientURL("https://play.min.io/yrdy"), Size: 10000, Time: localTime, Type: 0o644, ETag: "blahblah", Metadata: map[string]string{"cusom-key": "custom-value"}}, "play"},
	// }
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
			if statMsg.Expires != nil {
				if testCase.content.Expires != *statMsg.Expires {
					t.Errorf("Expecting %s, got %s", testCase.content.Expires, statMsg.Expires)
				}
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
            humanizedSize := humanize.Comma(int64(testCase.content.Size))
            if humanizedSize != testCase.expectedHumanizedSize {
                t.Errorf("Expected humanized size %s, got %s for size %d", testCase.expectedHumanizedSize, humanizedSize, testCase.content.Size)
            }
		})
	}
}

// Mock data representing a large object count
func TestHumanizedHistogramOutput(t *testing.T) {
    largeObjectCount := uint64(1000000000) // 1 billion
    humanizedCount := humanize.Comma(int64(largeObjectCount))
    output := fmt.Sprintf("%12s objects", humanizedCount)
    expected := "1,000,000,000 objects"

    if output != expected {
        t.Errorf("expected %s, got %s", expected, output)
    }
}
