/*
 * Minimalist Object Storage, (C) 2014,2015 Minio, Inc.
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

package main

import (
	"os"
	"path"

	"encoding/json"
	"io/ioutil"

	"github.com/minio-io/cli"
)

type nodeConfig struct {
	ActiveDisks   []string
	InactiveDisks []string
}

type donutConfig struct {
	Node map[string]nodeConfig
}

type mcDonutConfig struct {
	Donuts map[string]donutConfig
}

// Is alphanumeric?
func isalnum(c rune) bool {
	return '0' <= c && c <= '9' || 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

// isValidDonutName - verify donutName to be valid
func isValidDonutName(donutName string) bool {
	if len(donutName) > 1024 || len(donutName) == 0 {
		return false
	}
	for _, char := range donutName {
		if isalnum(char) {
			continue
		}
		switch char {
		case '-':
		case '.':
		case '_':
		case '~':
			continue
		default:
			return false
		}
	}
	return true
}

func getDonutConfigFilename() string {
	return path.Join(getMcConfigDir(), "donuts.json")
}

// saveDonutConfig writes configuration data in json format to donut config file.
func saveDonutConfig(donutConfigData *mcDonutConfig) error {
	jsonConfig, err := json.MarshalIndent(donutConfigData, "", "\t")
	if err != nil {
		return err
	}

	err = os.MkdirAll(getMcConfigDir(), 0755)
	if !os.IsExist(err) && err != nil {
		return err
	}

	configFile, err := os.OpenFile(getDonutConfigFilename(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer configFile.Close()

	_, err = configFile.Write(jsonConfig)
	if err != nil {
		return err
	}
	return nil
}

func loadDonutConfig() (donutConfigData *mcDonutConfig, err error) {
	configFile := getDonutConfigFilename()
	_, err = os.Stat(configFile)
	if err != nil {
		return nil, err
	}

	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(configBytes, &donutConfigData)
	if err != nil {
		return nil, err
	}

	return donutConfigData, nil
}

func newDonutConfig(donutName string) (*mcDonutConfig, error) {
	mcDonutConfigData := new(mcDonutConfig)
	mcDonutConfigData.Donuts = make(map[string]donutConfig)
	mcDonutConfigData.Donuts[donutName] = donutConfig{
		Node: make(map[string]nodeConfig),
	}
	mcDonutConfigData.Donuts[donutName].Node["localhost"] = nodeConfig{
		ActiveDisks:   make([]string, 0),
		InactiveDisks: make([]string, 0),
	}
	return mcDonutConfigData, nil
}

// doMakeDonutCmd creates a new donut
func doMakeDonutCmd(c *cli.Context) {
	if len(c.Args()) != 1 {
		fatal("Invalid args")
	}
	donutName := c.Args().First()
	if !isValidDonutName(donutName) {
		fatal("Invalid donutName")
	}
	mcDonutConfigData, err := loadDonutConfig()
	if os.IsNotExist(err) {
		mcDonutConfigData, err = newDonutConfig(donutName)
		if err != nil {
			fatal(err.Error())
		}
		if err := saveDonutConfig(mcDonutConfigData); err != nil {
			fatal(err.Error())
		}
	} else if err != nil {
		fatal(err.Error())
	}
	mcDonutConfigData.Donuts[donutName] = donutConfig{
		Node: make(map[string]nodeConfig),
	}
	mcDonutConfigData.Donuts[donutName].Node["localhost"] = nodeConfig{
		ActiveDisks:   make([]string, 0),
		InactiveDisks: make([]string, 0),
	}
	if err := saveDonutConfig(mcDonutConfigData); err != nil {
		fatal(err.Error())
	}
}
