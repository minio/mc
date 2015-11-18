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
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio-xl/pkg/quick"
)

type configV6 struct {
	Version string                `json:"version"`
	Aliases map[string]string     `json:"alias"`
	Hosts   map[string]hostConfig `json:"hosts"`
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
	u, err := userCurrent()
	if err != nil {
		return "", err.Trace()
	}
	// For windows the path is slightly different
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(u.HomeDir, globalMCConfigWindowsDir), nil
	default:
		return filepath.Join(u.HomeDir, globalMCConfigDir), nil
	}
}

// mustGetMcConfigDir - construct minio client config folder or fail
func mustGetMcConfigDir() (configDir string) {
	configDir, err := getMcConfigDir()
	fatalIf(err.Trace(), "Unable to get mcConfigDir.")

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
	return filepath.Join(dir, globalMCConfigFile), nil
}

// mustGetMcConfigPath - similar to getMcConfigPath, ignores errors
func mustGetMcConfigPath() string {
	path, err := getMcConfigPath()
	fatalIf(err.Trace(), "Unable to get mcConfigPath.")

	return path
}

// getMcConfig - reads configuration file and returns config
func getMcConfig() (*configV6, *probe.Error) {
	if !isMcConfigExists() {
		return nil, errInvalidArgument().Trace()
	}

	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, err.Trace()
	}

	// Cached in private global variable.
	if v := cache.Get(); v != nil { // Use previously cached config.
		return v.(quick.Config).Data().(*configV6), nil
	}

	conf := new(configV6)
	conf.Version = globalMCConfigVersion
	qconf, err := quick.New(conf)
	if err != nil {
		return nil, err.Trace()
	}

	err = qconf.Load(configFile)
	if err != nil {
		return nil, err.Trace()
	}
	cache.Put(qconf)
	return qconf.Data().(*configV6), nil
}

// mustGetMcConfig - reads configuration file and returns configs, exits on error
func mustGetMcConfig() *configV6 {
	config, err := getMcConfig()
	fatalIf(err.Trace(), "Unable to read mc configuration.")
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
	if config == nil {
		return errInvalidArgument().Trace()
	}
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

func newConfigV6() *configV6 {
	conf := new(configV6)
	conf.Version = globalMCConfigVersion
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]hostConfig)
	conf.Aliases = make(map[string]string)

	localHostConfig := hostConfig{
		AccessKeyID:     "",
		SecretAccessKey: "",
		API:             "S3v4",
	}

	s3HostConf := hostConfig{
		AccessKeyID:     globalAccessKeyID,
		SecretAccessKey: globalSecretAccessKey,
		API:             "S3v4",
	}

	googlHostConf := hostConfig{
		AccessKeyID:     globalAccessKeyID,
		SecretAccessKey: globalSecretAccessKey,
		API:             "S3v2",
	}

	// Your example host config
	exampleHostConf := hostConfig{
		AccessKeyID:     globalAccessKeyID,
		SecretAccessKey: globalSecretAccessKey,
		API:             "S3v4",
	}

	playHostConfig := hostConfig{
		AccessKeyID:     "",
		SecretAccessKey: "",
		API:             "S3v4",
	}

	dlHostConfig := hostConfig{
		AccessKeyID:     "",
		SecretAccessKey: "",
		API:             "S3v4",
	}

	conf.Hosts[globalExampleHostURL] = exampleHostConf
	conf.Hosts["localhost:9000"] = localHostConfig
	conf.Hosts["dl.minio.io:9000"] = dlHostConfig
	conf.Hosts["s3.amazonaws.com"] = s3HostConf
	conf.Hosts["play.minio.io:9000"] = playHostConfig
	conf.Hosts["storage.googleapis.com"] = googlHostConf

	aliases := make(map[string]string)
	aliases["s3"] = "https://s3.amazonaws.com"
	aliases["dl"] = "https://dl.minio.io:9000"
	aliases["gcs"] = "https://storage.googleapis.com"
	aliases["play"] = "https://play.minio.io:9000"
	aliases["local"] = "http://localhost:9000"
	conf.Aliases = aliases
	return conf
}

// newConfig - get new config interface
func newConfig() (config quick.Config, err *probe.Error) {
	config, err = quick.New(newConfigV6())
	if err != nil {
		return nil, err.Trace()
	}
	return config, nil
}

// fixConfigV6 - fix all the unnecessary glob URLs present in existing config version 6.
func fixConfigV6() {
	if !isMcConfigExists() {
		return
	}
	config, err := quick.New(newConfigV6())
	fatalIf(err.Trace(), "Unable to initialize config.")

	err = config.Load(mustGetMcConfigPath())
	fatalIf(err.Trace(), "Unable to load config.")

	if config.Data().(*configV6).Version == "6" {
		newConfig := new(configV6)
		newConfig.Aliases = make(map[string]string)
		newConfig.Hosts = make(map[string]hostConfig)
		newConfig.Version = "6"
		newConfig.Aliases = config.Data().(*configV6).Aliases
		for host, hostCfg := range config.Data().(*configV6).Hosts {
			if strings.Contains(host, "*s3*") || strings.Contains(host, "*.s3*") {
				console.Infoln("Found glob url, replacing " + host + " with s3.amazonaws.com")
				newConfig.Hosts["s3.amazonaws.com"] = hostCfg
				continue
			}
			if strings.Contains(host, "s3*") {
				console.Infoln("Found glob url, replacing " + host + " with s3.amazonaws.com")
				newConfig.Hosts["s3.amazonaws.com"] = hostCfg
				continue
			}
			if strings.Contains(host, "*amazonaws.com") || strings.Contains(host, "*.amazonaws.com") {
				console.Infoln("Found glob url, replacing " + host + " with s3.amazonaws.com")
				newConfig.Hosts["s3.amazonaws.com"] = hostCfg
				continue
			}
			if strings.Contains(host, "*storage.googleapis.com") {
				console.Infoln("Found glob url, replacing " + host + " with storage.googleapis.com")
				newConfig.Hosts["storage.googleapis.com"] = hostCfg
				continue
			}
			if strings.Contains(host, "localhost:*") {
				console.Infoln("Found glob url, replacing " + host + " with localhost:9000")
				newConfig.Hosts["localhost:9000"] = hostCfg
				continue
			}
			if strings.Contains(host, "127.0.0.1:*") {
				console.Infoln("Found glob url, replacing " + host + " with 127.0.0.1:9000")
				newConfig.Hosts["127.0.0.1:9000"] = hostCfg
				continue
			}
			newConfig.Hosts[host] = hostCfg
		}
		newConf, err := quick.New(newConfig)
		fatalIf(err.Trace(), "Unable to initialize newly fixed config.")

		err = newConf.Save(mustGetMcConfigPath())
		fatalIf(err.Trace(mustGetMcConfigPath()), "Unable to save newly fixed config path.")
	}
}

func migrateConfig() {
	// Migrate config V1 to V101
	migrateConfigV1ToV101()
	// Migrate config V101 to V2
	migrateConfigV101ToV2()
	// Migrate config V2 to V3
	migrateConfigV2ToV3()
	// Migrate config V3 to V4
	migrateConfigV3ToV4()
	// Migrate config V4 to V5
	migrateConfigV4ToV5()
	// Migrate config V5 to V6
	migrateConfigV5ToV6()
}

func fixConfig() {
	// Fix config V3
	fixConfigV3()
	// Fix config V6
	fixConfigV6()
}
