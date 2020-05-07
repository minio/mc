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
		hostURL      string
		accessKey    string
		secretKey    string
		sessionToken string
		hostname     string
		port         string
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
		{
			hostURL:      "https://minio:minio123:token@localhost:9000",
			accessKey:    "minio",
			secretKey:    "minio123",
			sessionToken: "token",
			hostname:     "localhost",
			port:         "9000",
		},
		{
			hostURL:   "https://minio@localhost:9000",
			accessKey: "minio",
			hostname:  "localhost",
			port:      "9000",
		},
		{
			hostURL:   "https://minio:@localhost:9000",
			accessKey: "minio",
			hostname:  "localhost",
			port:      "9000",
		},
		{
			hostURL:  "https://:@localhost:9000",
			hostname: "localhost",
			port:     "9000",
		},
		{
			hostURL:   "https://:minio123@localhost:9000",
			hostname:  "localhost",
			secretKey: "minio123",
			port:      "9000",
		},
		{
			hostURL:      "https://:minio123:token@localhost:9000",
			hostname:     "localhost",
			secretKey:    "minio123",
			sessionToken: "token",
			port:         "9000",
		},
		{
			hostURL:   "https://:minio123:@localhost:9000",
			hostname:  "localhost",
			secretKey: "minio123",
			port:      "9000",
		},
	}

	for _, testCase := range testCases {
		t.Run("", func(t *testing.T) {
			url, ak, sk, token, err := parseEnvURLStr(testCase.hostURL)
			if testCase.accessKey != ak {
				t.Fatalf("Expected %s, got %s", testCase.accessKey, ak)
			}
			if testCase.secretKey != sk {
				t.Fatalf("Expected %s, got %s", testCase.secretKey, sk)
			}
			if testCase.sessionToken != token {
				t.Fatalf("Expected %s, got %s", testCase.sessionToken, token)
			}
			if testCase.hostname != url.Hostname() {
				t.Fatalf("Expected %s, got %s", testCase.hostname, url.Hostname())
			}
			if testCase.port != url.Port() {
				t.Fatalf("Expected %s, got %s", testCase.port, url.Port())
			}
			if err != nil {
				t.Fatalf("Expected test to pass. Failed with err %s", err)
			}
		})
	}
}

func TestParseEnvURLStrInvalid(t *testing.T) {
	_, _, _, _, err := parseEnvURLStr("")
	if err == nil {
		t.Fatalf("Expected failure")
	}
}
