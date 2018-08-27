/*
 * Minio Client (C) 2018 Minio, Inc.
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
func TestparseEnvURLStr(t *testing.T) {
	testCases := []struct {
		hostURL   string
		accessKey string
		secretKey string
		url       string
	}{
		{
			hostURL:   "https://minio:minio1#23@localhost:9000",
			accessKey: "minio",
			secretKey: "minio#123",
			url:       "https://localhost:9000",
		},
		{
			hostURL:   "https://minio:minio123@localhost:9000",
			accessKey: "minio",
			secretKey: "minio123",
			url:       "https://localhost:9000",
		},
		{
			hostURL:   "http://minio:minio1#23@localhost:9000",
			accessKey: "minio",
			secretKey: "minio#123",
			url:       "http://localhost:9000",
		},
		{
			hostURL:   "https://localhost:9000",
			accessKey: "",
			secretKey: "",
			url:       "https://localhost:9000",
		},
	}

	for _, testCase := range testCases {
		url, ak, sk, err := parseEnvURLStr(testCase.hostURL)
		if testCase.accessKey != sk {
			t.Fatalf("Expected %s, got %s", testCase.accessKey, ak)
		}
		if testCase.secretKey != sk {
			t.Fatalf("Expected %s, got %s", testCase.secretKey, sk)
		}
		if testCase.url != url.Hostname() {
			t.Fatalf("Expected %s, got %s", testCase.url, url.Hostname())
		}
		if err != nil {
			t.Fatalf("Expected test to pass. Failed with err %s", err)
		}
	}
}
