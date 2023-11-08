// Copyright (c) 2015-2022 MinIO, Inc.
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
	"os"
	"path/filepath"

	json "github.com/minio/colorjson"
)

// isValidAccessPERM - is provided access perm string supported.
func (b accessPerms) isValidAccessPERM() bool {
	switch b {
	case accessNone, accessDownload, accessUpload, accessPrivate, accessPublic:
		return true
	}
	return false
}

type PolicyDocument struct {
	Version   string `json:"Version"`
	Statement []struct {
		Effect    string    `json:"Effect"`
		Action    []string  `json:"Action"`
		Resource  []string  `json:"Resource"`
		Condition Condition `json:"Condition"`
	} `json:"Statement"`
}

type Condition map[string]map[string]interface{}

func (b accessPerms) isValidAccessFile() bool {

	if filepath.Ext(string(b)) != ".json" {
		fatalIf(errDummy().Trace(), "Invalid access file extension. Only .json files are supported.")
		return false
	}

	file, err := os.Open(string(b))
	if err != nil {
		fatalIf(errDummy().Trace(), "Unable to open access file.")
		return false
	}
	defer file.Close()

	var policy PolicyDocument
	if json.NewDecoder(file).Decode(&policy) != nil {
		fatalIf(errDummy().Trace(), "Unable to parse access file.")
		return false
	}

	if policy.Version != "2012-10-17" {
		fatalIf(errDummy().Trace(), "Invalid policy version. Only 2012-10-17 is supported.")
		return false
	}

	for _, statement := range policy.Statement {
		if statement.Effect != "Allow" && statement.Effect != "Deny" {
			fatalIf(errDummy().Trace(), "Invalid policy effect. Only Allow and Deny are supported.")
			return false
		}
	}

	return true
}

// accessPerms - access level.
type accessPerms string

// different types of Access perm's currently supported by policy command.
const (
	accessNone     = accessPerms("none")
	accessDownload = accessPerms("download")
	accessUpload   = accessPerms("upload")
	accessPrivate  = accessPerms("private")
	accessPublic   = accessPerms("public")
	accessCustom   = accessPerms("custom")
)
