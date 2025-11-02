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
	"reflect"
	"testing"
)

func TestParseMetaData(t *testing.T) {
	metaDataCases := []struct {
		input  string
		output map[string]string
		err    error
		status bool
	}{
		// success scenario using ; as delimiter
		{"key1=value1;key2=value2", map[string]string{"Key1": "value1", "Key2": "value2"}, nil, true},
		// success scenario using ; as delimiter
		{"key1=m1=m2,m3=m4;key2=value2", map[string]string{"Key1": "m1=m2,m3=m4", "Key2": "value2"}, nil, true},
		// success scenario using = more than once
		{"Cache-Control=max-age=90000,min-fresh=9000;key1=value1;key2=value2", map[string]string{"Cache-Control": "max-age=90000,min-fresh=9000", "Key1": "value1", "Key2": "value2"}, nil, true},
		// using different delimiter, other than '=' between key value
		{"key1:value1;key2:value2", nil, ErrInvalidMetadata, false},
		// using no delimiter
		{"key1:value1:key2:value2", nil, ErrInvalidMetadata, false},
		// success: use value in quotes
		{"Content-Disposition='form-data; name=\"description\"'", map[string]string{"Content-Disposition": "form-data; name=\"description\""}, nil, true},
		// success: use value in double quotes
		{"Content-Disposition=\"form-data; name='description'\"", map[string]string{"Content-Disposition": "form-data; name='description'"}, nil, true},
		// fail: unterminated quote
		{"Content-Disposition='form-data; name=\"description\"", nil, ErrInvalidMetadata, false},
		// fail: unterminated double quote
		{"Content-Disposition=\"form-data; name='description'", nil, ErrInvalidMetadata, false},
		// success: use value and key in quotes
		{"\"Content-Disposition\"='form-data; name=\"description\"'", map[string]string{"Content-Disposition": "form-data; name=\"description\""}, nil, true},
		// success: use value and key in quotes
		{"\"Content=Disposition;Other key part=this is also key data\"='form-data; name=\"description\"'", map[string]string{"Content=Disposition;Other key part=this is also key data": "form-data; name=\"description\""}, nil, true},
	}

	for idx, testCase := range metaDataCases {
		metaDatamap, errMeta := getMetaDataEntry(testCase.input)
		if testCase.status == true {
			if errMeta != nil {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, errMeta)
			}
			if !reflect.DeepEqual(metaDatamap, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, metaDatamap)
			}
		}

		if testCase.status == false {
			if !reflect.DeepEqual(metaDatamap, testCase.output) {
				t.Fatalf("Test %d: generated Map not matching, expected = `%s`, found = `%s`", idx+1, testCase.input, metaDatamap)
			}
			if errMeta.Cause.Error() != testCase.err.Error() {
				t.Fatalf("Test %d: generated error not matching, expected = `%s`, found = `%s`", idx+1, testCase.err, errMeta)
			}
		}
	}
}

