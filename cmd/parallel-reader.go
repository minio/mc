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
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"fmt"
	"io"
	"sync"
)

const (
	// workerQueueDepth controls channel buffer sizing relative to parallelism.
	// Prevents workers from blocking and keeps the download pipeline full while using
	// minimal memory (only part numbers and pointers, not actual data buffers).
	workerQueueDepth = 2
)

// parallelReader reads an object in parallel using range requests
type parallelReader struct {
	ctx         context.Context
	cancelCause context.CancelCauseFunc
	client      Client
	size        int64
	partSize    int64
	parallelism int
	opts        GetOptions

	// State management
	currentPart   int64
	totalParts    int64
	readOffset    int64
	currentData   []byte
	currentBufPtr *[]byte
	currentIndex  int

	// Coordination
	downloadWg  sync.WaitGroup
	requestCh   chan int64
	responseChs chan chan *partData
	started     bool

	// Buffer
	bufferPool sync.Pool
}

type partData struct {
	partNum int64
	data    []byte
	bufPtr  *[]byte
	err     error
}

// NewParallelReader creates a new parallel reader for downloading objects
func NewParallelReader(ctx context.Context, client Client, size int64, partSize int64, parallelism int, opts GetOptions) *parallelReader {
	totalParts := (size + partSize - 1) / partSize

	// Create a cancellable context for internal cancellation
	derivedCtx, cancelCause := context.WithCancelCause(ctx)

	pr := &parallelReader{
		ctx:         derivedCtx,
		cancelCause: cancelCause,
		client:      client,
		size:        size,
		partSize:    partSize,
		parallelism: parallelism,
		opts:        opts,
		totalParts:  totalParts,
		requestCh:   make(chan int64, parallelism*workerQueueDepth),
		responseChs: make(chan chan *partData, parallelism*workerQueueDepth),
	}

	// Initialize buffer pool to reuse allocations
	pr.bufferPool.New = func() any {
		buf := make([]byte, partSize)
		return &buf
	}

	return pr
}

// Start begins parallel downloading
func (pr *parallelReader) Start() error {
	if pr.started {
		return nil
	}
	pr.started = true

	// Start worker goroutines for downloading
	for i := 0; i < pr.parallelism; i++ {
		pr.downloadWg.Add(1)
		go pr.downloadWorker()
	}

	// Start result collector
	go pr.collectResults()

	// Start request scheduler
	go pr.scheduleRequests()

	return nil
}

// scheduleRequests sends part numbers to download workers
func (pr *parallelReader) scheduleRequests() {
	defer close(pr.requestCh)

	for partNum := range pr.totalParts {
		select {
		case <-pr.ctx.Done():
			return
		case pr.requestCh <- partNum:
		}
	}
}

// downloadWorker downloads parts from the source
func (pr *parallelReader) downloadWorker() {
	defer pr.downloadWg.Done()

	for {
		select {
		case <-pr.ctx.Done():
			return
		case responseCh, ok := <-pr.responseChs:
			if !ok {
				return
			}
			pr.downloadPart(responseCh)
		}
	}
}

// downloadPart downloads a single part and sends it back on the response channel
func (pr *parallelReader) downloadPart(responseCh chan *partData) {
	// Get part number from request channel
	select {
	case <-pr.ctx.Done():
		return
	case partNum, ok := <-pr.requestCh:
		if !ok {
			return
		}

		start := partNum * pr.partSize
		end := min(pr.size, start+pr.partSize) - 1
		length := end - start + 1

		// Create a copy of opts with range set
		opts := pr.opts
		opts.RangeStart = start

		// Download the part
		reader, _, err := pr.client.Get(pr.ctx, opts)
		if err != nil {
			responseCh <- &partData{
				partNum: partNum,
				err:     err.ToGoError(),
			}
			return
		}
		defer reader.Close()

		// Get a buffer from the pool
		bufPtr := pr.bufferPool.Get().(*[]byte)
		buf := *bufPtr
		data := buf[:length] // Slice to the actual length needed

		n, readErr := io.ReadFull(reader, data)
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			pr.bufferPool.Put(bufPtr) // Return buffer to pool on error
			responseCh <- &partData{
				partNum: partNum,
				err:     readErr,
			}
			return
		}

		responseCh <- &partData{
			partNum: partNum,
			data:    data[:n],
			bufPtr:  bufPtr,
			err:     nil,
		}
	}
}

// collectResults collects downloaded parts and buffers them
func (pr *parallelReader) collectResults() {
	pr.downloadWg.Wait()
	close(pr.responseChs)
}

// Read implements io.Reader interface
func (pr *parallelReader) Read(p []byte) (n int, err error) {
	// Check if context is cancelled
	if err := context.Cause(pr.ctx); err != nil {
		return 0, err
	}

	// If we have data in current buffer, return it
	if pr.currentData != nil && pr.currentIndex < len(pr.currentData) {
		n = copy(p, pr.currentData[pr.currentIndex:])
		pr.currentIndex += n
		pr.readOffset += int64(n)

		// If we've consumed all current data, return buffer to pool
		if pr.currentIndex >= len(pr.currentData) {
			if pr.currentBufPtr != nil {
				pr.bufferPool.Put(pr.currentBufPtr)
				pr.currentBufPtr = nil
			}
			pr.currentData = nil
			pr.currentIndex = 0
		}
		return n, nil
	}

	// Check if we've read everything
	if pr.currentPart >= pr.totalParts {
		if pr.readOffset >= pr.size {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("unexpected end of download stream")
	}

	// Request next part
	responseCh := make(chan *partData, 1)
	select {
	case <-pr.ctx.Done():
		return 0, context.Cause(pr.ctx)
	case pr.responseChs <- responseCh:
	}

	// Wait for the part
	select {
	case <-pr.ctx.Done():
		return 0, context.Cause(pr.ctx)
	case part := <-responseCh:
		if part.err != nil {
			// Return buffer to pool on error
			if part.bufPtr != nil {
				pr.bufferPool.Put(part.bufPtr)
			}
			// Cancel context with the error cause
			pr.cancelCause(part.err)
			return 0, part.err
		}

		pr.currentPart++
		pr.currentData = part.data
		pr.currentBufPtr = part.bufPtr
		pr.currentIndex = 0

		// Copy data to output
		n = copy(p, pr.currentData)
		pr.currentIndex += n
		pr.readOffset += int64(n)

		// If we've consumed all current data, return buffer to pool
		if pr.currentIndex >= len(pr.currentData) {
			if pr.currentBufPtr != nil {
				pr.bufferPool.Put(pr.currentBufPtr)
				pr.currentBufPtr = nil
			}
			pr.currentData = nil
			pr.currentIndex = 0
		}
		return n, nil
	}
}

// Close implements io.Closer
func (pr *parallelReader) Close() error {
	if !pr.started {
		return nil
	}

	// Cancel the context if not already cancelled
	pr.cancelCause(nil)

	pr.downloadWg.Wait()

	// Return current data buffer if any
	if pr.currentBufPtr != nil {
		pr.bufferPool.Put(pr.currentBufPtr)
		pr.currentBufPtr = nil
		pr.currentData = nil
	}

	return nil
}
