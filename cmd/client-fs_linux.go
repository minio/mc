// +build linux

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