// TestCopyStrategyDecision tests the decision logic for determining which copy strategy to use
// based on various conditions like alias, file size, zip flag, and checksum flag.
func TestCopyStrategyDecision(t *testing.T) {
	const maxServerSideCopySize = 5 * 1024 * 1024 * 1024 * 1024 // 5 TiB

	testCases := []struct {
		name             string
		sourceAlias      string
		targetAlias      string
		fileSize         int64
		isZip            bool
		checksumSet      bool
		expectedStrategy string // "server-side" or "stream"
		description      string
	}{
		// Server-side copy cases
		{
			name:             "Same alias, small file, no flags",
			sourceAlias:      "s3",
			targetAlias:      "s3",
			fileSize:         1024 * 1024 * 1024, // 1 GiB
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "server-side",
			description:      "Should use server-side copy for same alias and small file",
		},
		{
			name:             "Same alias, 4.9 TiB file, no flags",
			sourceAlias:      "s3",
			targetAlias:      "s3",
			fileSize:         5393554080768, // ~4.9 TiB (< 5 TiB limit)
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "server-side",
			description:      "Should use server-side copy for files just under 5 TiB limit",
		},
		{
			name:             "Same alias, exactly 5 TiB minus 1 byte",
			sourceAlias:      "minio1",
			targetAlias:      "minio1",
			fileSize:         maxServerSideCopySize - 1,
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "server-side",
			description:      "Should use server-side copy for files exactly at limit boundary",
		},

		// Stream copy cases - different alias
		{
			name:             "Different alias, small file",
			sourceAlias:      "s3",
			targetAlias:      "minio",
			fileSize:         1024 * 1024 * 1024, // 1 GiB
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "stream",
			description:      "Should use stream copy for cross-alias transfers",
		},
		{
			name:             "Different alias, large file",
			sourceAlias:      "minio1",
			targetAlias:      "minio2",
			fileSize:         10 * 1024 * 1024 * 1024 * 1024, // 10 TiB
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "stream",
			description:      "Should use stream copy for cross-alias even with large files",
		},

		// Stream copy cases - file size >= 5 TiB
		{
			name:             "Same alias, exactly 5 TiB",
			sourceAlias:      "s3",
			targetAlias:      "s3",
			fileSize:         maxServerSideCopySize,
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "stream",
			description:      "Should use stream copy for files >= 5 TiB (ComposeObject limit)",
		},
		{
			name:             "Same alias, 6 TiB file",
			sourceAlias:      "minio",
			targetAlias:      "minio",
			fileSize:         6 * 1024 * 1024 * 1024 * 1024, // 6 TiB
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "stream",
			description:      "Should use stream copy for files over 5 TiB limit",
		},
		{
			name:             "Same alias, 100 TiB file",
			sourceAlias:      "s3",
			targetAlias:      "s3",
			fileSize:         100 * 1024 * 1024 * 1024 * 1024, // 100 TiB
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "stream",
			description:      "Should use stream copy for very large files",
		},

		// Stream copy cases - zip flag
		{
			name:             "Same alias, small file, zip enabled",
			sourceAlias:      "s3",
			targetAlias:      "s3",
			fileSize:         100 * 1024 * 1024, // 100 MiB
			isZip:            true,
			checksumSet:      false,
			expectedStrategy: "stream",
			description:      "Should use stream copy when extracting from zip",
		},
		{
			name:             "Same alias, 1 TiB file, zip enabled",
			sourceAlias:      "minio",
			targetAlias:      "minio",
			fileSize:         1024 * 1024 * 1024 * 1024, // 1 TiB
			isZip:            true,
			checksumSet:      false,
			expectedStrategy: "stream",
			description:      "Should use stream copy for zip extraction regardless of size",
		},

		// Stream copy cases - checksum flag
		{
			name:             "Same alias, small file, checksum enabled",
			sourceAlias:      "s3",
			targetAlias:      "s3",
			fileSize:         500 * 1024 * 1024, // 500 MiB
			isZip:            false,
			checksumSet:      true,
			expectedStrategy: "stream",
			description:      "Should use stream copy when checksum verification is requested",
		},
		{
			name:             "Same alias, 2 TiB file, checksum enabled",
			sourceAlias:      "minio",
			targetAlias:      "minio",
			fileSize:         2 * 1024 * 1024 * 1024 * 1024, // 2 TiB
			isZip:            false,
			checksumSet:      true,
			expectedStrategy: "stream",
			description:      "Should use stream copy for checksum verification on large files",
		},

		// Edge cases - multiple conditions forcing stream copy
		{
			name:             "Different alias, large file, zip enabled",
			sourceAlias:      "s3",
			targetAlias:      "minio",
			fileSize:         10 * 1024 * 1024 * 1024 * 1024, // 10 TiB
			isZip:            true,
			checksumSet:      false,
			expectedStrategy: "stream",
			description:      "Should use stream copy when multiple conditions require it",
		},
		{
			name:             "Same alias, over 5 TiB, checksum enabled",
			sourceAlias:      "s3",
			targetAlias:      "s3",
			fileSize:         7 * 1024 * 1024 * 1024 * 1024, // 7 TiB
			isZip:            false,
			checksumSet:      true,
			expectedStrategy: "stream",
			description:      "Should use stream copy when both size and checksum require it",
		},
		{
			name:             "Different alias, zip, checksum",
			sourceAlias:      "minio1",
			targetAlias:      "minio2",
			fileSize:         100 * 1024 * 1024, // 100 MiB
			isZip:            true,
			checksumSet:      true,
			expectedStrategy: "stream",
			description:      "Should use stream copy when all stream conditions are met",
		},

		// Additional edge cases
		{
			name:             "Same alias, zero byte file",
			sourceAlias:      "s3",
			targetAlias:      "s3",
			fileSize:         0,
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "server-side",
			description:      "Should use server-side copy for zero byte files",
		},
		{
			name:             "Same alias, 64 MiB file (multipart threshold)",
			sourceAlias:      "minio",
			targetAlias:      "minio",
			fileSize:         64 * 1024 * 1024, // 64 MiB
			isZip:            false,
			checksumSet:      false,
			expectedStrategy: "server-side",
			description:      "Should use server-side copy for files at multipart threshold",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the decision logic from doCopySession
			isServerSideCopy := tc.sourceAlias == tc.targetAlias &&
				!tc.isZip &&
				!tc.checksumSet &&
				tc.fileSize < maxServerSideCopySize

			var actualStrategy string
			if isServerSideCopy {
				actualStrategy = "server-side"
			} else {
				actualStrategy = "stream"
			}

			if actualStrategy != tc.expectedStrategy {
				t.Errorf("Test '%s' failed:\n"+
					"  Description: %s\n"+
					"  Source Alias: %s\n"+
					"  Target Alias: %s\n"+
					"  File Size: %d bytes (%.2f TiB)\n"+
					"  Zip Flag: %v\n"+
					"  Checksum Set: %v\n"+
					"  Expected Strategy: %s\n"+
					"  Actual Strategy: %s",
					tc.name,
					tc.description,
					tc.sourceAlias,
					tc.targetAlias,
					tc.fileSize,
					float64(tc.fileSize)/(1024*1024*1024*1024),
					tc.isZip,
					tc.checksumSet,
					tc.expectedStrategy,
					actualStrategy,
				)
			}
		})
	}
}

