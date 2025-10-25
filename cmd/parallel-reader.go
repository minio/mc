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
	"context"
	"fmt"
	"io"
	"sync"
)

// parallelReader reads an object in parallel using range requests
type parallelReader struct {
	ctx         context.Context
	client      Client
	size        int64
	partSize    int64
	parallelism int
	opts        GetOptions

	// State management
	currentPart  int64
	totalParts   int64
	partBuffer   map[int64][]byte
	bufferMu     sync.Mutex
	readOffset   int64
	currentData  []byte
	currentIndex int

	// Error handling
	err     error
	errOnce sync.Once

	// Coordination
	downloadWg sync.WaitGroup
	requestCh  chan int64
	resultCh   chan *partData
	done       chan struct{}
	started    bool
}

type partData struct {
	partNum int64
	data    []byte
	err     error
}

// NewParallelReader creates a new parallel reader for downloading objects
func NewParallelReader(ctx context.Context, client Client, size int64, partSize int64, parallelism int, opts GetOptions) *parallelReader {
	totalParts := (size + partSize - 1) / partSize

	return &parallelReader{
		ctx:         ctx,
		client:      client,
		size:        size,
		partSize:    partSize,
		parallelism: parallelism,
		opts:        opts,
		totalParts:  totalParts,
		partBuffer:  make(map[int64][]byte),
		requestCh:   make(chan int64, parallelism*2),
		resultCh:    make(chan *partData, parallelism*2),
		done:        make(chan struct{}),
	}
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

	for partNum := int64(0); partNum < pr.totalParts; partNum++ {
		select {
		case <-pr.done:
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
		case <-pr.done:
			return
		case partNum, ok := <-pr.requestCh:
			if !ok {
				return
			}
			pr.downloadPart(partNum)
		}
	}
}

// downloadPart downloads a single part
func (pr *parallelReader) downloadPart(partNum int64) {
	start := partNum * pr.partSize
	end := start + pr.partSize - 1
	if end >= pr.size {
		end = pr.size - 1
	}
	length := end - start + 1

	// Create a copy of opts with range set
	opts := pr.opts
	opts.RangeStart = start

	// Download the part
	reader, _, err := pr.client.Get(pr.ctx, opts)
	if err != nil {
		pr.resultCh <- &partData{
			partNum: partNum,
			err:     err.ToGoError(),
		}
		return
	}
	defer reader.Close()

	// Read the entire part into memory
	data := make([]byte, length)
	n, readErr := io.ReadFull(reader, data)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		pr.resultCh <- &partData{
			partNum: partNum,
			err:     readErr,
		}
		return
	}

	pr.resultCh <- &partData{
		partNum: partNum,
		data:    data[:n],
		err:     nil,
	}
}

// collectResults collects downloaded parts and buffers them
func (pr *parallelReader) collectResults() {
	pr.downloadWg.Wait()
	close(pr.resultCh)
}

// Read implements io.Reader interface
func (pr *parallelReader) Read(p []byte) (n int, err error) {
	// Return any previous error
	if pr.err != nil {
		return 0, pr.err
	}

	// If we have data in current buffer, return it
	if pr.currentData != nil && pr.currentIndex < len(pr.currentData) {
		n = copy(p, pr.currentData[pr.currentIndex:])
		pr.currentIndex += n
		pr.readOffset += int64(n)

		// If we've consumed all current data, clear it
		if pr.currentIndex >= len(pr.currentData) {
			pr.currentData = nil
			pr.currentIndex = 0
		}
		return n, nil
	}

	// Need to get next part
	for {
		// Check if we already have the next part buffered
		pr.bufferMu.Lock()
		if data, ok := pr.partBuffer[pr.currentPart]; ok {
			delete(pr.partBuffer, pr.currentPart)
			pr.bufferMu.Unlock()

			pr.currentPart++
			pr.currentData = data
			pr.currentIndex = 0

			// Copy data to output
			n = copy(p, pr.currentData)
			pr.currentIndex += n
			pr.readOffset += int64(n)

			// If we've consumed all current data, clear it
			if pr.currentIndex >= len(pr.currentData) {
				pr.currentData = nil
				pr.currentIndex = 0
			}
			return n, nil
		}
		pr.bufferMu.Unlock()

		// Wait for next part from result channel
		part, ok := <-pr.resultCh
		if !ok {
			// No more parts
			if pr.readOffset >= pr.size {
				return 0, io.EOF
			}
			return 0, fmt.Errorf("unexpected end of download stream")
		}

		if part.err != nil {
			pr.errOnce.Do(func() {
				pr.err = part.err
				close(pr.done)
			})
			return 0, part.err
		}

		// Check if this is the part we're waiting for
		if part.partNum == pr.currentPart {
			pr.currentPart++
			pr.currentData = part.data
			pr.currentIndex = 0

			// Copy data to output
			n = copy(p, pr.currentData)
			pr.currentIndex += n
			pr.readOffset += int64(n)

			// If we've consumed all current data, clear it
			if pr.currentIndex >= len(pr.currentData) {
				pr.currentData = nil
				pr.currentIndex = 0
			}
			return n, nil
		}

		// Buffer this part for later
		pr.bufferMu.Lock()
		pr.partBuffer[part.partNum] = part.data
		pr.bufferMu.Unlock()
	}
}

// Close implements io.Closer
func (pr *parallelReader) Close() error {
	if !pr.started {
		return nil
	}

	select {
	case <-pr.done:
		// Already closed
	default:
		close(pr.done)
	}

	pr.downloadWg.Wait()
	return nil
}
