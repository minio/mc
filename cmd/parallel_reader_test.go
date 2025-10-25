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
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/replication"
	checkv1 "gopkg.in/check.v1"
)

// mockClient implements Client interface for testing parallel reader
type mockClient struct {
	data []byte
	size int64
}

func (m *mockClient) Get(ctx context.Context, opts GetOptions) (io.ReadCloser, *ClientContent, *probe.Error) {
	start := opts.RangeStart
	end := m.size

	if start >= m.size {
		return nil, nil, probe.NewError(fmt.Errorf("range start %d exceeds size %d", start, m.size))
	}

	// Return data from start position to end
	data := m.data[start:end]
	reader := io.NopCloser(bytes.NewReader(data))

	content := &ClientContent{
		Size: m.size,
	}

	return reader, content, nil
}

// Implement required interface methods as no-ops for testing
func (m *mockClient) Stat(ctx context.Context, opts StatOptions) (*ClientContent, *probe.Error) {
	return &ClientContent{Size: m.size}, nil
}

func (m *mockClient) List(ctx context.Context, opts ListOptions) <-chan *ClientContent {
	return nil
}

func (m *mockClient) Put(ctx context.Context, reader io.Reader, size int64, progress io.Reader, opts PutOptions) (int64, *probe.Error) {
	return 0, nil
}

func (m *mockClient) Copy(ctx context.Context, source string, opts CopyOptions, progress io.Reader) *probe.Error {
	return nil
}

func (m *mockClient) GetURL() ClientURL {
	return ClientURL{}
}

func (m *mockClient) AddUserAgent(app, version string) {}

func (m *mockClient) Select(ctx context.Context, expression string, sse encrypt.ServerSide, opts SelectObjectOpts) (io.ReadCloser, *probe.Error) {
	return nil, nil
}

func (m *mockClient) MakeBucket(ctx context.Context, region string, ignoreExisting, withLock bool) *probe.Error {
	return nil
}

func (m *mockClient) RemoveBucket(ctx context.Context, forceRemove bool) *probe.Error {
	return nil
}

func (m *mockClient) ListBuckets(ctx context.Context) ([]*ClientContent, *probe.Error) {
	return nil, nil
}

func (m *mockClient) SetObjectLockConfig(ctx context.Context, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit) *probe.Error {
	return nil
}

func (m *mockClient) GetObjectLockConfig(ctx context.Context) (status string, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit, perr *probe.Error) {
	return "", "", 0, "", nil
}

func (m *mockClient) GetAccess(ctx context.Context) (access, policyJSON string, err *probe.Error) {
	return "", "", nil
}

func (m *mockClient) GetAccessRules(ctx context.Context) (policyRules map[string]string, err *probe.Error) {
	return nil, nil
}

func (m *mockClient) SetAccess(ctx context.Context, access string, isJSON bool) *probe.Error {
	return nil
}

func (m *mockClient) PutObjectRetention(ctx context.Context, versionID string, mode minio.RetentionMode, retainUntilDate time.Time, bypassGovernance bool) *probe.Error {
	return nil
}

func (m *mockClient) GetObjectRetention(ctx context.Context, versionID string) (minio.RetentionMode, time.Time, *probe.Error) {
	return "", time.Time{}, nil
}

func (m *mockClient) PutObjectLegalHold(ctx context.Context, versionID string, hold minio.LegalHoldStatus) *probe.Error {
	return nil
}

func (m *mockClient) GetObjectLegalHold(ctx context.Context, versionID string) (minio.LegalHoldStatus, *probe.Error) {
	return "", nil
}

func (m *mockClient) ShareDownload(ctx context.Context, versionID string, expires time.Duration) (string, *probe.Error) {
	return "", nil
}

func (m *mockClient) ShareUpload(ctx context.Context, isRecursive bool, expires time.Duration, contentType string) (string, map[string]string, *probe.Error) {
	return "", nil, nil
}

func (m *mockClient) Watch(ctx context.Context, options WatchOptions) (*WatchObject, *probe.Error) {
	return nil, nil
}

func (m *mockClient) Remove(ctx context.Context, isIncomplete, isRemoveBucket, isBypass, isForceDel bool, contentCh <-chan *ClientContent) <-chan RemoveResult {
	return nil
}

func (m *mockClient) GetTags(ctx context.Context, versionID string) (map[string]string, *probe.Error) {
	return nil, nil
}

func (m *mockClient) SetTags(ctx context.Context, versionID, tags string) *probe.Error {
	return nil
}

func (m *mockClient) DeleteTags(ctx context.Context, versionID string) *probe.Error {
	return nil
}

