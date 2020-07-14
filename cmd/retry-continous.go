/*
 * MinIO Go Library for Amazon S3 Compatible Cloud Storage
 * Copyright 2015-2017 MinIO, Inc.
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
