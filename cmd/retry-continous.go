// Copyright (c) 2015-2021 MinIO, Inc.
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
	"math/rand"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
)

// lockedRandSource provides protected rand source, implements rand.Source interface.
type lockedRandSource struct {
	lk  sync.Mutex
	src rand.Source
}

// Int63 returns a non-negative pseudo-random 63-bit integer as an int64.
func (r *lockedRandSource) Int63() (n int64) {
	r.lk.Lock()
	n = r.src.Int63()
	r.lk.Unlock()
	return
}

// Seed uses the provided seed value to initialize the generator to a
// deterministic state.
func (r *lockedRandSource) Seed(seed int64) {
	r.lk.Lock()
	r.src.Seed(seed)
	r.lk.Unlock()
}

// Introduce a new locked random seed.
var random = rand.New(&lockedRandSource{src: rand.NewSource(time.Now().UTC().UnixNano())})

// newRetryTimerContinous creates a timer with exponentially increasing delays forever.
func newRetryTimerContinous(ctx context.Context, unit time.Duration, cap time.Duration, jitter float64) <-chan int {
	attemptCh := make(chan int)

	// normalize jitter to the range [0, 1.0]
	if jitter < minio.NoJitter {
		jitter = minio.NoJitter
	}
	if jitter > minio.MaxJitter {
		jitter = minio.MaxJitter
	}

	// computes the exponential backoff duration according to
	// https://www.awsarchitectureblog.com/2015/03/backoff.html
	exponentialBackoffWait := func(attempt int) time.Duration {
		// 1<<uint(attempt) below could overflow, so limit the value of attempt
		maxAttempt := 30
		if attempt > maxAttempt {
			attempt = maxAttempt
		}
		//sleep = random_between(0, min(cap, base * 2 ** attempt))
		sleep := unit * time.Duration(1<<uint(attempt))
		if sleep > cap {
			sleep = cap
		}
		if jitter != minio.NoJitter {
			sleep -= time.Duration(random.Float64() * float64(sleep) * jitter)
		}
		return sleep
	}

	go func() {
		defer close(attemptCh)
		var nextBackoff int
		for {
			select {
			case attemptCh <- nextBackoff:
				nextBackoff++
			case <-ctx.Done():
				return
			}

			select {
			case <-time.After(exponentialBackoffWait(nextBackoff)):
			case <-ctx.Done():
				return
			}
		}
	}()
	return attemptCh
}
