/*
 * qdb - Quick key value store for config files and persistent state files
 *
 * Mini Copy, (C) 2015 Minio, Inc.
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

package qdb

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"encoding/json"
	"io/ioutil"
)

// Store - generic db interface functions
type Store interface {
	Int
	Float64
	String
	Map
	MapMap
	GetVersion() Version
	Save(string) error
	Load(string) error
	Merge(Store) error
	Diff(Store) ([]string, error)
	DeepDiff(Store) ([]string, error)
	GetStore() map[string]interface{}
	String() string
}

// Int - integer generic interface functions for qdb
type Int interface {
	SetInt(string, int)
	GetInt(string) int
	SetIntSlice(string, []int)
	GetIntSlice(string) []int
}

// Float64 - float64 generic interface functions for qdb
type Float64 interface {
	SetFloat64(string, float64)
	GetFloat64(string) float64
	SetFloat64Slice(string, []float64)
	GetFloat64Slice(string) []float64
}

// String - string generic interface functions for qdb
type String interface {
	SetString(string, string)
	GetString(string) string
	SetStringSlice(string, []string)
	GetStringSlice(string) []string
}

// Map - map generic interface functions for qdb
type Map interface {
	SetMapString(string, map[string]string)
	GetMapString(string) map[string]string
	SetMapStringSlice(string, map[string][]string)
	GetMapStringSlice(string) map[string][]string
}

// MapMap - two level map indirection interface functions for qdb
type MapMap interface {
	SetMapMapString(string, map[string]map[string]string)
	GetMapMapString(string) map[string]map[string]string
}

// Version info
type Version struct {
	Major int
	Minor int
	Patch int
}

// Equal compares for equality between two version structures
func (v Version) Equal(newVer Version) bool {
	return reflect.DeepEqual(v, newVer)
}

// String - get version string
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Str2Version - converts string to version
func Str2Version(verStr string) Version {
	var v Version
	fmt.Sscanf(verStr, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	return v
}

// store - implements qdb.Store interface
type store struct {
	store map[string]interface{}
	lock  *sync.RWMutex
}

// NewStore - instantiate a new db
func NewStore(version Version) Store {
	// error condition
	if version.Major == 0 {
		return nil
	}

	db := new(store)
	db.store = make(map[string]interface{})
	db.store["Version"] = version.String()
	db.lock = new(sync.RWMutex)

	return db
}

// GetVersion returns the current db file format version
func (s store) GetVersion() Version {
	val, ok := s.store["Version"].(string)
	if !ok {
		return Version{}
	}
	return Str2Version(val)
}

// SetInt sets int value
func (s *store) SetInt(key string, value int) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = value
}

// GetInt returns int value
func (s store) GetInt(key string) int {
	intVal, ok := s.store[key].(int)
	if !ok {
		interfaceIntVal, _ := s.store[key].(interface{})
		return int(reflect.ValueOf(interfaceIntVal).Float())
	}
	return intVal
}

// SetIntSlice sets list of int values
func (s *store) SetIntSlice(key string, values []int) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = values
}

// GetIntSlice returns list of int values
func (s store) GetIntSlice(key string) []int {
	interfaceIntSliceVal, ok := s.store[key].([]interface{})
	if !ok {
		intSliceVal, _ := s.store[key].([]int)
		return intSliceVal
	}
	var actualIntSliceVal []int
	for _, v := range interfaceIntSliceVal {
		vInt := reflect.ValueOf(v)
		actualIntSliceVal = append(actualIntSliceVal, int(vInt.Float()))
	}
	return actualIntSliceVal
}

// SetFloat64 sets 64-bit float value
func (s *store) SetFloat64(key string, value float64) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = value
}

// GetFloat64 returns 64-bit float value
func (s store) GetFloat64(key string) float64 {
	val := reflect.ValueOf(s.store[key])
	return val.Float()
}

// SetFloat64Slice sets a list of 64-bit float values
func (s *store) SetFloat64Slice(key string, values []float64) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = values
}

// GetFloat64Slice returns a list of 64-bit float values
func (s store) GetFloat64Slice(key string) []float64 {
	interfaceFloatSliceVal, ok := s.store[key].([]interface{})
	if !ok {
		floatSliceVal, _ := s.store[key].([]float64)
		return floatSliceVal
	}
	var actualFloatSliceVal []float64
	for _, v := range interfaceFloatSliceVal {
		val := reflect.ValueOf(v)
		actualFloatSliceVal = append(actualFloatSliceVal, val.Float())
	}
	return actualFloatSliceVal
}

// SetString sets string value
func (s *store) SetString(key string, value string) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = value
}

// GetString returns string value
func (s store) GetString(key string) string {
	val, _ := s.store[key].(string)
	return val
}

// SetStringSlice sets list of strings
func (s *store) SetStringSlice(key string, values []string) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = values
}

// GetStringSlice returns list of strings
func (s store) GetStringSlice(key string) []string {
	interfaceStrSliceVal, ok := s.store[key].([]interface{})
	if !ok {
		strSliceVal, _ := s.store[key].([]string)
		return strSliceVal
	}
	var actualStrSliceVal []string
	for _, v := range interfaceStrSliceVal {
		val := reflect.ValueOf(v)
		actualStrSliceVal = append(actualStrSliceVal, val.String())
	}
	return actualStrSliceVal
}

// SetMapString sets a map of strings
func (s *store) SetMapString(key string, value map[string]string) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = value
}

// GetMapString returns a map of strings
func (s store) GetMapString(key string) map[string]string {
	interfaceMapVal, ok := s.store[key].(map[string]interface{})
	if !ok {
		val, _ := s.store[key].(map[string]string)
		return val
	}
	actualMapVal := make(map[string]string)
	for k, v := range interfaceMapVal {
		rv := reflect.ValueOf(v)
		actualMapVal[k] = rv.String()
	}
	return actualMapVal
}

// SetMapStringSlice sets a map of string list
func (s *store) SetMapStringSlice(key string, value map[string][]string) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = value
}

// GetMapStringSlice returns a map of string list
func (s store) GetMapStringSlice(key string) map[string][]string {
	interfacelMapSliceVal, ok := s.store[key].(map[string]interface{})
	if !ok {
		mapSliceVal, _ := s.store[key].(map[string][]string)
		return mapSliceVal
	}
	actualMapSliceVal := make(map[string][]string)
	for k, v := range interfacelMapSliceVal {
		var actualStrSliceVal []string
		rv, _ := v.([]interface{})
		for _, l := range rv {
			val := reflect.ValueOf(l)
			actualStrSliceVal = append(actualStrSliceVal, val.String())
		}
		actualMapSliceVal[k] = actualStrSliceVal
	}
	return actualMapSliceVal
}

// SetMapMapString sets a map of a map string
func (s *store) SetMapMapString(key string, value map[string]map[string]string) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()
	(*s).store[key] = value
}

// GetMapMapString gets a map of a map string
func (s *store) GetMapMapString(key string) map[string]map[string]string {
	interfaceMapMapVal, ok := s.store[key].(map[string]interface{})
	if !ok {
		val, _ := s.store[key].(map[string]map[string]string)
		return val
	}
	actualMapMapVal := make(map[string]map[string]string)
	for k, v := range interfaceMapMapVal {
		rv, _ := v.(map[string]interface{})
		nestedActualMapVal := make(map[string]string)
		for m, n := range rv {
			rn := reflect.ValueOf(n)
			nestedActualMapVal[m] = rn.String()
		}
		actualMapMapVal[k] = nestedActualMapVal
	}
	return actualMapMapVal
}

// Save writes db data in JSON format to a file.
func (s store) Save(filename string) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	jsonStore, err := json.MarshalIndent(s.store, "", "\t")
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	if runtime.GOOS == "windows" {
		jsonStore = []byte(strings.Replace(string(jsonStore), "\n", "\r\n", -1))
	}
	_, err = file.Write(jsonStore)
	if err != nil {
		return err
	}
	return nil

}

// Load - loads JSON db from file and merge with currently set values
func (s *store) Load(filename string) (err error) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()

	_, err = os.Stat(filename)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		data = []byte(strings.Replace(string(data), "\r\n", "\n", -1))
	}

	var loadedStore map[string]interface{}
	err = json.Unmarshal(data, &loadedStore)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual((*s).store["Version"], loadedStore["Version"]) {
		return errors.New("Version mismatch")
	}

	// Merge pre-set keys
	for key := range loadedStore {
		(*s).store[key] = loadedStore[key]
	}
	return nil
}

// GetStore - grab internal store map for reading
func (s store) GetStore() map[string]interface{} {
	return s.store
}

// Merge - fast forward old keys to old+new keys
func (s *store) Merge(m Store) (err error) {
	(*s).lock.Lock()
	defer (*s).lock.Unlock()

	for key := range m.GetStore() {
		(*s).store[key] = m.GetStore()[key]
	}
	return nil
}

// Diff - returns list of keys that are in A but in B
func (s store) Diff(b Store) (keys []string, err error) {
	for key := range s.store {
		_, ok := b.GetStore()[key]
		if !ok {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// Diff - returns list of keys in A but not in B or of different values
func (s store) DeepDiff(b Store) (keys []string, err error) {
	for key, valA := range s.store {
		valB, ok := b.GetStore()[key]
		if !ok {
			keys = append(keys, key)
		} else {
			if !reflect.DeepEqual(valA, valB) {
				keys = append(keys, key)
			}
		}
	}
	return keys, nil
}

// String converts JSON db to printable string
func (s store) String() string {
	dbBytes, _ := json.MarshalIndent(s.store, "", "\t")
	return string(dbBytes)
}
