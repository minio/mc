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
		for range totalMsgs {
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
