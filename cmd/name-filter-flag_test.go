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
	"github.com/minio/cli"
	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestNameFilterNoFlagSet(c *C) {
	flagValue := &nameFiltersFlagValue{}

	c.Assert(flagValue.Get(), IsNil)
}

func (s *TestSuite) TestNameFilterTypeMapper(c *C) {
	flagValue := &nameFiltersFlagValue{}
	excludeFlag := newExcludeWildcardFilterFlag("exclude", "", flagValue)
	includeFlag := newIncludeWildcardFilterFlag("include", "", flagValue)

	optionsData := []struct {
		pattern string
		value   cli.Generic
		filter  nameFilter
	}{
		{"pat1", excludeFlag.Value, excludeWildcardFilter{"pat1"}},
		{"pat2", includeFlag.Value, includeWildcardFilter{"pat2"}},
		{"pat3", excludeFlag.Value, excludeWildcardFilter{"pat3"}},
	}

	targetFilters := make([]nameFilter, len(optionsData))
	for i, f := range optionsData {
		err := f.value.Set(f.pattern)
		c.Assert(err, IsNil)
		targetFilters[i] = f.filter
	}

	nameFilters := flagValue.Get()
	c.Assert(nameFilters, DeepEquals, targetFilters)
}
