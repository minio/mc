/*
 * MinIO Client (C) 2018 MinIO, Inc.
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

import "testing"

// Tests valid host URL functionality.
func TestParseEnvURLStr(t *testing.T) {
	testCases := []struct {
		hostURL   string
		accessKey string
		secretKey string
		hostname  string
		port      string
	}{
		{
			hostURL:   "https://minio:minio1#23@localhost:9000",
			accessKey: "minio",
			secretKey: "minio1#23",
			hostname:  "localhost",
			port:      "9000",
		},
		{
			hostURL:   "https://minio:minio123@localhost:9000",
			accessKey: "minio",
			secretKey: "minio123",
			hostname:  "localhost",
			port:      "9000",
		},
		{
			hostURL:   "https://localhost:9000",
			accessKey: "",
			secretKey: "",
			hostname:  "localhost",
			port:      "9000",
		},
	}

	for i, testCase := range testCases {
		url, ak, sk, err := parseEnvURLStr(testCase.hostURL)
		if testCase.accessKey != ak {
			t.Fatalf("Test %d: Expected %s, got %s", i+1, testCase.accessKey, ak)
		}
		if testCase.secretKey != sk {
			t.Fatalf("Test %d: Expected %s, got %s", i+1, testCase.secretKey, sk)
		}
		if testCase.hostname != url.Hostname() {
			t.Fatalf("Test %d: Expected %s, got %s", i+1, testCase.hostname, url.Hostname())
		}
		if testCase.port != url.Port() {
			t.Fatalf("Test %d: Expected %s, got %s", i+1, testCase.port, url.Port())
		}
		if err != nil {
			t.Fatalf("Test %d: Expected test to pass. Failed with err %s", i+1, err)
		}
	}
}

func TestParseEnvURLStrInvalid(t *testing.T) {
	_, _, _, err := parseEnvURLStr("")
	if err == nil {
		t.Fatalf("Expected failure")
	}
}
