/*
 * Minio Client (C) 2017 Minio, Inc.
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
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Maximum number of parallel workers
	maxParallelWorkers = 32
	// Monitor tick to decide to add new workers
	monitorPeriod = 5 * time.Second
)

// ParallelManager - helps manage parallel workers to run tasks
type ParallelManager struct {
	// pb shows current progress
	pb Progress

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

// monitorProgress monitors realtime transfer speed of data
// and increases threads until it reaches a maximum number of
// threads or notice there is no apparent enhancement of
// transfer speed.
func (p *ParallelManager) monitorProgress() {
	if p.pb == nil {
		// Nothing to monitor
		return
	}

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
				sentBytes := p.pb.Get()
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

				p.addWorker()
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
func newParallelManager(resultCh chan URLs, pb Progress) (*ParallelManager, chan func() URLs) {
	p := &ParallelManager{
		pb:            pb,
		wg:            &sync.WaitGroup{},
		workersNum:    0,
		stopMonitorCh: make(chan struct{}),
		queueCh:       make(chan func() URLs),
		resultCh:      resultCh,
	}

	// Add at least one worker to execute the job
	p.addWorker()

	// Start monitoring tasks progress
	p.monitorProgress()

	return p, p.queueCh
}
