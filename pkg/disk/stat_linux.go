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

// Package disk fetches file system information for various OS
package disk

import (
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

// GetFileSystemAttrs return the file system attribute as string; containing mode,
// uid, gid, uname, Gname, atime, mtime, ctime and md5
func GetFileSystemAttrs(file string) (string, error) {
	fi, err := os.Stat(file)
	if err != nil {
		return "", err
	}
	st := fi.Sys().(*syscall.Stat_t)

	var fileAttr strings.Builder
	fileAttr.WriteString("atime:")
	fileAttr.WriteString(strconv.FormatInt(int64(st.Atim.Sec), 10) + "#" + strconv.FormatInt(int64(st.Atim.Nsec), 10))
	fileAttr.WriteString("/gid:")
	fileAttr.WriteString(strconv.Itoa(int(st.Gid)))

	g, err := user.LookupGroupId(strconv.FormatUint(uint64(st.Gid), 10))
	if err == nil {
		fileAttr.WriteString("/gname:")
		fileAttr.WriteString(g.Name)
	}

	fileAttr.WriteString("/mode:")
	fileAttr.WriteString(strconv.Itoa(int(st.Mode)))
	fileAttr.WriteString("/mtime:")
	fileAttr.WriteString(strconv.FormatInt(int64(st.Mtim.Sec), 10) + "#" + strconv.FormatInt(int64(st.Mtim.Nsec), 10))
	fileAttr.WriteString("/uid:")
	fileAttr.WriteString(strconv.Itoa(int(st.Uid)))

	u, err := user.LookupId(strconv.FormatUint(uint64(st.Uid), 10))
	if err == nil {
		fileAttr.WriteString("/uname:")
		fileAttr.WriteString(u.Username)
	}

	return fileAttr.String(), nil
}
