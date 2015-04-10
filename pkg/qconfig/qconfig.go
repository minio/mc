/*
 * QConfig - Quick way to implement a configuration file
 * (C) 2015 Minio, Inc.
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

// package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"encoding/json"
	"io/ioutil"
)

type Configure interface {
	GetVersion() Version
	SetInt(string, int)
	GetInt(string) int
	SetIntList(string, []int)
	GetIntList(string) []int
	SetFloat64(string, float64)
	GetFloat64(string) float64
	SetString(string, string)
	GetString(string) string
	SetStringList(string, []string)
	GetStringList(string) []string
	SetMapString(string, map[string]string)
	GetMapString(string) map[string]string
	SetMapStringList(string, map[string][]string)
	GetMapStringList(string) map[string][]string
	SaveConfig(string) error
	LoadConfig(string) error
	String() string
}

// Version info
type Version struct {
	Major int
	Minor int
	Patch int
}

// Quick Config
type Config map[string]interface{}

func NewConfig(version Version) Configure {
	config := make(Config)
	var verStr string
	verStr = fmt.Sprintf("%d.%d.%d", version.Major, version.Minor, version.Patch)
	config["Version"] = verStr
	return &config
}

// GetVersion returns the current config file format version
func (c Config) GetVersion() Version {
	val, _ := c["Version"].(string)
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

// GetIntList returns list of int values
func (c Config) GetIntList(key string) []int {
	val, _ := c[key].([]int)
	return val
}

// SetIntList sets list of int values
func (c *Config) SetIntList(key string, values []int) {
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

// SetStringList sets list of strings
func (c *Config) SetStringList(key string, values []string) {
	(*c)[key] = values
}

// GetStringList returns list of strings
func (c Config) GetStringList(key string) []string {
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

//SetMapStringList sets a map of string list
func (c *Config) SetMapStringList(key string, value map[string][]string) {
	(*c)[key] = value
}

//GetMapStringList returns a map of string list
func (c Config) GetMapStringList(key string) map[string][]string {
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

	cfg.SetStringList("MyDonut", []string{"/media/disk1", "/media/disk2", "/media/badDisk99", "/media/badDisk100"})
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
