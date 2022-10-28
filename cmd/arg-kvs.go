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

// argKV - is a shorthand of each key value.
type argKV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// argKVS - is a shorthand for some wrapper functions
// to operate on list of key values.
type argKVS []argKV

// Empty - return if kv is empty
func (kvs argKVS) Empty() bool {
	return len(kvs) == 0
}

// Set sets a value, if not sets a default value.
func (kvs *argKVS) Set(key, value string) {
	for i, kv := range *kvs {
		if kv.Key == key {
			(*kvs)[i] = argKV{
				Key:   key,
				Value: value,
			}
			return
		}
	}
	*kvs = append(*kvs, argKV{
		Key:   key,
		Value: value,
	})
}

// Get - returns the value of a key, if not found returns empty.
func (kvs argKVS) Get(key string) string {
	v, ok := kvs.Lookup(key)
	if ok {
		return v
	}
	return ""
}

// Lookup - lookup a key in a list of argKVS
func (kvs argKVS) Lookup(key string) (string, bool) {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}