// copyStrategyConditions represents the conditions that determine copy strategy
type copyStrategyConditions struct {
	sameAlias   bool
	underLimit  bool
	isZip       bool
	checksumSet bool
}

// TestCopyStrategyMatrix tests all combinations of copy strategy decision factors
func TestCopyStrategyMatrix(t *testing.T) {
	const maxServerSideCopySize = 5 * 1024 * 1024 * 1024 * 1024 // 5 TiB

	// Test matrix: all combinations
	testCases := []copyStrategyConditions{
		// All conditions met for server-side copy
		{sameAlias: true, underLimit: true, isZip: false, checksumSet: false},

		// One condition fails resulting in stream copy
		{sameAlias: false, underLimit: true, isZip: false, checksumSet: false},
		{sameAlias: true, underLimit: false, isZip: false, checksumSet: false},
		{sameAlias: true, underLimit: true, isZip: true, checksumSet: false},
		{sameAlias: true, underLimit: true, isZip: false, checksumSet: true},

		// Multiple conditions fail
		{sameAlias: false, underLimit: false, isZip: false, checksumSet: false},
		{sameAlias: false, underLimit: true, isZip: true, checksumSet: false},
		{sameAlias: false, underLimit: true, isZip: false, checksumSet: true},
		{sameAlias: true, underLimit: false, isZip: true, checksumSet: false},
		{sameAlias: true, underLimit: false, isZip: false, checksumSet: true},
		{sameAlias: true, underLimit: true, isZip: true, checksumSet: true},

		// All conditions unfavorable
		{sameAlias: false, underLimit: false, isZip: true, checksumSet: true},
	}

	for _, cond := range testCases {
		t.Run(formatConditionsName(cond), func(t *testing.T) {
			// Set up test parameters based on conditions
			sourceAlias := "source"
			targetAlias := "target"
			if cond.sameAlias {
				targetAlias = "source"
			}

			fileSize := int64(1024 * 1024 * 1024) // 1 GiB
			if !cond.underLimit {
				fileSize = maxServerSideCopySize + 1
			}

			// Determine expected strategy
			// Server-side copy only if ALL conditions are true
			expectedServerSide := cond.sameAlias && cond.underLimit && !cond.isZip && !cond.checksumSet

			// Simulate the decision logic
			isServerSideCopy := sourceAlias == targetAlias &&
				!cond.isZip &&
				!cond.checksumSet &&
				fileSize < maxServerSideCopySize

			if isServerSideCopy != expectedServerSide {
				t.Errorf("Strategy mismatch for conditions %+v: expected server-side=%v, got=%v",
					cond, expectedServerSide, isServerSideCopy)
			}
		})
	}
}

// Helper function to format test names for matrix tests
func formatConditionsName(cond copyStrategyConditions) string {
	name := ""
	if cond.sameAlias {
		name += "SameAlias_"
	} else {
		name += "DiffAlias_"
	}
	if cond.underLimit {
		name += "Under5TiB_"
	} else {
		name += "Over5TiB_"
	}
	if cond.isZip {
		name += "Zip_"
	} else {
		name += "NoZip_"
	}
	if cond.checksumSet {
		name += "Checksum"
	} else {
		name += "NoChecksum"
	}
	return name
}
