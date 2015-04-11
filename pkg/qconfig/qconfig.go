/*
 * QConfig - Quick way to implement a configuration file
 *
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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

package qconfig

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"encoding/json"
	"io/ioutil"
)

// Configure - generic config interface functions
type Configure interface {
	Int
	Float64
	String
	StringSlice
	Map
	GetVersion() Version
	SaveConfig(string) error
	LoadConfig(string) error
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
}

// String - string generic interface functions for qconfig
type String interface {
	SetString(string, string)
	GetString(string) string
}

// StringSlice - string slice generic interface functions for qconfig
type StringSlice interface {
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

// GetMajor version
func (v Version) GetMajor() int {
	return v.Major
}

// GetMinor version
func (v Version) GetMinor() int {
	return v.Minor
}

// GetPatch version
func (v Version) GetPatch() int {
	return v.Patch
}

// String - get version string
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Config -
type Config map[string]interface{}

// NewConfig - instantiate a new config
func NewConfig(major, minor, patch int) Configure {
	// error condition
	if major < 0 || minor < 0 || patch < 0 {
		return nil
	}
	config := make(Config)
	version := Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
	config["Version"] = version.String()
	return &config
}

// GetVersion returns the current config file format version
func (c Config) GetVersion() Version {
	val, ok := c["Version"].(string)
	if !ok {
		return Version{}
	}
	var version Version
	fmt.Sscanf(val, "%d.%d.%d", &version.Major, &version.Minor, &version.Patch)
	return version
}

// SetInt sets int value
func (c *Config) SetInt(key string, value int) {
	(*c)[key] = value
}

// GetInt returns int value
func (c Config) GetInt(key string) int {
	val, _ := c[key].(int)
	return val
}

// GetIntSlice returns list of int values
func (c Config) GetIntSlice(key string) []int {
	val, _ := c[key].([]int)
	return val
}

// SetIntSlice sets list of int values
func (c *Config) SetIntSlice(key string, values []int) {
	(*c)[key] = values
}

// SetFloat64 sets 64-bit float value
func (c *Config) SetFloat64(key string, value float64) {
	(*c)[key] = value
}

// GetFloat64 returns 64-bit float value
func (c Config) GetFloat64(key string) float64 {
	val, _ := c[key].(float64)
	return val
}

// SetString sets string value
func (c *Config) SetString(key string, value string) {
	(*c)[key] = value
}

// GetString returns string value
func (c Config) GetString(key string) string {
	val, _ := c[key].(string)
	return val
}

// SetStringSlice sets list of strings
func (c *Config) SetStringSlice(key string, values []string) {
	(*c)[key] = values
}

// GetStringSlice returns list of strings
func (c Config) GetStringSlice(key string) []string {
	val, _ := c[key].([]string)
	return val
}

//SetMapString sets a map of strings
func (c *Config) SetMapString(key string, value map[string]string) {
	(*c)[key] = value
}

//GetMapString returns a map of strings
func (c Config) GetMapString(key string) map[string]string {
	val, _ := c[key].(map[string]string)
	return val
}

//SetMapStringSlice sets a map of string list
func (c *Config) SetMapStringSlice(key string, value map[string][]string) {
	(*c)[key] = value
}

//GetMapStringSlice returns a map of string list
func (c Config) GetMapStringSlice(key string) map[string][]string {
	val, _ := c[key].(map[string][]string)
	return val
}

// SaveConfig writes configuration data in JSON format to donut config file.
func (c Config) SaveConfig(filename string) (err error) {
	jsonConfig, err := json.MarshalIndent(c, "", "\t")
	// yamlConfig, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	// err = os.MkdirAll(GetDonutConfigDir(), 0755)
	err = os.MkdirAll(".", 0755)
	if !os.IsExist(err) && err != nil {
		return err
	}

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonConfig)
	// _, err = file.Write(yamlConfig)
	if err != nil {
		return err
	}
	return nil

}

// LoadConfig loads JSON config from file
func (c *Config) LoadConfig(filename string) (err error) {
	_, err = os.Stat(filename)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var loadedConfig Config
	err = json.Unmarshal(data, &loadedConfig)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual((*c)["Version"], loadedConfig["Version"]) {
		return errors.New("Version mismatch")
	}

	// Merge pre-set keys
	for key := range loadedConfig {
		(*c)[key] = loadedConfig[key]
	}
	return nil
}

// String converts JSON config to printable string
func (c Config) String() string {
	// configBytes, _ := yaml.Marshal(c)
	configBytes, _ := json.MarshalIndent(c, "", "\t")
	return string(configBytes)
}

/*
func main() {

	cfg := NewConfig(Version{1, 0, 0})

	cfg.SetInt("mykey", 345)
	fmt.Println(cfg.GetInt("mykey"))

	cfg.SetString("mykeyQ", "Hello Q")
	fmt.Println(cfg.GetString("mykeyQ"))

	cfg.SetMapString("mymap", map[string]string{"mykey": "Hello Q"})
	fmt.Println(cfg.GetMapString("mymap"))

	//cfg.SaveConfig("test.json")

	cfg.SetStringSlice("MyDonut", []string{"/media/disk1", "/media/disk2", "/media/badDisk99", "/media/badDisk100"})
	cfg.SetString("MyDonut1", "/media/disk1")
	cfg.SaveConfig("test.json")

	newCfg := NewConfig(Version{1, 1, 0})
	if err := newCfg.LoadConfig("test.json"); err != nil {
		fmt.Printf("Error loading config, %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("%v\n", newCfg.String())
}
*/
