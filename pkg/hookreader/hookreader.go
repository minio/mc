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

// Package hookreader hooks additional reader in the source
// stream. It is useful for making progress bars. Second reader is
// appropriately notified about the exact number of bytes read from
// the primary source on each Read operation.
package hookreader

import "io"

// hookReader hooks additional reader in the source stream. It is
// useful for making progress bars. Second reader is appropriately
// notified about the exact number of bytes read from the primary
// source on each Read operation.
type hookReader struct {
	source io.Reader
	hook   io.Reader
}

// Seek implements io.Seeker. Seeks source first, and if necessary
// seeks hook if Seek method is appropriately found.
func (hr *hookReader) Seek(offset int64, whence int) (n int64, err error) {
	// Verify for source has embedded Seeker, use it.
	sourceSeeker, ok := hr.source.(io.Seeker)
	if ok {
		return sourceSeeker.Seek(offset, whence)
	}
	// Verify if hook has embedded Seeker, use it.
	hookSeeker, ok := hr.hook.(io.Seeker)
	if ok {
		return hookSeeker.Seek(offset, whence)
	}
	return n, nil
}

// Read implements io.Reader. Always reads from the source, the return
// value 'n' number of bytes are reported through the hook. Returns
// error for all non io.EOF conditions.
func (hr *hookReader) Read(b []byte) (n int, err error) {
	n, err = hr.source.Read(b)
	if err != nil && err != io.EOF {
		return n, err
	}
	// Progress the hook with the total read bytes from the source.
	if _, herr := hr.hook.Read(b[:n]); herr != nil {
		if herr != io.EOF {
			return n, herr
		}
	}
	return n, err
}

// NewHook returns a io.Reader which implements hookReader that
// reports the data read from the source to the hook.
func NewHook(source, hook io.Reader) io.Reader {
	if hook == nil {
		return source
	}
	return &hookReader{source, hook}
}
