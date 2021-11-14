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

// NB. Since go 1.16 go-flags have "Func" and it might help to reduce code but go 1.12 specified in the package "cli"
import "github.com/minio/cli"

// nameFiltersFlagValue is a type for []nameFilter to satisfy flag.Value and flag.Getter
// We have multiple flag values refers to this slice,
// so we use an indirect structure to change it in the all places simultaneously
type nameFiltersFlagValue struct {
	filters []nameFilter
}

// Set appends the type pattern value to the list
func (f *nameFiltersFlagValue) Set(filter nameFilter) error {
	f.filters = append(f.filters, filter)
	return nil
}

// String designed return a readable representation of this value (for usage defaults) but return empty string
// because it is a compound flag
func (f *nameFiltersFlagValue) String() string {
	// Not implemented because it is a compound flag
	return ""
}

// Get returns the slice of filter set by this flag
func (f *nameFiltersFlagValue) Get() []nameFilter {
	return f.filters
}

// nameFilterTypeMapper is a type to bind multiple flags with single slice and to map flags to corresponding types
// We should use separate entities for mappers and values but Set(string) is not compatible for this right now
type nameFilterTypeMapper struct {
	// value is the reference to a shared flag value
	value *nameFiltersFlagValue

	// mapFunc is the function that maps string representation of a command line argument to an instance of nameFilter.
	// This works as a factory function
	mapFunc func(pattern string) (nameFilter, error)
}

func (f nameFilterTypeMapper) Set(value string) error {
	// Set do not provide information about the flag, so we cannot store multiple flags with different logic
	// in one slice. For this reason, we map the string to a typed value, so we can distinguish between them later
	filter, err := f.mapFunc(value)

	if err != nil {
		return err
	}

	return f.value.Set(filter)
}

func (f nameFilterTypeMapper) String() string {
	return f.value.String()
}

func (f nameFilterTypeMapper) Get() []nameFilter {
	return f.value.Get()
}

func newExcludeWildcardFilterFlag(name string, usage string, flagValue *nameFiltersFlagValue) *cli.GenericFlag {
	// We do not have the special function in the package 'cli' to create a typed flag, so we use a generic type
	return &cli.GenericFlag{
		Name:  name,
		Usage: usage,
		Value: nameFilterTypeMapper{
			flagValue,
			func(pattern string) (nameFilter, error) {
				return excludeWildcardFilter{pattern}, nil
			}},
	}
}

func newIncludeWildcardFilterFlag(name string, usage string, flagValue *nameFiltersFlagValue) *cli.GenericFlag {
	// We do not have the special function in the package 'cli' to create a typed flag, so we use a generic type
	return &cli.GenericFlag{
		Name:  name,
		Usage: usage,
		Value: nameFilterTypeMapper{
			flagValue,
			func(pattern string) (nameFilter, error) {
				return includeWildcardFilter{pattern}, nil
			}},
	}
}
