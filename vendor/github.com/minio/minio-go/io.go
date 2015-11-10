/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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

package minio

import "io"

type readSeekCloser struct {
	r io.Reader
}

// NewReadSeekCloser wraps an io.Reader returning a ReadSeekCloser
func NewReadSeekCloser(r io.Reader) ReadSeekCloser {
	return &readSeekCloser{r}
}

// ReadSeekCloser is the container
// that groups the basic Seek and Close methods.
type ReadSeekCloser interface {
	io.Reader
	io.Closer
	io.Seeker
}

// Read reads up to len(p) bytes into p.  It returns the number of bytes
// read (0 <= n <= len(p)) and any error encountered.  Even if Read
// returns n < len(p), it may use all of p as scratch space during the call.
// If some data is available but not len(p) bytes, Read conventionally
// returns what is available instead of waiting for more.
//
// When Read encounters an error or end-of-file condition after
// successfully reading n > 0 bytes, it returns the number of
// bytes read.  It may return the (non-nil) error from the same call
// or return the error (and n == 0) from a subsequent call.
// An instance of this general case is that a Reader returning
// a non-zero number of bytes at the end of the input stream may
// return either err == EOF or err == nil.  The next Read should
// return 0, EOF.
//
// If no io.Reader found nothing will be done.
func (r *readSeekCloser) Read(p []byte) (int, error) {
	switch t := r.r.(type) {
	case io.Reader:
		return t.Read(p)
	}
	return 0, nil
}

// Seek sets the offset for the next Read or Write to offset,
// interpreted according to whence: 0 means relative to the start of
// the file, 1 means relative to the current offset, and 2 means
// relative to the end. Seek returns the new offset relative to the
// start of the file and an error, if any.
//
// Seeking to an offset before the start of the file is an error.
//
// If no io.Seeker found nothing will be done.
func (r *readSeekCloser) Seek(offset int64, whence int) (int64, error) {
	switch t := r.r.(type) {
	case io.Seeker:
		return t.Seek(offset, whence)
	}
	return int64(0), nil
}

// Close closes the ReadSeekCloser.
//
// The behavior of Close after the first call is undefined.
// Specific implementations may document their own behavior.
//
// If no io.Closer found nothing will be done.
func (r *readSeekCloser) Close() error {
	switch t := r.r.(type) {
	case io.Closer:
		return t.Close()
	}
	return nil
}
