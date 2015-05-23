/*
 * Minio Client (C) 2015 Minio, Inc.
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
	"os/user"
	"path"
	"runtime"

	"github.com/minio/mc/pkg/quick"
	"github.com/minio/minio/pkg/iodine"
)

type configV1 struct {
	Version string
	Aliases map[string]string
	Hosts   map[string]*hostConfig
}

// cached variables should *NEVER* be accessed directly from outside this file.
var cache struct {
	config       quick.Config
	configLoaded bool // set to true if cache is valid.
}

// customConfigDir used internally only by test functions
var customConfigDir string

// getMcConfigDir - construct minio client config folder
func getMcConfigDir() (string, error) {
	if customConfigDir != "" {
		// For windows the path is slightly different
		switch runtime.GOOS {
		case "windows":
			return path.Join(customConfigDir, mcConfigWindowsDir), nil
		default:
			return path.Join(customConfigDir, mcConfigDir), nil
		}
	}
	u, err := user.Current()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	// For windows the path is slightly different
	switch runtime.GOOS {
	case "windows":
		return path.Join(u.HomeDir, mcConfigWindowsDir), nil
	default:
		return path.Join(u.HomeDir, mcConfigDir), nil
	}
}

// createMcConfigDir - create minio client config folder
func createMcConfigDir() error {
	p, err := getMcConfigDir()
	if err != nil {
		return iodine.New(err, nil)
	}
	err = os.MkdirAll(p, 0700)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// getMcConfigPath - construct minio client configuration path
func getMcConfigPath() (string, error) {
	dir, err := getMcConfigDir()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return path.Join(dir, mcConfigFile), nil
}

// mustGetMcConfigPath - similar to getMcConfigPath, ignores errors
func mustGetMcConfigPath() string {
	p, _ := getMcConfigPath()
	return p
}

// getMcConfig - reads configuration file and returns config
func getMcConfig() (*configV1, error) {
	if !isMcConfigExist() {
		return nil, iodine.New(errInvalidArgument{}, nil)
	}

	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	// Cached in private global variable.
	if cache.configLoaded { // Use previously cached config.
		return cache.config.Data().(*configV1), nil
	}

	conf := newConfigV1()
	cache.config, err = quick.New(conf)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	err = cache.config.Load(configFile)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	cache.configLoaded = true

	return cache.config.Data().(*configV1), nil

}

// isMcConfigExist returns err if config doesn't exist
func isMcConfigExist() bool {
	configFile, err := getMcConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(configFile)
	if err != nil {
		return false
	}
	return true
}

// writeConfig - write configuration file
func writeConfig(config quick.Config) error {
	err := createMcConfigDir()
	if err != nil {
		return iodine.New(err, nil)
	}
	configPath, err := getMcConfigPath()
	if err != nil {
		return iodine.New(err, nil)
	}
	if err := config.Save(configPath); err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// newConfigV1() - get new config version 1.0
func newConfigV1() *configV1 {
	conf := new(configV1)
	conf.Version = mcCurrentConfigVersion
	// make sure to allocate map's otherwise Golang
	// exists silently without providing any errors
	conf.Hosts = make(map[string]*hostConfig)
	conf.Aliases = make(map[string]string)
	return conf
}

// newConfig - get new config interface
func newConfig() (config quick.Config, err error) {
	conf := newConfigV1()
	s3HostConf := new(hostConfig)
	s3HostConf.AccessKeyID = ""
	s3HostConf.SecretAccessKey = ""

	playHostConfig := new(hostConfig)
	playHostConfig.AccessKeyID = ""
	playHostConfig.SecretAccessKey = ""

	dlHostConfig := new(hostConfig)
	dlHostConfig.AccessKeyID = ""
	dlHostConfig.SecretAccessKey = ""

	// Your example host config
	exampleHostConf := new(hostConfig)
	exampleHostConf.AccessKeyID = globalAccessKeyID
	exampleHostConf.SecretAccessKey = globalSecretAccessKey

	conf.Hosts[exampleHostURL] = exampleHostConf
	conf.Hosts["s3*.amazonaws.com"] = s3HostConf
	conf.Hosts["play.minio.io:9000"] = playHostConfig
	conf.Hosts["dl.minio.io:9000"] = dlHostConfig

	aliases := make(map[string]string)
	aliases["s3"] = "https://s3.amazonaws.com"
	aliases["play"] = "http://play.minio.io:9000"
	aliases["dl"] = "http://dl.minio.io:9000"
	aliases["localhost"] = "http://localhost:9000"
	conf.Aliases = aliases
	config, err = quick.New(conf)
	if err != nil {
		return nil, iodine.New(err, nil)
	}

	return config, nil
}
