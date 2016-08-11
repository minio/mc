/*
 * Minio Client, (C) 2016 Minio, Inc.
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

package mc

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Comparer is an interface for queued objects, used to identify duplicate objects
// in queue.
type Comparer interface {
	Equal(interface{}) bool
}

// Queue allows objects to be queued on a first in first out base.
type Queue struct {
	a []interface{}

	m sync.Mutex
	i int // read index
	j int // write index

	idleCh chan interface{}

	closed bool

	session *sessionV7
}

// NewQueue creates a new queue
func NewQueue(session *sessionV7) *Queue {
	return &Queue{
		idleCh:  make(chan interface{}),
		closed:  false,
		session: session,
	}
}

// Save writes the current queue content to the writer
func (q *Queue) Save(dst io.Writer) error {
	q.m.Lock()
	defer q.m.Unlock()

	for i := q.i; i < q.j; i++ {
		jsonData, err := json.Marshal(q.a[i])
		if err != nil {
			return err
		}

		fmt.Fprintln(dst, string(jsonData))
	}

	return nil
}

// Wait for items to become available
func (q *Queue) Wait() error {
	q.m.Lock()

	if q.closed {
		q.m.Unlock()
		return fmt.Errorf("Channel closed")
	}

	if q.i < q.j {
		q.m.Unlock()
		return nil
	}

	q.m.Unlock()

	_, ok := <-q.idleCh
	if !ok {
		return fmt.Errorf("Channel closed")
	}

	return nil
}

// Close and disable the queue
func (q *Queue) Close() {
	q.m.Lock()
	defer q.m.Unlock()

	q.closed = true
	close(q.idleCh)
}

// Pop the first object on the queue
func (q *Queue) Pop() interface{} {
	q.m.Lock()
	defer q.m.Unlock()

	if q.closed {
		return nil
	}

	if q.i >= q.j {
		return nil
	}

	defer func() {
		q.i++
	}()

	return q.a[q.i]
}

func (q *Queue) grow(n int) {
	a := make([]interface{}, q.j+10)

	// we can slide in future, for now just grow the array
	copy(a[0:q.j], q.a[0:q.j])
	q.a = a
}

// ErrObjectAlreadyQueued occurs when the object being pushed already exists
// in the queue
var ErrObjectAlreadyQueued = fmt.Errorf("Object already queued.")

// Count returns the number of items in the queue
func (q *Queue) Count() int {
	q.m.Lock()
	defer q.m.Unlock()
	return q.j - q.i
}

// Push a new object to the queue
func (q *Queue) Push(u interface{}) error {
	q.m.Lock()
	defer q.m.Unlock()

	for i := q.i; i < q.j; i++ {
		// prevent items not yet copied, to be queued twice

		if s, ok := u.(Comparer); !ok {
			// no comparer interface
			break
		} else if !s.Equal(q.a[i]) {
			// not equal
			continue
		}

		return ErrObjectAlreadyQueued
	}

	if q.j >= len(q.a) {
		q.grow(10)
	}

	q.a[q.j] = u
	q.j++

	select {
	case q.idleCh <- true:
	default:
		// non blocking send, causes wait to resume
	}

	return nil
}
