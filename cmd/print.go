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
	"bytes"
	"encoding/json"
	"strings"

	"github.com/minio/pkg/console"
)

// message interface for all structured messages implementing JSON(), String() methods.
type message interface {
	JSON() string
	String() string
}

// printMsg prints message string or JSON structure depending on the type of output console.
func printMsg(msg message) {
	var msgStr string
	if !globalJSON {
		msgStr = msg.String()
	} else {
		msgStr = msg.JSON()
		if globalJSONLine && strings.ContainsRune(msgStr, '\n') {
			// Reformat.
			var dst bytes.Buffer
			if err := json.Compact(&dst, []byte(msgStr)); err == nil {
				msgStr = dst.String()
			}
		}
	}
	console.Println(msgStr)
}
