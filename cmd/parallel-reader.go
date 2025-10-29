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
	"io"
	"sync"
)

const (
	// workerQueueDepth controls channel buffer sizing relative to parallelism.
	// Prevents workers from blocking and keeps the download pipeline full while using
	// minimal memory (only part numbers and pointers, not actual data buffers).
	workerQueueDepth = 2
)

// ParallelReader reads an object in parallel using range requests
type ParallelReader struct {
	ctx         context.Context
	cancelCause context.CancelCauseFunc
	client      Client
	size        int64
	partSize    int64
	parallelism int
	opts        GetOptions

	// State management
	totalParts    int64
	readOffset    int64
	currentData   []byte
	currentBufPtr *[]byte
	currentIndex  int

	// Coordination
	downloadWg  sync.WaitGroup
	collectorWg sync.WaitGroup
	requestCh   chan int64
	resultCh    chan *partData
	partBuffer  map[int64]*partData
	nextPart    int64
	bufferMu    sync.Mutex
	resultReady *sync.Cond
	started     bool
	closeMu     sync.Mutex
	closed      bool

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
func NewParallelReader(ctx context.Context, client Client, size int64, partSize int64, parallelism int, opts GetOptions) *ParallelReader {
	totalParts := (size + partSize - 1) / partSize

	// Create a cancellable context for internal cancellation
	derivedCtx, cancelCause := context.WithCancelCause(ctx)

	pr := &ParallelReader{
		ctx:         derivedCtx,
		cancelCause: cancelCause,
		client:      client,
		size:        size,
		partSize:    partSize,
		parallelism: parallelism,
		opts:        opts,
		totalParts:  totalParts,
		requestCh:   make(chan int64, parallelism*workerQueueDepth),
		resultCh:    make(chan *partData, parallelism*workerQueueDepth),
		partBuffer:  make(map[int64]*partData),
	}
	pr.resultReady = sync.NewCond(&pr.bufferMu)

	// Initialize buffer pool to reuse allocations
	pr.bufferPool.New = func() any {
		buf := make([]byte, partSize)
		return &buf
	}

	return pr
}

// Start begins parallel downloading
func (pr *ParallelReader) Start() error {
	if pr.started {
		return nil
	}
	pr.started = true

	for i := 0; i < pr.parallelism; i++ {
		pr.downloadWg.Add(1)
		go pr.downloadWorker()
	}

	// Start result collector and request scheduler
	pr.collectorWg.Add(1)
	go pr.collectResults()
	go pr.scheduleRequests()

	return nil
}

// scheduleRequests sends part numbers to download workers
func (pr *ParallelReader) scheduleRequests() {
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
func (pr *ParallelReader) downloadWorker() {
	defer pr.downloadWg.Done()

	for {
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
			opts.RangeEnd = end // Set end for precise range request

			// Download the part
			reader, _, err := pr.client.Get(pr.ctx, opts)
			if err != nil {
				select {
				case pr.resultCh <- &partData{partNum: partNum, err: err.ToGoError()}:
				case <-pr.ctx.Done():
				}
				continue
			}

			// Get a buffer from the pool
			bufPtr := pr.bufferPool.Get().(*[]byte)
			buf := *bufPtr
			data := buf[:length]

			n, readErr := io.ReadFull(reader, data)
			reader.Close()

			if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
				pr.bufferPool.Put(bufPtr) // Return buffer to pool on error
				select {
				case pr.resultCh <- &partData{partNum: partNum, err: readErr}:
				case <-pr.ctx.Done():
				}
				continue
			}

			select {
			case pr.resultCh <- &partData{partNum: partNum, data: data[:n], bufPtr: bufPtr}:
			case <-pr.ctx.Done():
				pr.bufferPool.Put(bufPtr)
				return
			}
		}
	}
}

// collectResults collects downloaded parts and buffers them
func (pr *ParallelReader) collectResults() {
	defer pr.collectorWg.Done()
	defer func() {
		// Wake up any waiting Read() calls when collector exits
		// This prevents deadlock if context is canceled while Read() is waiting
		pr.bufferMu.Lock()
		pr.resultReady.Broadcast()
		pr.bufferMu.Unlock()
	}()

	for part := range pr.resultCh {
		pr.bufferMu.Lock()
		pr.partBuffer[part.partNum] = part
		pr.resultReady.Broadcast() // Wake up Read() if waiting for this part
		pr.bufferMu.Unlock()
	}
}

// Read implements io.Reader interface
func (pr *ParallelReader) Read(p []byte) (n int, err error) {
	// Check if context is canceled
	select {
	case <-pr.ctx.Done():
		return 0, context.Cause(pr.ctx)
	default:
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

	// Wait for the next sequential part - check EOF under lock
	pr.bufferMu.Lock()

	// Check if we've read everything (now protected by lock)
	if pr.nextPart >= pr.totalParts {
		pr.bufferMu.Unlock()
		if pr.readOffset >= pr.size {
			return 0, io.EOF
		}
		return 0, io.ErrUnexpectedEOF
	}

	for {
		// Check if we have the part we need
		part, ok := pr.partBuffer[pr.nextPart]
		if ok {
			// Remove from buffer
			delete(pr.partBuffer, pr.nextPart)
			pr.nextPart++
			pr.bufferMu.Unlock()

			// Handle error
			if part.err != nil {
				if part.bufPtr != nil {
					pr.bufferPool.Put(part.bufPtr)
				}
				pr.cancelCause(part.err)
				return 0, part.err
			}

			// Set as current data
			pr.currentData = part.data
			pr.currentBufPtr = part.bufPtr
			pr.currentIndex = 0

			// Copy to output
			n = copy(p, pr.currentData)
			pr.currentIndex += n
			pr.readOffset += int64(n)

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

		// Check for cancellation
		select {
		case <-pr.ctx.Done():
			pr.bufferMu.Unlock()
			return 0, context.Cause(pr.ctx)
		default:
		}

		// Wait for signal that a new part arrived
		pr.resultReady.Wait()
	}
}

// Close implements io.Closer
func (pr *ParallelReader) Close() error {
	pr.closeMu.Lock()
	if !pr.started || pr.closed {
		pr.closeMu.Unlock()
		return nil
	}
	pr.closed = true
	pr.closeMu.Unlock()

	// Cancel the context if not already canceled
	pr.cancelCause(nil)

	// Wait for workers to finish
	pr.downloadWg.Wait()

	// Close result channel and wait for collector to finish
	close(pr.resultCh)
	pr.collectorWg.Wait()

	// Clean up any buffered parts safely after collector is finished
	pr.bufferMu.Lock()
	for _, part := range pr.partBuffer {
		if part.bufPtr != nil {
			pr.bufferPool.Put(part.bufPtr)
		}
	}
	pr.partBuffer = nil
	pr.bufferMu.Unlock()

	// Return current data buffer if any
	if pr.currentBufPtr != nil {
		pr.bufferPool.Put(pr.currentBufPtr)
		pr.currentBufPtr = nil
		pr.currentData = nil
	}

	return nil
}
