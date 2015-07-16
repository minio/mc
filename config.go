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
	"path/filepath"
	"runtime"
	"sync"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/quick"
	"github.com/minio/minio/pkg/iodine"
)

type configV1 struct {
	Version string
	Aliases map[string]string
	Hosts   map[string]*hostConfig
}

// cached variables should *NEVER* be accessed directly from outside this file.
var cache sync.Pool

// customConfigDir contains the whole path to config dir. Only access via get/set functions.
var mcCustomConfigDir string

// setMcConfigDir - construct minio client config folder.
func setMcConfigDir(configDir string) {
	mcCustomConfigDir = configDir
}

// getMcConfigDir - construct minio client config folder.
func getMcConfigDir() (string, error) {
	if mcCustomConfigDir != "" {
		return mcCustomConfigDir, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", NewIodine(iodine.New(err, nil))
	}
	// For windows the path is slightly different
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(u.HomeDir, mcConfigWindowsDir), nil
	default:
		return filepath.Join(u.HomeDir, mcConfigDir), nil
	}
}

// mustGetMcConfigDir - construct minio client config folder or fail
func mustGetMcConfigDir() (configDir string) {
	configDir, err := getMcConfigDir()
	if err != nil {
		console.Fatalf("Unable to determine default configuration folder. %s\n", NewIodine(iodine.New(err, nil)))
	}
	return configDir
}

// createMcConfigDir - create minio client config folder
func createMcConfigDir() error {
	p, err := getMcConfigDir()
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}
	err = os.MkdirAll(p, 0700)
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}
	return nil
}

// getMcConfigPath - construct minio client configuration path
func getMcConfigPath() (string, error) {
	dir, err := getMcConfigDir()
	if err != nil {
		return "", NewIodine(iodine.New(err, nil))
	}
	return filepath.Join(dir, mcConfigFile), nil
}

// mustGetMcConfigPath - similar to getMcConfigPath, ignores errors
func mustGetMcConfigPath() string {
	p, err := getMcConfigPath()
	if err != nil {
		console.Fatalf("Unable to determine default config path. %s\n", NewIodine(iodine.New(err, nil)))
	}
	return p
}

// getMcConfig - reads configuration file and returns config
func getMcConfig() (*configV1, error) {
	if !isMcConfigExists() {
		return nil, NewIodine(iodine.New(errInvalidArgument{}, nil))
	}

	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}

	// Cached in private global variable.
	if v := cache.Get(); v != nil { // Use previously cached config.
		return v.(quick.Config).Data().(*configV1), nil
	}

	conf := newConfigV1()
	qconf, err := quick.New(conf)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}

	err = qconf.Load(configFile)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}
	cache.Put(qconf)
	return qconf.Data().(*configV1), nil

}

// mustGetMcConfig - reads configuration file and returns configs, exits on error
func mustGetMcConfig() *configV1 {
	config, err := getMcConfig()
	if err != nil {
		console.Fatalf("Unable to retrieve mc configuration. %s\n", err)
	}
	return config
}

// isMcConfigExists xreturns err if config doesn't exist
func isMcConfigExists() bool {
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
		return NewIodine(iodine.New(err, nil))
	}
	configPath, err := getMcConfigPath()
	if err != nil {
		return NewIodine(iodine.New(err, nil))
	}
	if err := config.Save(configPath); err != nil {
		return NewIodine(iodine.New(err, nil))
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
	s3HostConf.AccessKeyID = globalAccessKeyID
	s3HostConf.SecretAccessKey = globalSecretAccessKey

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
	aliases["play"] = "https://play.minio.io:9000"
	aliases["dl"] = "https://dl.minio.io:9000"
	aliases["localhost"] = "http://localhost:9000"
	conf.Aliases = aliases
	config, err = quick.New(conf)
	if err != nil {
		return nil, NewIodine(iodine.New(err, nil))
	}

	return config, nil
}

// mustNewConfig instantiates a new config handler, exists upon error
func mustNewConfig() quick.Config {
	config, err := newConfig()
	if err != nil {
		console.Fatalf("Unable to instantiate a new config handler. %s\n", NewIodine(iodine.New(err, nil)))
	}
	return config
}
