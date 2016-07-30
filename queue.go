package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

type Comparer interface {
	Equal(interface{}) bool
}

type Queue struct {
	a []interface{}

	m sync.Mutex
	i int // read index
	j int // write index

	idleCh chan interface{}

	closed bool

	session *sessionV7
}

func NewQueue(session *sessionV7) *Queue {
	return &Queue{
		idleCh:  make(chan interface{}),
		closed:  false,
		session: session,
	}
}

func (q *Queue) Save(dst io.Writer) error {
	q.m.Lock()
	defer q.m.Unlock()

	for i := q.i; i < q.j; i++ {
		if jsonData, err := json.Marshal(q.a[i]); err != nil {
			return err
		} else {
			fmt.Fprintln(dst, string(jsonData))
		}
	}

	return nil
}

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

func (q *Queue) Close() {
	q.m.Lock()
	defer q.m.Unlock()

	q.closed = true
	close(q.idleCh)
}

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

var ErrObjectAlreadyQueued = fmt.Errorf("Object already queued.")

func (q *Queue) Count() int {
	q.m.Lock()
	defer q.m.Unlock()
	return len(q.a)
}

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
