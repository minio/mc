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
	"fmt"

	"github.com/minio/pkg/wildcard"
)

// nameFilter is a general interface for implementing a simple filter,
// such as filters for excluding or including names in a processing list.
// For the reason, only name is available we can implement a few useful filters:
// - the name is equal to a predefined value
// - the name matches the wildcard pattern
// - the name is in a predefined list
// - the name matches a regular expression
// - the name matches using a custom algorithm
// and so on
type nameFilter interface {
	// name is a string, such as a file name
	// included specifies the current state of file processing and helps optimize the matching
	filter(name string, included bool) (includeAfter bool)
}

type nameFilterSlice []nameFilter

// filter helps to process the name throughout the list
func (f nameFilterSlice) filter(name string) (includeAfter bool) {
	// We use the positive sense to avoid massive double negation operations
	includeAfter = true
	for _, filter := range f {
		includeAfter = filter.filter(name, includeAfter)
	}
	return
}

// includeWildcardFilter implements a filter that includes a name if it matches a wildcard pattern
type includeWildcardFilter struct {
	pattern string
}

func (f includeWildcardFilter) filter(name string, included bool) (includeAfter bool) {
	// 1. Do not perform matching is the name is already included
	// 2. Include the name if it matches the pattern
	return included || wildcard.Match(f.pattern, name)
}

// String returns a string representation for debugging purposes
func (f includeWildcardFilter) String() string {
	return fmt.Sprintf("{include \"%s\"}", f.pattern)
}

// excludeWildcardFilter implements a filter that excludes a name if it matches a wildcard pattern
type excludeWildcardFilter struct {
	pattern string
}

func (f excludeWildcardFilter) filter(name string, included bool) (includeAfter bool) {
	// 1. Do not perform matching when the name is already excluded
	// 2. Exclude the name when it not matches the pattern
	return included && !wildcard.Match(f.pattern, name)
}

// String returns a string representation for debugging purposes
func (f excludeWildcardFilter) String() string {
	return fmt.Sprintf("{exclude \"%s\"}", f.pattern)
}
