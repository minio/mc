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
	"github.com/minio/mc/pkg/probe"
)

// URLs contains source and target urls
type URLs struct {
	SourceAlias      string
	SourceContent    *ClientContent
	TargetAlias      string
	TargetContent    *ClientContent
	TotalCount       int64
	TotalSize        int64
	MD5              bool
	DisableMultipart bool
	encKeyDB         map[string][]prefixSSEPair
	Error            *probe.Error `json:"-"`
	ErrorCond        differType   `json:"-"`
}

// WithError sets the error and returns object
func (m URLs) WithError(err *probe.Error) URLs {
	m.Error = err
	return m
}

// Equal tests if both urls are equal
func (m URLs) Equal(n URLs) bool {
	if m.SourceContent == nil && n.SourceContent == nil {
	} else if m.SourceContent != nil && n.SourceContent == nil {
		return false
	} else if m.SourceContent == nil && n.SourceContent != nil {
		return false
	} else if m.SourceContent.URL != n.SourceContent.URL {
		return false
	}

	if m.TargetContent == nil && n.TargetContent == nil {
	} else if m.TargetContent != nil && n.TargetContent == nil {
		return false
	} else if m.TargetContent == nil && n.TargetContent != nil {
		return false
	} else if m.TargetContent.URL != n.TargetContent.URL {
		return false
	}

	return true
}
