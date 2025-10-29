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

import (
	"testing"
)

func TestGetBandwidthInBytes(t *testing.T) {
	type args struct {
		bandwidthStr string
	}
	f1 := 999.1234567 * 1024 * 1024
	f2 := 10.123456789 * 1024 * 1024 * 1024
	f3 := 10000.123456789 * 1024 * 1024 * 1024
	f4 := 0.001 * 1024 * 1024 * 1024
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "1MegaByte",
			args: args{
				bandwidthStr: "1Mi",
			},
			want: 1024 * 1024,
		},
		{
			name: "1MegaBit",
			args: args{
				bandwidthStr: "1M",
			},
			want: 1000000,
		},
		{
			name: "1GigaByte",
			args: args{
				bandwidthStr: "1G",
			},
			want: 1000000000,
		},
		{
			name: "1GibiByte",
			args: args{
				bandwidthStr: "1Gi",
			},
			want: 1024 * 1024 * 1024,
		},
		{
			name: "FractionalMegaBytes",
			args: args{
				bandwidthStr: "999.123456789123456789M",
			},
			want: 999123456,
		},
		{
			name: "FractionalGigaBytes",
			args: args{
				bandwidthStr: "10.123456789123456789123456G",
			},
			want: 10123456789,
		},
		{
			name: "FractionalBigGigaBytes",
			args: args{
				bandwidthStr: "10000.123456789123456789123456G",
			},
			want: 10000123456789,
		},
		{
			name: "FractionalMebiBytes",
			args: args{
				bandwidthStr: "999.123456789123456789Mi",
			},
			want: uint64(f1),
		},
		{
			name: "FractionalGibiBytes",
			args: args{
				bandwidthStr: "10.123456789123456789123456Gi",
			},
			want: uint64(f2),
		},
		{
			name: "FractionalBigGibiBytes",
			args: args{
				bandwidthStr: "10000.123456789123456789123456Gi",
			},
			want: uint64(f3),
		},
		{
			name: "SmallGiga",
			args: args{
				bandwidthStr: "0.001Gi",
			},
			want: uint64(f4),
		},
		{
			name: "LargeK",
			args: args{
				bandwidthStr: "1024Ki",
			},
			want: 1024 * 1024,
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := getBandwidthInBytes(tt.args.bandwidthStr); err != nil || got != tt.want {
				t.Errorf("getBandwidthInBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
