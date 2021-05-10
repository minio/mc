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

// Package probe implements a simple mechanism to trace and return errors in large programs.
package probe

// wrappedError implements a container for *probe.Error.
type wrappedError struct {
	err *Error
}

// WrapError function wraps a *probe.Error into a 'error' compatible duck type.
func WrapError(err *Error) error {
	return &wrappedError{err: err}
}

// UnwrapError tries to convert generic 'error' into typed *probe.Error and returns true, false otherwise.
func UnwrapError(err error) (*Error, bool) {
	switch e := err.(type) {
	case *wrappedError:
		return e.err, true
	default:
		return nil, false
	}
}

// Error interface method.
func (w *wrappedError) Error() string {
	return w.err.String()
}
