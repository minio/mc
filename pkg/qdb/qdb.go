/*
 * qdb - Quick key value store
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

// Store - generic config interface functions
type Store interface {
	Int
	Float64
	String
	Map
	GetVersion() Version
	Save(string) error
	Load(string) error
	Merge(Store) error
	Diff(Store) ([]string, error)
	GetStore() map[string]interface{}
	String() string
}

// Int - integer generic interface functions for qconfig
type Int interface {
	SetInt(string, int)
	GetInt(string) int
	SetIntSlice(string, []int)
	GetIntSlice(string) []int
}

// Float64 - float64 generic interface functions for qconfig
type Float64 interface {
	SetFloat64(string, float64)
	GetFloat64(string) float64
	SetFloat64Slice(string, []float64)
	GetFloat64Slice(string) []float64
}

// String - string generic interface functions for qconfig
type String interface {
	SetString(string, string)
	GetString(string) string
	SetStringSlice(string, []string)
	GetStringSlice(string) []string
}

// Map - map generic interface functions for qconfig
type Map interface {
	SetMapString(string, map[string]string)
	GetMapString(string) map[string]string
	SetMapStringSlice(string, map[string][]string)
	GetMapStringSlice(string) map[string][]string
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

// NewStore - instantiate a new config
func NewStore(version Version) Store {
	// error condition
	if version.Major == 0 {
		return nil
	}

	config := new(store)
	config.store = make(map[string]interface{})
	config.store["Version"] = version.String()
	config.lock = new(sync.RWMutex)

	return config
}

// GetVersion returns the current config file format version
func (c store) GetVersion() Version {
	val, ok := c.store["Version"].(string)
	if !ok {
		return Version{}
	}
	return Str2Version(val)
}

// SetInt sets int value
func (c *store) SetInt(key string, value int) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()
	(*c).store[key] = value
}

// GetInt returns int value
func (c store) GetInt(key string) int {
	val, _ := c.store[key].(int)
	return val
}

// GetIntSlice returns list of int values
func (c store) GetIntSlice(key string) []int {
	val, _ := c.store[key].([]int)
	return val
}

// SetIntSlice sets list of int values
func (c *store) SetIntSlice(key string, values []int) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()
	(*c).store[key] = values
}

// SetFloat64 sets 64-bit float value
func (c *store) SetFloat64(key string, value float64) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()
	(*c).store[key] = value
}

// GetFloat64 returns 64-bit float value
func (c store) GetFloat64(key string) float64 {
	val, _ := c.store[key].(float64)
	return val
}

// SetFloat64Slice sets a list of 64-bit float values
func (c *store) SetFloat64Slice(key string, values []float64) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()
	(*c).store[key] = values
}

// GetFloat64Slice returns a list of 64-bit float values
func (c store) GetFloat64Slice(key string) []float64 {
	val, _ := c.store[key].([]float64)
	return val
}

// SetString sets string value
func (c *store) SetString(key string, value string) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()
	(*c).store[key] = value
}

// GetString returns string value
func (c store) GetString(key string) string {
	val, _ := c.store[key].(string)
	return val
}

// SetStringSlice sets list of strings
func (c *store) SetStringSlice(key string, values []string) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()
	(*c).store[key] = values
}

// GetStringSlice returns list of strings
func (c store) GetStringSlice(key string) []string {
	val, _ := c.store[key].([]string)
	return val
}

//SetMapString sets a map of strings
func (c *store) SetMapString(key string, value map[string]string) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()
	(*c).store[key] = value
}

//GetMapString returns a map of strings
func (c store) GetMapString(key string) map[string]string {
	val, _ := c.store[key].(map[string]string)
	return val
}

//SetMapStringSlice sets a map of string list
func (c *store) SetMapStringSlice(key string, value map[string][]string) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()
	(*c).store[key] = value
}

//GetMapStringSlice returns a map of string list
func (c store) GetMapStringSlice(key string) map[string][]string {
	val, _ := c.store[key].(map[string][]string)
	return val
}

// Save writes configuration data in JSON format to donut config file.
func (c store) Save(filename string) (err error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	jsonStore, err := json.MarshalIndent(c.store, "", "\t")
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

// Load - loads JSON config from file and also automatically merges new changes
func (c *store) Load(filename string) (err error) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()

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

	if !reflect.DeepEqual((*c).store["Version"], loadedStore["Version"]) {
		return errors.New("Version mismatch")
	}

	// Merge pre-set keys
	for key := range loadedStore {
		(*c).store[key] = loadedStore[key]
	}

	return nil
}

// GetStore - grab internal store map for reading
func (c store) GetStore() map[string]interface{} {
	return c.store
}

// Merge - fast forward old keys to old+new keys
func (c *store) Merge(s Store) (err error) {
	(*c).lock.Lock()
	defer (*c).lock.Unlock()

	for key := range s.GetStore() {
		(*c).store[key] = s.GetStore()[key]
	}
	return nil
}

// Diff - provide difference between two qdb.Store bi-directionally
func (c store) Diff(s Store) (diffErrors []string, err error) {
	if reflect.DeepEqual(c.store["Version"], s.GetStore()["Version"]) == false {
		msg := fmt.Sprintf("Version mismatch newVersion:%s oldVersion:%s", c.store["Version"].(string), s.GetStore()["Version"].(string))
		diffErrors = append(diffErrors, msg)
	}

	for key := range s.GetStore() {
		_, ok := c.store[key]
		if !ok {
			msg := fmt.Sprintf("Key %s missing from newVersion: %s", key, c.store["Version"].(string))
			diffErrors = append(diffErrors, msg)
		}
	}

	for key := range c.store {
		_, ok := s.GetStore()[key]
		if !ok {
			msg := fmt.Sprintf("Key %s missing from oldVersion: %s", key, s.GetStore()["Version"].(string))
			diffErrors = append(diffErrors, msg)
		}
	}

	return diffErrors, nil
}

// String converts JSON config to printable string
func (c store) String() string {
	configBytes, _ := json.MarshalIndent(c.store, "", "\t")
	return string(configBytes)
}
