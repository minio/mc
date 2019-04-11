// +build solaris openbsd

/*
 * MinIO Client (C) 2015, 2016, 2017 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this fs except in compliance with the License.
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
