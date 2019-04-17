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
	"reflect"
	"testing"

	"github.com/minio/minio-go/pkg/encrypt"
)

func TestParseURLEnv(t *testing.T) {
	testCases := []struct {
		envURL         string
		expectedURL    string
		expectedAccess string
		expectedSecret string
		success        bool
	}{
		{
			envURL:         "https://username:password@play.min.io:9000/",
			expectedURL:    "https://play.min.io:9000/",
			expectedAccess: "username",
			expectedSecret: "password",
			success:        true,
		},
		{
			envURL:      "https://play.min.io:9000/",
			expectedURL: "https://play.min.io:9000/",
			success:     true,
		},
		{
			envURL:  "ftp://play.min.io:9000/",
			success: false,
		},
		{
			envURL:  "",
			success: false,
		},
		{
			envURL:  "https://play.min.io:9000/path",
			success: false,
		},
		{
			envURL:  "https://play.min.io:9000/?path=value",
			success: false,
		},
	}

	for i, testCase := range testCases {
		u, accessKey, secretKey, err := parseEnvURL(testCase.envURL)
		if err != nil && testCase.success {
			t.Fatalf("Test %d: Expected success, got %s", i+1, err)
		}
		if err == nil && !testCase.success {
			t.Fatalf("Test %d: Expected error, got success", i+1)
		}
		if accessKey != testCase.expectedAccess {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedAccess, accessKey)
		}
		if secretKey != testCase.expectedSecret {
			t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedSecret, secretKey)
		}
		if err == nil {
			if u.String() != testCase.expectedURL {
				t.Errorf("Test %d: Expected %s, got %s", i+1, testCase.expectedURL, u.String())
			}
		}
	}
}

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
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": []prefixSSEPair{prefixSSEPair{
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
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": []prefixSSEPair{prefixSSEPair{
				Prefix: "myminio1/test2",
				SSE:    sseCommaKey1,
			}}},
			success: true,
		},
		{
			encryptionKey: "myminio1/test2=32byteslongsecret   mustbegiven1",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": []prefixSSEPair{prefixSSEPair{
				Prefix: "myminio1/test2",
				SSE:    sseSpaceKey1,
			}}},
			success: true,
		},
		{
			encryptionKey: "myminio1/test2=32byteslongsecretkeymustbegiven2,myminio1/test1/a=32byteslongsecretkeymustbegiven1",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": []prefixSSEPair{prefixSSEPair{
				Prefix: "myminio1/test1/a",
				SSE:    sseKey2,
			}, prefixSSEPair{
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