func (m *mockClient) GetLifecycle(ctx context.Context) (*lifecycle.Configuration, time.Time, *probe.Error) {
	return nil, time.Time{}, nil
}

func (m *mockClient) SetLifecycle(ctx context.Context, config *lifecycle.Configuration) *probe.Error {
	return nil
}

func (m *mockClient) GetVersion(ctx context.Context) (minio.BucketVersioningConfiguration, *probe.Error) {
	return minio.BucketVersioningConfiguration{}, nil
}

func (m *mockClient) SetVersion(ctx context.Context, status string, prefixes []string, excludeFolders bool) *probe.Error {
	return nil
}

func (m *mockClient) GetReplication(ctx context.Context) (replication.Config, *probe.Error) {
	return replication.Config{}, nil
}

func (m *mockClient) SetReplication(ctx context.Context, cfg *replication.Config, opts replication.Options) *probe.Error {
	return nil
}

func (m *mockClient) RemoveReplication(ctx context.Context) *probe.Error {
	return nil
}

func (m *mockClient) GetReplicationMetrics(ctx context.Context) (replication.MetricsV2, *probe.Error) {
	return replication.MetricsV2{}, nil
}

func (m *mockClient) ResetReplication(ctx context.Context, before time.Duration, arn string) (replication.ResyncTargetsInfo, *probe.Error) {
	return replication.ResyncTargetsInfo{}, nil
}

func (m *mockClient) ReplicationResyncStatus(ctx context.Context, arn string) (replication.ResyncTargetsInfo, *probe.Error) {
	return replication.ResyncTargetsInfo{}, nil
}

func (m *mockClient) GetEncryption(ctx context.Context) (string, string, *probe.Error) {
	return "", "", nil
}

func (m *mockClient) SetEncryption(ctx context.Context, algorithm, kmsKeyID string) *probe.Error {
	return nil
}

func (m *mockClient) DeleteEncryption(ctx context.Context) *probe.Error {
	return nil
}

func (m *mockClient) GetBucketInfo(ctx context.Context) (BucketInfo, *probe.Error) {
	return BucketInfo{}, nil
}

func (m *mockClient) Restore(ctx context.Context, versionID string, days int) *probe.Error {
	return nil
}

func (m *mockClient) GetPart(ctx context.Context, part int) (io.ReadCloser, *probe.Error) {
	return nil, nil
}

func (m *mockClient) PutPart(ctx context.Context, reader io.Reader, size int64, progress io.Reader, opts PutOptions) (int64, *probe.Error) {
	return 0, nil
}

func (m *mockClient) GetBucketCors(ctx context.Context) (*cors.Config, *probe.Error) {
	return nil, nil
}

func (m *mockClient) SetBucketCors(ctx context.Context, corsXML []byte) *probe.Error {
	return nil
}

func (m *mockClient) DeleteBucketCors(ctx context.Context) *probe.Error {
	return nil
}

// Test parallel reader with small data
func (s *TestSuite) TestParallelReaderSmall(c *checkv1.C) {
	ctx := context.Background()

	// Create test data
	testData := []byte("Hello, World! This is a test of parallel reading.")
	client := &mockClient{
		data: testData,
		size: int64(len(testData)),
	}

	// Create parallel reader with 2 threads and 10-byte parts
	pr := NewParallelReader(ctx, client, int64(len(testData)), 10, 2, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)
	defer pr.Close()

	// Read all data
	result, readErr := io.ReadAll(pr)
	c.Assert(readErr, checkv1.IsNil)
	c.Assert(bytes.Equal(result, testData), checkv1.Equals, true)
}

// Test parallel reader with larger data and more threads
func (s *TestSuite) TestParallelReaderLarge(c *checkv1.C) {
	ctx := context.Background()

	// Create larger test data (1MB)
	size := 1024 * 1024
	testData := make([]byte, size)
	for i := 0; i < size; i++ {
		testData[i] = byte(i % 256)
	}

	client := &mockClient{
		data: testData,
		size: int64(len(testData)),
	}

	// Create parallel reader with 8 threads and 128KB parts
	partSize := int64(128 * 1024)
	pr := NewParallelReader(ctx, client, int64(len(testData)), partSize, 8, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)
	defer pr.Close()

	// Read all data
	result, readErr := io.ReadAll(pr)
	c.Assert(readErr, checkv1.IsNil)
	c.Assert(len(result), checkv1.Equals, len(testData))
	c.Assert(bytes.Equal(result, testData), checkv1.Equals, true)
}

