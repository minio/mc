// Copyright (c) 2022 MinIO, Inc.
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

package ilm

import (
	"testing"

	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

func TestILMTags(t *testing.T) {
	tests := []struct {
		rule     lifecycle.Rule
		expected string
	}{
		{
			rule: lifecycle.Rule{
				ID: "one-tag",
				RuleFilter: lifecycle.Filter{
					Tag: lifecycle.Tag{
						Key:   "key1",
						Value: "val1",
					},
				},
			},
			expected: "key1=val1",
		},
		{
			rule: lifecycle.Rule{
				ID: "many-tags",
				RuleFilter: lifecycle.Filter{
					And: lifecycle.And{
						Tags: []lifecycle.Tag{
							{
								Key:   "key1",
								Value: "val1",
							},
							{
								Key:   "key2",
								Value: "val2",
							},
							{
								Key:   "key3",
								Value: "val3",
							},
						},
					},
				},
			},
			expected: "key1=val1&key2=val2&key3=val3",
		},
	}
	for i, test := range tests {
		if got := getTags(test.rule); got != test.expected {
			t.Fatalf("%d: Expected %s but got %s", i+1, test.expected, got)
		}
	}
}
