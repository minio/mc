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

// Package yielder relinquishes control to other go routines more
// frequently in the middle of busy I/O activity
package yielder

import (
	"io"
	"runtime"
)

// NewReader returns an initialized yielding proxy reader.
func NewReader(reader io.ReadCloser) *Reader {
	return &Reader{reader, true}
}

// Reader implements io.Reader compatible scheduler.
type Reader struct {
	io.ReadCloser
	enabled bool
}

// Read yields CPU to other go routines. Useful in making low-priority I/O routines.
func (r Reader) Read(p []byte) (n int, err error) {
	if r.enabled {
		runtime.Gosched()
	}
	n, err = r.ReadCloser.Read(p)
	return
}

// Close closes yielder
func (r Reader) Close() (err error) {
	return r.ReadCloser.Close()
}

// Enable turns on the yielder.
func (r *Reader) Enable() {
	r.enabled = true
}

// Disable turns off the yielder.
func (r *Reader) Disable() {
	r.enabled = false
}
