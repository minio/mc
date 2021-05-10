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

import "github.com/rjeczalik/notify"

// Dynamically sized logical channel: a pipe which never blocks even when
// it receives too many elements. Memory consumption is increased and decresed
// on the fly following the number of elements. Here is the benchmark which
// compares a regular channel which a large capacity (like 1 Million) in
// contrast to a PipeChan with an realtime increasing/decreasing capacity.
//
// BenchmarkRegular1M-4                 200         172809399 ns/op
// BenchmarkPipeChan1M-4                100         316668338 ns/op
// BenchmarkRegular100K-4              2000          16540648 ns/op
// BenchmarkPipeChan100K-4             1000          31905966 ns/op
// BenchmarkRegular10K-4              20000           1637665 ns/op
// BenchmarkPipeChan10K-4             10000           3324329 ns/op
// BenchmarkRegular1K-4              200000            168512 ns/op
// BenchmarkPipeChan1K-4             100000            550623 ns/op

// PipeChan builds a new dynamically sized channel
func PipeChan(capacity int) (inputCh chan notify.EventInfo, outputCh chan notify.EventInfo) {

	// A set of channels which store all elements received from input
	channels := make(chan chan notify.EventInfo, 1000)

	inputCh = make(chan notify.EventInfo, capacity)

	// A goroutine which receives elements from inputCh and creates
	// new channels when needed.
	go func() {
		// Create the first channel
		currCh := make(chan notify.EventInfo, capacity)
		channels <- currCh

		for elem := range inputCh {
			// Prepare next channel with a double capacity when
			// half of the current channel is already filled.
			if len(currCh) >= cap(currCh)/2 {
				close(currCh)
				currCh = make(chan notify.EventInfo, cap(currCh)*2)
				channels <- currCh
			}
			// Prepare next channel with half capacity when
			// current channel is 1/4 filled
			if len(currCh) >= capacity && len(currCh) <= cap(currCh)/4 {
				close(currCh)
				currCh = make(chan notify.EventInfo, cap(currCh)/2)
				channels <- currCh
			}
			// Send element to current channel
			currCh <- elem
		}

		close(currCh)
		close(channels)
	}()

	// Copy elements from infinite channel set to the output
	outputCh = make(chan notify.EventInfo, capacity)
	go func() {
		for {
			currCh, ok := <-channels
			if !ok {
				break
			}
			for v := range currCh {
				outputCh <- v
			}
		}
		close(outputCh)
	}()
	return inputCh, outputCh
}
