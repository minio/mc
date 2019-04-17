/*
 * MinIO Client (C) 2015 MinIO, Inc.
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

import "path/filepath"

// isValidAccessPERM - is provided access perm string supported.
func (b accessPerms) isValidAccessPERM() bool {
	switch b {
	case accessNone, accessDownload, accessUpload, accessPublic:
		return true
	}
	return false
}

func (b accessPerms) isValidAccessFile() bool {
	return filepath.Ext(string(b)) == ".json"
}

// accessPerms - access level.
type accessPerms string

// different types of Access perm's currently supported by policy command.
const (
	accessNone     = accessPerms("none")
	accessDownload = accessPerms("download")
	accessUpload   = accessPerms("upload")
	accessPublic   = accessPerms("public")
	accessCustom   = accessPerms("custom")
)