// Test parallel reader with single thread (should behave like normal reader)
func (s *TestSuite) TestParallelReaderSingleThread(c *checkv1.C) {
	ctx := context.Background()

	testData := []byte("Single threaded test data")
	client := &mockClient{
		data: testData,
		size: int64(len(testData)),
	}

	// Create parallel reader with 1 thread
	pr := NewParallelReader(ctx, client, int64(len(testData)), 5, 1, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)
	defer pr.Close()

	// Read all data
	result, readErr := io.ReadAll(pr)
	c.Assert(readErr, checkv1.IsNil)
	c.Assert(bytes.Equal(result, testData), checkv1.Equals, true)
}

// Test parallel reader with exact part boundaries
func (s *TestSuite) TestParallelReaderExactParts(c *checkv1.C) {
	ctx := context.Background()

	// Create data that divides evenly into parts
	testData := []byte("AAAABBBBCCCCDDDD") // 16 bytes
	client := &mockClient{
		data: testData,
		size: int64(len(testData)),
	}

	// Create parallel reader with 4-byte parts (exactly 4 parts)
	pr := NewParallelReader(ctx, client, int64(len(testData)), 4, 4, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)
	defer pr.Close()

	// Read all data
	result, readErr := io.ReadAll(pr)
	c.Assert(readErr, checkv1.IsNil)
	c.Assert(bytes.Equal(result, testData), checkv1.Equals, true)
}

// Test parallel reader with uneven part boundaries
func (s *TestSuite) TestParallelReaderUnevenParts(c *checkv1.C) {
	ctx := context.Background()

	// Create data that doesn't divide evenly
	testData := []byte("12345678901234567") // 17 bytes
	client := &mockClient{
		data: testData,
		size: int64(len(testData)),
	}

	// Create parallel reader with 5-byte parts (3 full parts + 1 partial)
	pr := NewParallelReader(ctx, client, int64(len(testData)), 5, 4, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)
	defer pr.Close()

	// Read all data
	result, readErr := io.ReadAll(pr)
	c.Assert(readErr, checkv1.IsNil)
	c.Assert(bytes.Equal(result, testData), checkv1.Equals, true)
}

// Test parallel reader with small buffer reads
func (s *TestSuite) TestParallelReaderSmallReads(c *checkv1.C) {
	ctx := context.Background()

	testData := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	client := &mockClient{
		data: testData,
		size: int64(len(testData)),
	}

	pr := NewParallelReader(ctx, client, int64(len(testData)), 5, 2, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)
	defer pr.Close()

	// Read data in small 3-byte chunks
	var result bytes.Buffer
	buf := make([]byte, 3)
	for {
		n, readErr := pr.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if readErr == io.EOF {
			break
		}
		c.Assert(readErr, checkv1.IsNil)
	}

	c.Assert(bytes.Equal(result.Bytes(), testData), checkv1.Equals, true)
}

// Test parallel reader closes properly
func (s *TestSuite) TestParallelReaderClose(c *checkv1.C) {
	ctx := context.Background()

	testData := []byte("Test data for close")
	client := &mockClient{
		data: testData,
		size: int64(len(testData)),
	}

	pr := NewParallelReader(ctx, client, int64(len(testData)), 5, 2, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)

	// Close immediately without reading
	closeErr := pr.Close()
	c.Assert(closeErr, checkv1.IsNil)

	// Verify we can close multiple times
	closeErr = pr.Close()
	c.Assert(closeErr, checkv1.IsNil)
}

// Test parallel reader with large part size (larger than total size)
func (s *TestSuite) TestParallelReaderLargePartSize(c *checkv1.C) {
	ctx := context.Background()

	testData := []byte("Small data")
	client := &mockClient{
		data: testData,
		size: int64(len(testData)),
	}

	// Part size larger than data size (should result in 1 part)
	pr := NewParallelReader(ctx, client, int64(len(testData)), 1000, 4, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)
	defer pr.Close()

	result, readErr := io.ReadAll(pr)
	c.Assert(readErr, checkv1.IsNil)
	c.Assert(bytes.Equal(result, testData), checkv1.Equals, true)
}

// Test parallel reader with empty data
func (s *TestSuite) TestParallelReaderEmpty(c *checkv1.C) {
	ctx := context.Background()

	testData := []byte{}
	client := &mockClient{
		data: testData,
		size: 0,
	}

	pr := NewParallelReader(ctx, client, 0, 10, 2, GetOptions{})
	err := pr.Start()
	c.Assert(err, checkv1.IsNil)
	defer pr.Close()

	result, readErr := io.ReadAll(pr)
	c.Assert(readErr, checkv1.IsNil)
	c.Assert(len(result), checkv1.Equals, 0)
}
