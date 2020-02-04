/*
 * MinIO Client (C) 2017 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Maximum number of parallel workers
	maxParallelWorkers = 128

	// Monitor tick to decide to add new workers
	monitorPeriod = 4 * time.Second

	// Number of workers added per bandwidth monitoring.
	defaultWorkerFactor = 2
)

// ParallelManager - helps manage parallel workers to run tasks
type ParallelManager struct {
	// Calculate sent bytes.
	// Keep this as first element of struct because it guarantees 64bit
	// alignment on 32 bit machines. atomic.* functions crash if operand is not
	// aligned at 64bit. See https://github.com/golang/go/issues/599
	sentBytes int64

	// Synchronize workers
	wg *sync.WaitGroup

	// Current threads number
	workersNum uint32

	// Channel to receive tasks to run
	queueCh chan func() URLs
	// Channel to send back results
	resultCh chan URLs

	stopMonitorCh chan struct{}
}

// addWorker creates a new worker to process tasks
func (p *ParallelManager) addWorker() {
	if atomic.LoadUint32(&p.workersNum) >= maxParallelWorkers {
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
			fn, ok := <-p.queueCh
			if !ok {
				// No more tasks, quit
				p.wg.Done()
				return
			}
			// Execute the task and send the result
			// to result channel.
			p.resultCh <- fn()
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

				for i := 0; i < defaultWorkerFactor; i++ {
					p.addWorker()
				}
			}
		}
	}()
}

// Wait for all workers to finish tasks before shutting down Parallel
func (p *ParallelManager) wait() {
	p.wg.Wait()
	close(p.stopMonitorCh)
}

// newParallelManager starts new workers waiting for executing tasks
func newParallelManager(resultCh chan URLs) (*ParallelManager, chan func() URLs) {
	p := &ParallelManager{
		wg:            &sync.WaitGroup{},
		workersNum:    0,
		stopMonitorCh: make(chan struct{}),
		queueCh:       make(chan func() URLs),
		resultCh:      resultCh,
	}

	// Start with runtime.NumCPU().
	for i := 0; i < runtime.NumCPU(); i++ {
		p.addWorker()
	}

	// Start monitoring tasks progress
	p.monitorProgress()

	return p, p.queueCh
}
