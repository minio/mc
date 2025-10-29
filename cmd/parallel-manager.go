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
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/shirou/gopsutil/v3/mem"
)

const (
	// Maximum number of parallel workers
	maxParallelWorkers = 128

	// Monitor tick to decide to add new workers
	monitorPeriod = 4 * time.Second
)

// Number of workers added per bandwidth monitoring.
var defaultWorkerFactor = runtime.GOMAXPROCS(0)

// A task is a copy/mirror action that needs to be executed
type task struct {
	// The function to execute in this task
	fn func() URLs
	// If set to true, ensure no tasks are
	// executed in parallel to this one.
	barrier bool
	// The total size of the information that we need to upload
	uploadSize int64
}

// ParallelManager - helps manage parallel workers to run tasks
type ParallelManager struct {
	// Calculate sent bytes.
	// Keep this as first element of struct because it guarantees 64bit
	// alignment on 32 bit machines. atomic.* functions crash if operand is not
	// aligned at 64bit. See https://github.com/golang/go/issues/599
	sentBytes int64

	// Synchronize workers
	wg          *sync.WaitGroup
	barrierSync sync.RWMutex

	// Current threads number
	workersNum uint32

	// Channel to receive tasks to run
	queueCh chan task

	// Channel to send back results
	resultCh chan URLs

	stopMonitorCh chan struct{}

	// The maximum memory to use
	maxMem uint64

	// The maximum workers will be created
	maxWorkers int
}

// addWorker creates a new worker to process tasks
func (p *ParallelManager) addWorker() {
	maxWorkers := maxParallelWorkers
	if p.maxWorkers > 0 {
		maxWorkers = p.maxWorkers
	}
	if int(atomic.LoadUint32(&p.workersNum)) >= maxWorkers {
		// Number of maximum workers is reached, no need to
		// to create a new one.
		return
	}

	// Update number of threads
	atomic.AddUint32(&p.workersNum, 1)

	// Start a new worker
	p.wg.Add(1)
	go func() {
		for {
			// Wait for jobs
			t, ok := <-p.queueCh
			if !ok {
				// No more tasks, quit
				p.wg.Done()
				return
			}

			// Execute the task and send the result to channel.
			p.resultCh <- t.fn()

			if t.barrier {
				p.barrierSync.Unlock()
			} else {
				p.barrierSync.RUnlock()
			}
		}
	}()
}

func (p *ParallelManager) Read(b []byte) (n int, err error) {
	atomic.AddInt64(&p.sentBytes, int64(len(b)))
	return len(b), nil
}

// monitorProgress monitors realtime transfer speed of data
// and increases threads until it reaches a maximum number of
// threads or notice there is no apparent enhancement of
// transfer speed.
func (p *ParallelManager) monitorProgress() {
	go func() {
		ticker := time.NewTicker(monitorPeriod)
		defer ticker.Stop()

		var prevSentBytes, maxBandwidth int64
		var retry int

		for {
			select {
			case <-p.stopMonitorCh:
				// Ordered to quit immediately
				return
			case <-ticker.C:
				// Compute new bandwidth from counted sent bytes
				sentBytes := atomic.LoadInt64(&p.sentBytes)
				bandwidth := sentBytes - prevSentBytes
				prevSentBytes = sentBytes

				if bandwidth <= maxBandwidth {
					retry++
					// We still want to add more workers
					// until we are sure that it is not
					// useful to add more of them.
					if retry > 2 {
						return
					}
				} else {
					retry = 0
					maxBandwidth = bandwidth
				}

				for range defaultWorkerFactor {
					p.addWorker()
				}
			}
		}
	}()
}

// Queue task in parallel
func (p *ParallelManager) queueTask(fn func() URLs, uploadSize int64) {
	p.doQueueTask(task{fn: fn, uploadSize: uploadSize})
}

// Queue task but ensures that no tasks is running at parallel,
// which also means wait until all concurrent tasks finish before
// queueing this and execute it solely.
func (p *ParallelManager) queueTaskWithBarrier(fn func() URLs, uploadSize int64) {
	p.doQueueTask(task{fn: fn, barrier: true, uploadSize: uploadSize})
}

func (p *ParallelManager) enoughMemForUpload(uploadSize int64) bool {
	if uploadSize < 0 {
		panic("unexpected size")
	}

	if uploadSize == 0 || p.maxMem == 0 {
		return true
	}

	estimateNeededMemoryForUpload := func(size int64) uint64 {
		partsCount, partSize, _, e := minio.OptimalPartInfo(size, 0)
		if e != nil {
			panic(e)
		}
		if partsCount >= 4 {
			return 4 * uint64(partSize)
		}
		return uint64(size)
	}

	smem := runtime.MemStats{}
	if uploadSize > 50<<20 {
		// GC if upload is bigger than 50MB.
		runtime.GC()
	}
	runtime.ReadMemStats(&smem)

	return estimateNeededMemoryForUpload(uploadSize)+smem.Alloc < p.maxMem
}

func (p *ParallelManager) doQueueTask(t task) {
	// Check if we have enough memory to perform next task,
	// if not, wait to finish all currents tasks to continue
	if !p.enoughMemForUpload(t.uploadSize) {
		t.barrier = true
	}
	if t.barrier {
		p.barrierSync.Lock()
	} else {
		p.barrierSync.RLock()
	}
	p.queueCh <- t
}

// Wait for all workers to finish tasks before shutting down Parallel
func (p *ParallelManager) stopAndWait() {
	close(p.queueCh)
	p.wg.Wait()
	close(p.stopMonitorCh)
}

const cgroupLimitFile = "/sys/fs/cgroup/memory/memory.limit_in_bytes"

func cgroupLimit(limitFile string) (limit uint64) {
	buf, e := os.ReadFile(limitFile)
	if e != nil {
		return 9223372036854771712
	}
	limit, e = strconv.ParseUint(string(buf), 10, 64)
	if e != nil {
		return 9223372036854771712
	}
	return limit
}

func availableMemory() (available uint64) {
	available = 4 << 30 // Default to 4 GiB when we can't find the limits.

	if runtime.GOOS == "linux" {
		available = cgroupLimit(cgroupLimitFile)

		// No limit set, It's the highest positive signed 64-bit
		// integer (2^63-1), rounded down to multiples of 4096 (2^12),
		// the most common page size on x86 systems - for cgroup_limits.
		if available != 9223372036854771712 {
			// This means cgroup memory limit is configured.
			return
		} // no-limit set proceed to set the limits based on virtual memory.

	} // for all other platforms limits are based on virtual memory.

	memStats, _ := mem.VirtualMemory()
	if memStats.Available > 0 {
		available = memStats.Available
	}

	// Always use 50% of available memory.
	available = available / 2
	return
}

// newParallelManager starts new workers waiting for executing tasks
func newParallelManager(resultCh chan URLs, maxWorkers int) *ParallelManager {
	p := &ParallelManager{
		wg:            &sync.WaitGroup{},
		workersNum:    0,
		stopMonitorCh: make(chan struct{}),
		queueCh:       make(chan task),
		resultCh:      resultCh,
		maxMem:        availableMemory(),
		maxWorkers:    maxWorkers,
	}

	// Start with runtime.NumCPU().
	for i := 0; i < runtime.NumCPU(); i++ {
		p.addWorker()
	}

	// Start monitoring tasks progress
	p.monitorProgress()

	return p
}
