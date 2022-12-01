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
