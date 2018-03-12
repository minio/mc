/*
 * Minio Client (C) 2017 Minio, Inc.
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

func TestParseURLEnv(t *testing.T) {
	testCases := []struct {
		envURL         string
		expectedURL    string
		expectedAccess string
		expectedSecret string
		success        bool
	}{
		{
			envURL:         "https://username:password@play.minio.io:9000/",
			expectedURL:    "https://play.minio.io:9000/",
			expectedAccess: "username",
			expectedSecret: "password",
			success:        true,
		},
		{
			envURL:      "https://play.minio.io:9000/",
			expectedURL: "https://play.minio.io:9000/",
			success:     true,
		},
		{
			envURL:  "ftp://play.minio.io:9000/",
			success: false,
		},
		{
			envURL:  "",
			success: false,
		},
		{
			envURL:  "https://play.minio.io:9000/path",
			success: false,
		},
		{
			envURL:  "https://play.minio.io:9000/?path=value",
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
	testCases := []struct {
		encryptionKey  string
		expectedEncMap map[string][]prefixSSEPair
		success        bool
	}{
		{
			encryptionKey:  "myminio1/test2=32byteslongsecretkeymustbegiven2",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": []prefixSSEPair{prefixSSEPair{prefix: "myminio1/test2", sseKey: "32byteslongsecretkeymustbegiven2"}}},
			success:        true,
		},
		{
			encryptionKey:  "myminio1/test2=32byteslongsecretkeymustbegiven",
			expectedEncMap: nil,
			success:        false,
		},
		{
			encryptionKey:  "myminio1/test2=32byteslongsecretkey,ustbegiven1",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": []prefixSSEPair{prefixSSEPair{prefix: "myminio1/test2", sseKey: "32byteslongsecretkey,ustbegiven1"}}},
			success:        true,
		},
		{
			encryptionKey:  "myminio1/test2=32byteslongsecret   mustbegiven1",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": []prefixSSEPair{prefixSSEPair{prefix: "myminio1/test2", sseKey: "32byteslongsecret   mustbegiven1"}}},
			success:        true,
		},
		{
			encryptionKey:  "myminio1/test2=32byteslongsecretkeymustbegiven2,myminio1/test1/a=32byteslongsecretkeymustbegiven1",
			expectedEncMap: map[string][]prefixSSEPair{"myminio1": []prefixSSEPair{prefixSSEPair{prefix: "myminio1/test1/a", sseKey: "32byteslongsecretkeymustbegiven1"}, prefixSSEPair{prefix: "myminio1/test2", sseKey: "32byteslongsecretkeymustbegiven2"}}},
			success:        true,
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
