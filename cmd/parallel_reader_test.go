// Copyright (c) 2015-2025 MinIO, Inc.
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
// along with this program.  If not, see <http://www.gnu.org/licenses/\>.

package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/minio-go/v7/pkg/encrypt"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio-go/v7/pkg/replication"
)

// mockClient implements Client interface for testing parallel reader
type mockClient struct {
	data          []byte
	size          int64
	getRangeCount atomic.Int64
	failAt        int64
}

func (m *mockClient) Get(ctx context.Context, opts GetOptions) (io.ReadCloser, *ClientContent, *probe.Error) {
	reqNum := m.getRangeCount.Add(1)

	if m.failAt > 0 && reqNum == m.failAt {
		return nil, nil, probe.NewError(errors.New("simulated failure"))
	}

	start := opts.RangeStart
	if start >= m.size {
		return nil, nil, probe.NewError(fmt.Errorf("range start %d exceeds size %d", start, m.size))
	}

	data := m.data[start:]
	reader := io.NopCloser(bytes.NewReader(data))
	content := &ClientContent{Size: m.size}

	return reader, content, nil
}

func (m *mockClient) Stat(ctx context.Context, opts StatOptions) (*ClientContent, *probe.Error) {
	return &ClientContent{Size: m.size}, nil
}
func (m *mockClient) List(ctx context.Context, opts ListOptions) <-chan *ClientContent { return nil }
func (m *mockClient) Put(ctx context.Context, reader io.Reader, size int64, progress io.Reader, opts PutOptions) (int64, *probe.Error) {
	return 0, nil
}
func (m *mockClient) Copy(ctx context.Context, source string, opts CopyOptions, progress io.Reader) *probe.Error {
	return nil
}
func (m *mockClient) GetURL() ClientURL                { return ClientURL{} }
func (m *mockClient) AddUserAgent(app, version string) {}
func (m *mockClient) Remove(ctx context.Context, isIncomplete, isRemoveBucket, isBypass, isForceDel bool, contentCh <-chan *ClientContent) <-chan RemoveResult {
	return nil
}
func (m *mockClient) Select(ctx context.Context, expression string, sse encrypt.ServerSide, opts SelectObjectOpts) (io.ReadCloser, *probe.Error) {
	return nil, nil
}
func (m *mockClient) MakeBucket(ctx context.Context, region string, ignoreExisting, withLock bool) *probe.Error {
	return nil
}
func (m *mockClient) RemoveBucket(ctx context.Context, forceRemove bool) *probe.Error { return nil }
func (m *mockClient) ListBuckets(ctx context.Context) ([]*ClientContent, *probe.Error) {
	return nil, nil
}
func (m *mockClient) SetObjectLockConfig(ctx context.Context, mode minio.RetentionMode, validity uint64, unit minio.ValidityUnit) *probe.Error {
	return nil
}
func (m *mockClient) GetObjectLockConfig(ctx context.Context) (string, minio.RetentionMode, uint64, minio.ValidityUnit, *probe.Error) {
	return "", "", 0, "", nil
}
func (m *mockClient) GetAccess(ctx context.Context) (string, string, *probe.Error) {
	return "", "", nil
}
func (m *mockClient) GetAccessRules(ctx context.Context) (map[string]string, *probe.Error) {
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
func (m *mockClient) GetTags(ctx context.Context, versionID string) (map[string]string, *probe.Error) {
	return nil, nil
}
func (m *mockClient) SetTags(ctx context.Context, versionID, tags string) *probe.Error { return nil }
func (m *mockClient) DeleteTags(ctx context.Context, versionID string) *probe.Error    { return nil }
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
func (m *mockClient) RemoveReplication(ctx context.Context) *probe.Error { return nil }
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
func (m *mockClient) DeleteEncryption(ctx context.Context) *probe.Error { return nil }
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
func (m *mockClient) GetBucketCors(ctx context.Context) (*cors.Config, *probe.Error) { return nil, nil }
func (m *mockClient) SetBucketCors(ctx context.Context, corsXML []byte) *probe.Error { return nil }
func (m *mockClient) DeleteBucketCors(ctx context.Context) *probe.Error              { return nil }

func TestParallelReader_Basic(t *testing.T) {
	testData := []byte("Hello, World!")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 5, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	result, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(result, testData) {
		t.Error("Data mismatch")
	}
}

func TestParallelReader_LargeData(t *testing.T) {
	size := 1024 * 1024
	testData := make([]byte, size)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	client := &mockClient{data: testData, size: int64(size)}
	pr := NewParallelReader(context.Background(), client, client.size, 128*1024, 8, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	result, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(result, testData) {
		t.Error("Data mismatch")
	}
}

func TestParallelReader_SmallBufferReads(t *testing.T) {
	testData := []byte("ABCDEFGHIJKLMNOP")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 5, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	var result bytes.Buffer
	buf := make([]byte, 3)
	for {
		n, err := pr.Read(buf)
		if n > 0 {
			result.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
	}

	if !bytes.Equal(result.Bytes(), testData) {
		t.Error("Data mismatch")
	}
}

func TestParallelReader_EmptyData(t *testing.T) {
	client := &mockClient{data: []byte{}, size: 0}

	pr := NewParallelReader(context.Background(), client, 0, 10, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	result, err := io.ReadAll(pr)
	if err != nil || len(result) != 0 {
		t.Error("Expected empty result")
	}
}

func TestParallelReader_ContextCancellation(t *testing.T) {
	testData := make([]byte, 1000)
	client := &mockClient{data: testData, size: int64(len(testData))}

	ctx, cancel := context.WithCancel(context.Background())
	pr := NewParallelReader(ctx, client, client.size, 100, 4, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	buf := make([]byte, 50)
	if _, err := pr.Read(buf); err != nil {
		t.Fatalf("First read failed: %v", err)
	}

	cancel()

	if _, err := pr.Read(buf); err == nil {
		t.Error("Expected error after cancellation")
	}
}

func TestParallelReader_DownloadError(t *testing.T) {
	testData := make([]byte, 100)
	for i := range testData {
		testData[i] = byte(i)
	}
	client := &mockClient{
		data:   testData,
		size:   int64(len(testData)),
		failAt: 3, // Fail on third request
	}

	pr := NewParallelReader(context.Background(), client, client.size, 25, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	// Should eventually hit the error during reading
	_, err := io.ReadAll(pr)
	if err == nil {
		t.Error("Expected error from download failure")
	}
}

// Test various part sizes
func TestParallelReader_PartSizes(t *testing.T) {
	tests := []struct {
		name         string
		description  string
		testData     []byte
		partSize     int64
		workers      int
		expectedReqs int64
	}{
		{
			name:         "SinglePart",
			description:  "Part size larger than data results in single part",
			testData:     []byte("Single part"),
			partSize:     1000,
			workers:      4,
			expectedReqs: 1,
		},
		{
			name:         "ExactBoundaries",
			description:  "16 bytes with 4-byte parts = exactly 4 parts",
			testData:     []byte("AAAABBBBCCCCDDDD"),
			partSize:     4,
			workers:      4,
			expectedReqs: 4,
		},
		{
			name:         "UnevenBoundaries",
			description:  "23 bytes with 7-byte parts: 3 full parts + 1 partial (2 bytes)",
			testData:     []byte("12345678901234567890123"),
			partSize:     7,
			workers:      3,
			expectedReqs: 4,
		},
		{
			name:         "VerySmallParts",
			description:  "1-byte parts = one part per byte",
			testData:     []byte("Test with 1-byte parts"),
			partSize:     1,
			workers:      4,
			expectedReqs: int64(len("Test with 1-byte parts")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockClient{data: tt.testData, size: int64(len(tt.testData))}

			pr := NewParallelReader(context.Background(), client, client.size, tt.partSize, tt.workers, GetOptions{})
			if err := pr.Start(); err != nil {
				t.Fatalf("Failed to start: %v", err)
			}
			defer pr.Close()

			result, err := io.ReadAll(pr)
			if err != nil {
				t.Fatalf("ReadAll failed: %v", err)
			}
			if !bytes.Equal(result, tt.testData) {
				t.Errorf("Data mismatch:\nwant: %s\ngot:  %s", tt.testData, result)
			}
			if client.getRangeCount.Load() != tt.expectedReqs {
				t.Errorf("Expected %d range requests, got %d", tt.expectedReqs, client.getRangeCount.Load())
			}
		})
	}
}

func TestParallelReader_ReadAfterClose(t *testing.T) {
	testData := []byte("Test data")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 5, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Read some data first
	buf := make([]byte, 5)
	if _, err := pr.Read(buf); err != nil {
		t.Fatalf("First read failed: %v", err)
	}

	// Close the reader
	if err := pr.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Attempt to read after close should fail
	_, err := pr.Read(buf)
	if err == nil {
		t.Error("Expected error when reading after close")
	}
}

func TestParallelReader_CloseWithoutStart(t *testing.T) {
	testData := []byte("Test data")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 5, 2, GetOptions{})

	// Close without calling Start() - should be safe
	if err := pr.Close(); err != nil {
		t.Errorf("Close without start failed: %v", err)
	}
}

func TestParallelReader_MultipleClose(t *testing.T) {
	testData := []byte("Test data")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 5, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}

	// Multiple closes should be idempotent
	for i := range 3 {
		if err := pr.Close(); err != nil {
			t.Errorf("Close #%d failed: %v", i+1, err)
		}
	}
}

func TestParallelReader_PartialRead(t *testing.T) {
	testData := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 10, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	// Read only part of the data
	buf := make([]byte, 10)
	n, err := pr.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 10 {
		t.Errorf("Expected to read 10 bytes, got %d", n)
	}
	if !bytes.Equal(buf, testData[:10]) {
		t.Error("Data mismatch on partial read")
	}
}

func TestParallelReader_LargeBuffer(t *testing.T) {
	testData := []byte("Small data, large buffer")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 8, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	// Use io.ReadAll since a single Read() may return less data same as io.Reader
	result, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(result, testData) {
		t.Errorf("Data mismatch:\nwant: %s\ngot:  %s", testData, result)
	}
}

func TestParallelReader_ReadZeroBytes(t *testing.T) {
	testData := []byte("Test data")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 5, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	// Zero-length buffer read
	buf := make([]byte, 0)
	n, _ := pr.Read(buf)
	if n != 0 {
		t.Errorf("Expected 0 bytes read, got %d", n)
	}
}

func TestParallelReader_ConcurrentReads(t *testing.T) {
	testData := make([]byte, 1000)
	for i := range testData {
		testData[i] = byte(i)
	}
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 100, 4, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	// Note: io.Reader is not safe for concurrent reads.
	// This test verifies basic functionality still works.
	result, err := io.ReadAll(pr)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !bytes.Equal(result, testData) {
		t.Error("Data mismatch")
	}
}

func TestParallelReader_DifferentWorkerCounts(t *testing.T) {
	testData := make([]byte, 5000)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for _, workers := range []int{1, 2, 4, 8, 16, 32} {
		t.Run(fmt.Sprintf("workers=%d", workers), func(t *testing.T) {
			client := &mockClient{data: testData, size: int64(len(testData))}

			pr := NewParallelReader(context.Background(), client, client.size, 500, workers, GetOptions{})
			if err := pr.Start(); err != nil {
				t.Fatalf("Failed to start: %v", err)
			}
			defer pr.Close()

			result, err := io.ReadAll(pr)
			if err != nil {
				t.Fatalf("Read failed: %v", err)
			}
			if !bytes.Equal(result, testData) {
				t.Errorf("Data mismatch with %d workers", workers)
			}
		})
	}
}

func TestParallelReader_ByteByByte(t *testing.T) {
	testData := []byte("ByteByByteRead")
	client := &mockClient{data: testData, size: int64(len(testData))}

	pr := NewParallelReader(context.Background(), client, client.size, 5, 2, GetOptions{})
	if err := pr.Start(); err != nil {
		t.Fatalf("Failed to start: %v", err)
	}
	defer pr.Close()

	// Read one byte at a time
	var result bytes.Buffer
	buf := make([]byte, 1)
	for {
		n, err := pr.Read(buf)
		if n > 0 {
			result.WriteByte(buf[0])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
	}

	if !bytes.Equal(result.Bytes(), testData) {
		t.Errorf("Data mismatch:\nwant: %s\ngot:  %s", testData, result.Bytes())
	}
}
