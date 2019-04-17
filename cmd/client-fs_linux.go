// +build linux

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

import (
	"encoding/hex"
	"strings"

	"github.com/pkg/xattr"
	"github.com/rjeczalik/notify"

	"unicode/utf8"
)

var (
	// EventTypePut contains the notify events that will cause a put (write)
	EventTypePut = []notify.Event{notify.InCloseWrite | notify.InMovedTo}
	// EventTypeDelete contains the notify events that will cause a delete (remove)
	EventTypeDelete = []notify.Event{notify.InDelete | notify.InDeleteSelf | notify.InMovedFrom}
	// EventTypeGet contains the notify events that will cause a get (read)
	EventTypeGet = []notify.Event{notify.InAccess | notify.InOpen}
)

// IsGetEvent checks if the event return is a get event.
func IsGetEvent(event notify.Event) bool {
	for _, ev := range EventTypeGet {
		if event&ev != 0 {
			return true
		}
	}
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
	for _, ev := range EventTypeDelete {
		if event&ev != 0 {
			return true
		}
	}
	return false
}

// getXAttr fetches the extended attribute for a particular key on
// file
func getXAttr(path, key string) (string, error) {
	data, e := xattr.Get(path, key)
	if e != nil {
		return "", e
	}
	if utf8.ValidString(string(data)) {
		return string(data), nil
	}
	return hex.EncodeToString(data), nil
}

// getAllXattrs returns the extended attributes for a file if supported
// by the OS
func getAllXattrs(path string) (map[string]string, error) {
	xMetadata := make(map[string]string)
	list, e := xattr.List(path)
	if e != nil {
		if isNotSupported(e) {
			return nil, nil
		}
		return nil, e
	}
	for _, key := range list {
		// filter out system specific xattr
		if strings.HasPrefix(key, "system.") {
			continue
		}
		xMetadata[key], e = getXAttr(path, key)
		if e != nil {
			if isNotSupported(e) {
				return nil, nil
			}
			return nil, e
		}
	}
	return xMetadata, nil
}
