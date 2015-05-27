/*
 * Minio Client (C) 2015 Minio, Inc.
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

// Package countlock is useful for synchronizing multiple go routines, when
// one of them has to take a lead. Up() method increments an internal
// counter. Down() method decrements, but blocks and waits when it
// reaches zero. One can call Down() only as many times Up() is called.
package countlock

// Locker defines a counting lock
type Locker interface {
	Up()    // Increment the counter.
	Down()  // Decrement the counter. When it reaches zero, block and wait.
	Close() // Release all resources and return.
}

// New returns a newly initialized counter lock object.
func New() Locker {
	upCh := make(chan bool)
	downCh := make(chan bool)
	closeCh := make(chan bool)

	go func(upCh, downCh, closeCh chan bool) {
		defer close(closeCh)
		defer close(downCh)
		defer close(upCh)

		var level int64 // Keeps track of Up and Down count.
		for {

			if level == 0 {
				// Up() has neven been called or may
				// be Down() has been called equal number
				// of times.

				select {
				case <-upCh:
					level++
				case <-closeCh: // We are done.
					return
				}
			}

			// Up is called at least once. It is safe to also listen on down.
			select {
			case <-upCh:
				level++
			case <-downCh:
				level--
			case <-closeCh:
				return
			}
		}
	}(upCh, downCh, closeCh)

	return &lock{upCh: upCh, downCh: downCh, closeCh: closeCh}
}

type lock struct {
	upCh    chan bool
	downCh  chan bool
	closeCh chan bool
}

func (l lock) Up() {
	l.upCh <- true
}

func (l lock) Down() {
	l.downCh <- true
}

func (l lock) Close() {
	l.closeCh <- true
}
