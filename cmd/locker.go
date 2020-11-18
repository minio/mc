/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
)

type lock struct {
	sync.Mutex
	ref int
}

type namedLocker struct {
	mu    sync.Mutex
	locks map[string]*lock
}

// Lock a name
func (m *namedLocker) Lock(name string) {
	m.mu.Lock()
	l, ok := m.locks[name]
	if !ok {
		l = &lock{}
		m.locks[name] = l
	}
	l.ref++
	m.mu.Unlock()

	l.Mutex.Lock()
}

// Unlock a name, do nothing if name is not locked
func (m *namedLocker) Unlock(name string) {
	m.mu.Lock()
	l, ok := m.locks[name]
	if !ok {
		m.mu.Unlock()
		// No lock found, do nothing
		return
	}
	l.ref--
	if l.ref < 1 {
		delete(m.locks, name)
	}
	m.mu.Unlock()

	l.Mutex.Unlock()
}

// newNamedLocker initializes a new named locker
func NewNamedLocker() *namedLocker {
	return &namedLocker{locks: make(map[string]*lock)}
}
