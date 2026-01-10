// Copyright (c) 2015-2022 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import "testing"

// TestStorageClassAllowed tests the storage class filtering logic used by the ls command.
func TestStorageClassAllowed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		storageClass string
		filter       string
		expected     bool
	}{
		{
			name:         "No filter - allow all",
			storageClass: "STANDARD",
			filter:       "",
			expected:     true,
		},
		{
			name:         "No filter with empty storage class",
			storageClass: "",
			filter:       "",
			expected:     true,
		},
		{
			name:         "Wildcard filter - allow all",
			storageClass: "STANDARD",
			filter:       "*",
			expected:     true,
		},
		{
			name:         "Wildcard filter with empty storage class",
			storageClass: "",
			filter:       "*",
			expected:     true,
		},
		{
			name:         "Empty storage class with specific filter - allow",
			storageClass: "",
			filter:       "GLACIER",
			expected:     true,
		},
		{
			name:         "Matching storage class",
			storageClass: "STANDARD",
			filter:       "STANDARD",
			expected:     true,
		},
		{
			name:         "Matching storage class GLACIER",
			storageClass: "GLACIER",
			filter:       "GLACIER",
			expected:     true,
		},
		{
			name:         "Non-matching storage class",
			storageClass: "STANDARD",
			filter:       "GLACIER",
			expected:     false,
		},
		{
			name:         "Non-matching storage class (different filter)",
			storageClass: "GLACIER",
			filter:       "STANDARD",
			expected:     false,
		},
		{
			name:         "Non-existent storage class filtered",
			storageClass: "STANDARD",
			filter:       "DEEP_ARCHIVE",
			expected:     false,
		},
		{
			name:         "Case-sensitive mismatch",
			storageClass: "standard",
			filter:       "STANDARD",
			expected:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			content := &ClientContent{StorageClass: tc.storageClass}
			got := storageClassAllowed(content, tc.filter)

			if got != tc.expected {
				t.Fatalf("storageClassAllowed(StorageClass=%q, filter=%q)=%v; want %v",
					tc.storageClass, tc.filter, got, tc.expected)
			}
		})
	}
}
