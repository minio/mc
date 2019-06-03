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
	"fmt"
	"sync"
	"testing"

	"github.com/rjeczalik/notify"
)

func testPipeChan(inputCh, outputCh chan notify.EventInfo, totalMsgs int) error {

	var wg sync.WaitGroup

	msgCtnt := notify.EventInfo(nil)

	// Send messages up to totalMsgs
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < totalMsgs; i++ {
			inputCh <- msgCtnt
		}
		close(inputCh)
	}()

	// Goroutine to receive and check messages
	var recvMsgs int
	var recvErr error

	wg.Add(1)

	go func() {
		defer wg.Done()
		for msg := range outputCh {
			recvMsgs++
			if msg != msgCtnt {
				recvErr = fmt.Errorf("Corrupted message, expected = `%s`, found = `%s`", msgCtnt, msg)
				return
			}
		}
	}()

	// Wait until we finish sending and receiving messages
	wg.Wait()

	if recvErr != nil {
		return recvErr
	}

	// Check if all messages are received
	if recvMsgs != totalMsgs {
		return fmt.Errorf("unable to receive all messages, expected: %d, lost: %d", totalMsgs, totalMsgs-recvMsgs)
	}

	return nil
}

// Testing code

func TestPipeChannel(t *testing.T) {
	fastCh, slowCh := PipeChan(1000)
	err := testPipeChan(fastCh, slowCh, 10*1000)
	if err != nil {
		t.Errorf("ERR: %v\n", err)
	}
}

func TestRegularChannel(t *testing.T) {
	ch := make(chan notify.EventInfo, 1000)
	err := testPipeChan(ch, ch, 10*1000)
	if err != nil {
		t.Errorf("ERR: %v\n", err)
	}
}

// Benchmark code

func benchmarkRegular(b *testing.B, msgsNum int) {
	for n := 0; n < b.N; n++ {
		ch := make(chan notify.EventInfo, msgsNum)
		testPipeChan(ch, ch, msgsNum)
	}
}

func benchmarkPipeChan(b *testing.B, msgsNum int) {
	for n := 0; n < b.N; n++ {
		fastCh, slowCh := PipeChan(1000)
		testPipeChan(fastCh, slowCh, msgsNum)
	}
}

func BenchmarkRegular1M(b *testing.B) {
	benchmarkRegular(b, 1*1000*1000)
}

func BenchmarkPipeChan1M(b *testing.B) {
	benchmarkPipeChan(b, 1*1000*1000)
}

func BenchmarkRegular100K(b *testing.B) {
	benchmarkRegular(b, 100*1000)
}

func BenchmarkPipeChan100K(b *testing.B) {
	benchmarkPipeChan(b, 100*1000)
}

func BenchmarkRegular10K(b *testing.B) {
	benchmarkRegular(b, 10*1000)
}

func BenchmarkPipeChan10K(b *testing.B) {
	benchmarkPipeChan(b, 10*1000)
}

func BenchmarkRegular1K(b *testing.B) {
	benchmarkRegular(b, 1*1000)
}

func BenchmarkPipeChan1K(b *testing.B) {
	benchmarkPipeChan(b, 1*1000)
}
