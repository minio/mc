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
