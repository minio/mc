/*
 * MinIO Client (C) 2016 MinIO, Inc.
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
func TestValidHostURL(t *testing.T) {
	testCases := []struct {
		hostURL string
		isHost  bool
	}{
		{
			hostURL: "https://localhost:9000",
			isHost:  true,
		},
		{
			hostURL: "/",
			isHost:  false,
		},
	}

	for _, testCase := range testCases {
		isHost := isValidHostURL(testCase.hostURL)
		if testCase.isHost != isHost {
			t.Fatalf("Expected %t, got %t", testCase.isHost, isHost)
		}
	}
}

func TestIsValidAPI(t *testing.T) {
	equalAssert(isValidAPI("s3V2"), true, t)
	equalAssert(isValidAPI("S3v2"), true, t)
	equalAssert(isValidAPI("s3"), false, t)
}

func equalAssert(ok1, ok2 bool, t *testing.T) {
	if ok1 != ok2 {
		t.Errorf("Expected %t, got %t", ok2, ok1)
	}
}

// Tests valid and invalid secret keys.
func TestValidSecretKeys(t *testing.T) {
	equalAssert(isValidSecretKey("aaa"), false, t)

	equalAssert(isValidSecretKey(""), true, t)
	equalAssert(isValidSecretKey("password"), true, t)
	equalAssert(isValidSecretKey("password%%"), true, t)
	equalAssert(isValidSecretKey("BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF"), true, t)
}

// Tests valid and invalid access keys.
func TestValidAccessKeys(t *testing.T) {
	equalAssert(isValidAccessKey("aa"), false, t)

	equalAssert(isValidAccessKey(""), true, t)
	equalAssert(isValidAccessKey("adm"), true, t)
	equalAssert(isValidAccessKey("admin"), true, t)
	equalAssert(isValidAccessKey("$$%%%%%3333"), true, t)
	equalAssert(isValidAccessKey("c67W2-r4MAyAYScRl"), true, t)
	equalAssert(isValidAccessKey("EXOb76bfeb1234562iu679f11588"), true, t)
	equalAssert(isValidAccessKey("BYvgJM101sHngl2uzjXS/OBF/aMxAN06JrJ3qJlF"), true, t)
}
