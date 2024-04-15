// Copyright (c) 2015-2024 MinIO, Inc.
//
// # This file is part of MinIO Object Storage stack
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
	"fmt"
	"testing"
)

func TestParseEncryptionKeys(t *testing.T) {
	baseAlias := "mintest"
	basePrefix := "two/layer/prefix"
	baseObject := "object_name"
	sseKey := "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	sseKeyPlain := "01234567890123456789012345678900"

	// INVALID KEYS
	sseKeyInvalidShort := "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2"
	sseKeyInvalidSymbols := "MDEyMzQ1Njc4O____jM0N!!!ODkwMTIzNDU2Nzg5MDA"
	sseKeyInvalidSpaces := "MDE   yMzQ1Njc4OTAxM   jM0NTY3ODkwMTIzNDU2Nzg5MDA"
	sseKeyInvalidPrefixSpace := "     MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDA"
	sseKeyInvalidOneShort := "MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MD"

	testCases := []struct {
		encryptionKey string
		keyPlain      string
		alias         string
		prefix        string
		object        string
		sseType       sseKeyType
		success       bool
	}{
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s=%s", baseAlias, basePrefix, baseObject, sseKey),
			keyPlain:      sseKeyPlain,
			alias:         baseAlias,
			prefix:        basePrefix,
			object:        baseObject,
			success:       true,
		},
		{
			encryptionKey: fmt.Sprintf("%s=/%s=/%s==%s", baseAlias, basePrefix, baseObject, sseKey),
			keyPlain:      sseKeyPlain,
			alias:         baseAlias + "=",
			prefix:        basePrefix + "=",
			object:        baseObject + "=",
			success:       true,
		},
		{
			encryptionKey: fmt.Sprintf("%s//%s//%s/=%s", baseAlias, basePrefix, baseObject, sseKey),
			keyPlain:      sseKeyPlain,
			alias:         baseAlias + "/",
			prefix:        basePrefix + "/",
			object:        baseObject + "/",
			success:       true,
		},
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s==%s", baseAlias, basePrefix, baseObject, sseKey),
			keyPlain:      sseKeyPlain,
			alias:         baseAlias,
			prefix:        basePrefix,
			object:        baseObject + "=",
			success:       true,
		},
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s!@_==_$^&*=%s", baseAlias, basePrefix, baseObject, sseKey),
			keyPlain:      sseKeyPlain,
			alias:         baseAlias,
			prefix:        basePrefix,
			object:        baseObject + "!@_==_$^&*",
			success:       true,
		},
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s=%sXXXXX", baseAlias, basePrefix, baseObject, sseKey),
			success:       false,
		},
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s=%s", baseAlias, basePrefix, baseObject, sseKeyInvalidShort),
			success:       false,
		},
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s=%s", baseAlias, basePrefix, baseObject, sseKeyInvalidSymbols),
			success:       false,
		},
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s=%s", baseAlias, basePrefix, baseObject, sseKeyInvalidSpaces),
			success:       false,
		},
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s=%s", baseAlias, basePrefix, baseObject, sseKeyInvalidPrefixSpace),
			success:       false,
		},
		{
			encryptionKey: fmt.Sprintf("%s/%s/%s==%s", baseAlias, basePrefix, baseObject, sseKeyInvalidOneShort),
			success:       false,
		},
	}

	for i, tc := range testCases {
		alias, prefix, key, err := parseSSEKey(tc.encryptionKey, sseC)
		if tc.success {
			if err != nil {
				t.Fatalf("Test %d: Expected success, got %s", i+1, err)
			}
			if fmt.Sprintf("%s/%s", alias, prefix) != fmt.Sprintf("%s/%s/%s", tc.alias, tc.prefix, tc.object) {
				t.Fatalf("Test %d: alias and prefix parsing was invalid, expected %s/%s/%s, got %s/%s", i, tc.alias, tc.prefix, tc.object, alias, prefix)
			}
			if key != tc.keyPlain {
				t.Fatalf("Test %d: sse key parsing is invalid, expected %s, got %s", i, tc.keyPlain, key)
			}
		}

		if !tc.success {
			if err == nil {
				t.Fatalf("Test %d: Expected error, got success", i+1)
			}
		}
	}
}
