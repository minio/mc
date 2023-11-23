// Copyright (c) 2015-2023 MinIO, Inc.
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
	"fmt"
	"testing"

	"github.com/dustin/go-humanize"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

func TestOptionFilter(t *testing.T) {
	emptyFilter := lifecycle.Filter{}
	emptyOpts := LifecycleOptions{}

	filterWithPrefix := lifecycle.Filter{
		Prefix: "doc/",
	}
	optsWithPrefix := LifecycleOptions{
		Prefix: strPtr("doc/"),
	}

	filterWithTag := lifecycle.Filter{
		Tag: lifecycle.Tag{
			Key:   "key1",
			Value: "value1",
		},
	}
	optsWithTag := LifecycleOptions{
		Tags: strPtr("key1=value1"),
	}

	filterWithSzLt := lifecycle.Filter{
		ObjectSizeLessThan: 100 * humanize.MiByte,
	}
	optsWithSzLt := LifecycleOptions{
		ObjectSizeLessThan: int64Ptr(100 * humanize.MiByte),
	}

	filterWithSzGt := lifecycle.Filter{
		ObjectSizeGreaterThan: 1 * humanize.MiByte,
	}
	optsWithSzGt := LifecycleOptions{
		ObjectSizeGreaterThan: int64Ptr(1 * humanize.MiByte),
	}

	filterWithAnd := lifecycle.Filter{
		And: lifecycle.And{
			Prefix: "doc/",
			Tags: []lifecycle.Tag{
				{
					Key:   "key1",
					Value: "value1",
				},
			},
			ObjectSizeLessThan:    100 * humanize.MiByte,
			ObjectSizeGreaterThan: 1 * humanize.MiByte,
		},
	}
	optsWithAnd := LifecycleOptions{
		Prefix:                strPtr("doc/"),
		Tags:                  strPtr("key1=value1"),
		ObjectSizeLessThan:    int64Ptr(100 * humanize.MiByte),
		ObjectSizeGreaterThan: int64Ptr(1 * humanize.MiByte),
	}

	tests := []struct {
		opts LifecycleOptions
		want lifecycle.Filter
	}{
		{
			opts: emptyOpts,
			want: emptyFilter,
		},
		{
			opts: optsWithPrefix,
			want: filterWithPrefix,
		},
		{
			opts: optsWithTag,
			want: filterWithTag,
		},
		{
			opts: optsWithSzGt,
			want: filterWithSzGt,
		},
		{
			opts: optsWithSzLt,
			want: filterWithSzLt,
		},
		{
			opts: optsWithAnd,
			want: filterWithAnd,
		},
	}

	filterEq := func(a, b lifecycle.Filter) bool {
		if a.ObjectSizeGreaterThan != b.ObjectSizeGreaterThan {
			return false
		}
		if a.ObjectSizeLessThan != b.ObjectSizeLessThan {
			return false
		}
		if a.Prefix != b.Prefix {
			return false
		}
		if a.Tag != b.Tag {
			return false
		}

		if a.And.ObjectSizeGreaterThan != b.And.ObjectSizeGreaterThan {
			return false
		}
		if a.And.ObjectSizeLessThan != b.And.ObjectSizeLessThan {
			return false
		}
		if a.And.Prefix != b.And.Prefix {
			return false
		}
		if len(a.And.Tags) != len(b.And.Tags) {
			return false
		}
		for i := range a.And.Tags {
			if a.And.Tags[i] != b.And.Tags[i] {
				return false
			}
		}

		return true
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test %d", i+1), func(t *testing.T) {
			if got := test.opts.Filter(); !filterEq(got, test.want) {
				t.Fatalf("Expected %#v but got %#v", test.want, got)
			}
		})
	}
}
