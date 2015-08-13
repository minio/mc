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

	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/internal/github.com/minio/minio/pkg/quick"
)

type configV1 struct {
	Version string
	Aliases map[string]string
	Hosts   map[string]hostConfig
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
func getMcConfigDir() (string, *probe.Error) {
	if mcCustomConfigDir != "" {
		return mcCustomConfigDir, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", probe.NewError(err)
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
	fatalIf(err)
	return configDir
}

// createMcConfigDir - create minio client config folder
func createMcConfigDir() *probe.Error {
	p, err := getMcConfigDir()
	if err != nil {
		return err.Trace()
	}
	if err := os.MkdirAll(p, 0700); err != nil {
		return probe.NewError(err)
	}
	return nil
}

// getMcConfigPath - construct minio client configuration path
func getMcConfigPath() (string, *probe.Error) {
	dir, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}
	return filepath.Join(dir, mcConfigFile), nil
}

// mustGetMcConfigPath - similar to getMcConfigPath, ignores errors
func mustGetMcConfigPath() string {
	path, err := getMcConfigPath()
	fatalIf(err)
	return path
}

// getMcConfig - reads configuration file and returns config
func getMcConfig() (*configV1, *probe.Error) {
	if !isMcConfigExists() {
		return nil, probe.NewError(errInvalidArgument{})
	}

	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, err.Trace()
	}

	// Cached in private global variable.
	if v := cache.Get(); v != nil { // Use previously cached config.
		return v.(quick.Config).Data().(*configV1), nil
	}

	conf := newConfigV1()
	qconf, err := quick.New(conf)
	if err != nil {
		return nil, err.Trace()
	}

	err = qconf.Load(configFile)
	if err != nil {
		return nil, err.Trace()
	}
	cache.Put(qconf)
	return qconf.Data().(*configV1), nil

}

// mustGetMcConfig - reads configuration file and returns configs, exits on error
func mustGetMcConfig() *configV1 {
	config, err := getMcConfig()
	fatalIf(err)
	return config
}

// isMcConfigExists xreturns err if config doesn't exist
func isMcConfigExists() bool {
	configFile, err := getMcConfigPath()
	if err != nil {
		return false
	}
	if _, err := os.Stat(configFile); err != nil {
		return false
	}
	return true
}

// writeConfig - write configuration file
func writeConfig(config quick.Config) *probe.Error {
	err := createMcConfigDir()
	if err != nil {
		return err.Trace()
	}
	configPath, err := getMcConfigPath()
	if err != nil {
		return err.Trace()
	}
	if err := config.Save(configPath); err != nil {
		return err.Trace()
	}
	return nil
}

func migrateConfig() {
	// Migrate session V1 to V101
	migrateConfigV1ToV101()
}

func migrateConfigV1ToV101() {
	if !isMcConfigExists() {
		return
	}
	conf := newConfigV1()
	config, err := quick.New(conf)
	fatalIf(err)
	err = config.Load(mustGetMcConfigPath())
	fatalIf(err)

	conf = config.Data().(*configV1)
	// version is the same return
	if conf.Version == mcCurrentConfigVersion {
		return
	}
	conf.Version = mcCurrentConfigVersion

	localHostConfig := hostConfig{}
	localHostConfig.AccessKeyID = ""
	localHostConfig.SecretAccessKey = ""

	s3HostConf := hostConfig{}
	s3HostConf.AccessKeyID = globalAccessKeyID
	s3HostConf.SecretAccessKey = globalSecretAccessKey

	if _, ok := conf.Hosts["localhost:*"]; !ok {
		conf.Hosts["localhost:*"] = localHostConfig
	}
	if _, ok := conf.Hosts["127.0.0.1:*"]; !ok {
		conf.Hosts["127.0.0.1:*"] = localHostConfig
	}
	if _, ok := conf.Hosts["*.s3*.amazonaws.com"]; !ok {
		conf.Hosts["*.s3*.amazonaws.com"] = s3HostConf
	}

	newConfig, perr := quick.New(conf)
	fatalIf(perr)
	perr = newConfig.Save(mustGetMcConfigPath())
	fatalIf(perr)

	console.Infof("Successfully migrated %s from version: %s to version: %s\n", mustGetMcConfigPath(), mcPreviousConfigVersion, mcCurrentConfigVersion)
}

// newConfigV1() - get new config version 1.0.0
func newConfigV1() *configV1 {
	conf := new(configV1)
	conf.Version = mcPreviousConfigVersion
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]hostConfig)
	conf.Aliases = make(map[string]string)
	return conf
}

// newConfigV101() - get new config version 1.0.1
func newConfigV101() *configV1 {
	conf := new(configV1)
	conf.Version = mcCurrentConfigVersion
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]hostConfig)
	conf.Aliases = make(map[string]string)

	localHostConfig := hostConfig{}
	localHostConfig.AccessKeyID = ""
	localHostConfig.SecretAccessKey = ""

	s3HostConf := hostConfig{}
	s3HostConf.AccessKeyID = globalAccessKeyID
	s3HostConf.SecretAccessKey = globalSecretAccessKey

	// Your example host config
	exampleHostConf := hostConfig{}
	exampleHostConf.AccessKeyID = globalAccessKeyID
	exampleHostConf.SecretAccessKey = globalSecretAccessKey

	playHostConfig := hostConfig{}
	playHostConfig.AccessKeyID = ""
	playHostConfig.SecretAccessKey = ""

	dlHostConfig := hostConfig{}
	dlHostConfig.AccessKeyID = ""
	dlHostConfig.SecretAccessKey = ""

	conf.Hosts[exampleHostURL] = exampleHostConf
	conf.Hosts["localhost:*"] = localHostConfig
	conf.Hosts["127.0.0.1:*"] = localHostConfig
	conf.Hosts["s3*.amazonaws.com"] = s3HostConf
	conf.Hosts["*.s3*.amazonaws.com"] = s3HostConf
	conf.Hosts["play.minio.io:9000"] = playHostConfig
	conf.Hosts["dl.minio.io:9000"] = dlHostConfig

	aliases := make(map[string]string)
	aliases["s3"] = "https://s3.amazonaws.com"
	aliases["play"] = "https://play.minio.io:9000"
	aliases["dl"] = "https://dl.minio.io:9000"
	aliases["localhost"] = "http://localhost:9000"
	conf.Aliases = aliases

	return conf
}

// newConfig - get new config interface
func newConfig() (config quick.Config, err *probe.Error) {
	conf := newConfigV101()
	config, err = quick.New(conf)
	if err != nil {
		return nil, err.Trace()
	}
	return config, nil
}
