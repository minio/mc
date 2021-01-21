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

	// Channels to receive tasks to run
	queueChannels []chan task

	// Channel to send back results
	resultCh chan URLs

	stopMonitorCh chan struct{}
}

// addWorker creates a new worker to process tasks
func (p *ParallelManager) addWorker(channelNum int) {
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
			t, ok := <-p.queueChannels[channelNum]
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

				for i := 0; i < defaultWorkerFactor; i++ {
					p.addWorker(1)
				}
			}
		}
	}()
}

// Queue task in parallel
func (p *ParallelManager) queueTask(fn func() URLs, longOp bool) {
	p.doQueueTask(task{fn: fn}, longOp)
}

// Queue task but ensures that no tasks is running at parallel,
// which also means wait until all concurrent tasks finish before
// queueing this and execute it solely.
func (p *ParallelManager) queueTaskWithBarrier(fn func() URLs, longOp bool) {
	p.doQueueTask(task{fn: fn, barrier: true}, longOp)
}

func (p *ParallelManager) doQueueTask(t task, longOp bool) {
	if t.barrier {
		p.barrierSync.Lock()
	} else {
		p.barrierSync.RLock()
	}
	if longOp {
		p.queueChannels[0] <- t
	} else {
		p.queueChannels[1] <- t
	}
}

// Wait for all workers to finish tasks before shutting down Parallel
func (p *ParallelManager) stopAndWait() {
	for _, q := range p.queueChannels {
		close(q)
	}
	p.wg.Wait()
	close(p.stopMonitorCh)
}

// newParallelManager starts new workers waiting for executing tasks
func newParallelManager(resultCh chan URLs) *ParallelManager {
	queueChannels := make([]chan task, 2)
	for i := range queueChannels {
		queueChannels[i] = make(chan task)
	}
	p := &ParallelManager{
		wg:            &sync.WaitGroup{},
		workersNum:    0,
		stopMonitorCh: make(chan struct{}),
		queueChannels: queueChannels,
		resultCh:      resultCh,
	}

	// Start one worker which uploads big files
	p.addWorker(0)

	// Start workers for small files with runtime.NumCPU().
	for i := 0; i < runtime.NumCPU(); i++ {
		p.addWorker(1)
	}

	// Start monitoring tasks progress
	p.monitorProgress()

	return p
}
