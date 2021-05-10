// +build solaris openbsd

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

var (
	// EventTypePut contains the notify events that will cause a put (writer)
	EventTypePut = []notify.Event{notify.Create, notify.Write, notify.Rename}
	// EventTypeDelete contains the notify events that will cause a delete (remove)
	EventTypeDelete = []notify.Event{notify.Remove}
	// EventTypeGet contains the notify events that will cause a get (read)
	EventTypeGet = []notify.Event{} // On macOS, FreeBSD, Solaris this is not available.
)

// IsGetEvent checks if the event return is a get event.
func IsGetEvent(event notify.Event) bool {
	return false
}

// IsPutEvent checks if the event returned is a put event
func IsPutEvent(event notify.Event) bool {
	for _, ev := range EventTypePut {
		if event&ev != 0 {
			return true
		}
	}
	return false
}

// IsDeleteEvent checks if the event returned is a delete event
func IsDeleteEvent(event notify.Event) bool {
	return event&notify.Remove != 0
}

// getAtllXAttrs returns the extended attributes for a file if supported
// by the OS
func getAllXattrs(path string) (map[string]string, error) {
	return nil, nil
}
