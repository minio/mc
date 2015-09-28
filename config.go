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
	"github.com/minio/minio/pkg/probe"
	"github.com/minio/minio/pkg/quick"
)

type configV5 struct {
	Version string                `json:"version"`
	Aliases map[string]string     `json:"alias"`
	Hosts   map[string]hostConfig `json:"hosts"`
}

type configV4 struct {
	Version string            `json:"version"`
	Aliases map[string]string `json:"alias"`
	Hosts   map[string]struct {
		AccessKeyID     string `json:"accessKeyId"`
		SecretAccessKey string `json:"secretAccessKey"`
		Signature       string `json:"signature"`
	} `json:"hosts"`
}

type configV3 struct {
	Version string            `json:"version"`
	Aliases map[string]string `json:"alias"`
	// custom anonymous struct is necessary from version to 3 to version 4
	// since hostConfig{} has changed to camelcase fields for unmarshal
	Hosts map[string]struct {
		AccessKeyID     string `json:"access-key-id"`
		SecretAccessKey string `json:"secret-access-key"`
	} `json:"hosts"`
}

type configV2 struct {
	Version string
	Aliases map[string]string
	// custom anonymous struct is necessary from version to 2 to version 3
	// since hostConfig{} has changed to lower case fields for unmarshal
	Hosts map[string]struct {
		AccessKeyID     string
		SecretAccessKey string
	}
}

// for backward compatibility
type configV101 configV2
type configV1 configV2

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
func getMcConfig() (*configV5, *probe.Error) {
	if !isMcConfigExists() {
		return nil, errInvalidArgument().Trace()
	}

	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, err.Trace()
	}

	// Cached in private global variable.
	if v := cache.Get(); v != nil { // Use previously cached config.
		return v.(quick.Config).Data().(*configV5), nil
	}

	conf := new(configV5)
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
	return qconf.Data().(*configV5), nil
}

// mustGetMcConfig - reads configuration file and returns configs, exits on error
func mustGetMcConfig() *configV5 {
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
}

func fixConfig() {
	// Fix config V3
	fixConfigV3()
}

// Migrate from config version ‘1.0’ to ‘1.0.1’
func migrateConfigV1ToV101() {
	if !isMcConfigExists() {
		return
	}
	mcConfigV1, err := quick.Load(mustGetMcConfigPath(), newConfigV1())
	fatalIf(err.Trace(), "Unable to load config version ‘1’.")

	// update to newer version
	if mcConfigV1.Version() == "1.0.0" {
		confV101 := mcConfigV1.Data().(*configV1)
		confV101.Version = "1.0.1"

		localHostConfig := struct {
			AccessKeyID     string
			SecretAccessKey string
		}{}
		localHostConfig.AccessKeyID = ""
		localHostConfig.SecretAccessKey = ""

		s3HostConf := struct {
			AccessKeyID     string
			SecretAccessKey string
		}{}
		s3HostConf.AccessKeyID = globalAccessKeyID
		s3HostConf.SecretAccessKey = globalSecretAccessKey

		if _, ok := confV101.Hosts["localhost:*"]; !ok {
			confV101.Hosts["localhost:*"] = localHostConfig
		}
		if _, ok := confV101.Hosts["127.0.0.1:*"]; !ok {
			confV101.Hosts["127.0.0.1:*"] = localHostConfig
		}
		if _, ok := confV101.Hosts["*.s3*.amazonaws.com"]; !ok {
			confV101.Hosts["*.s3*.amazonaws.com"] = s3HostConf
		}

		mcNewConfigV101, err := quick.New(confV101)
		fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘1.0.1’.")

		err = mcNewConfigV101.Save(mustGetMcConfigPath())
		fatalIf(err.Trace(), "Unable to save config version ‘1.0.1’.")

		console.Infof("Successfully migrated %s from version ‘1.0.0’ to version ‘1.0.1’.\n", mustGetMcConfigPath())
	}
}

// Migrate from config ‘1.0.1’ to ‘2’
func migrateConfigV101ToV2() {
	if !isMcConfigExists() {
		return
	}
	mcConfigV101, err := quick.Load(mustGetMcConfigPath(), newConfigV101())
	fatalIf(err.Trace(), "Unable to load config version ‘1.0.1’.")

	// update to newer version
	if mcConfigV101.Version() == "1.0.1" {
		confV2 := mcConfigV101.Data().(*configV101)
		confV2.Version = "2"

		mcNewConfigV2, err := quick.New(confV2)
		fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘2’.")

		err = mcNewConfigV2.Save(mustGetMcConfigPath())
		fatalIf(err.Trace(), "Unable to save config version ‘2’.")

		console.Infof("Successfully migrated %s from version ‘1.0.1’ to version ‘2’.\n", mustGetMcConfigPath())
	}
}

// Migrate from config ‘2’ to ‘3’
func migrateConfigV2ToV3() {
	if !isMcConfigExists() {
		return
	}
	mcConfigV2, err := quick.Load(mustGetMcConfigPath(), newConfigV2())
	fatalIf(err.Trace(), "Unable to load mc config V2.")

	// update to newer version
	if mcConfigV2.Version() == "2" {
		confV3 := new(configV3)
		confV3.Aliases = mcConfigV2.Data().(*configV2).Aliases
		confV3.Hosts = make(map[string]struct {
			AccessKeyID     string `json:"access-key-id"`
			SecretAccessKey string `json:"secret-access-key"`
		})
		for host, hostConf := range mcConfigV2.Data().(*configV2).Hosts {
			newHostConf := struct {
				AccessKeyID     string `json:"access-key-id"`
				SecretAccessKey string `json:"secret-access-key"`
			}{}
			newHostConf.AccessKeyID = hostConf.AccessKeyID
			newHostConf.SecretAccessKey = hostConf.SecretAccessKey
			confV3.Hosts[host] = newHostConf
		}
		confV3.Version = "3"

		mcNewConfigV3, err := quick.New(confV3)
		fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘3’.")

		err = mcNewConfigV3.Save(mustGetMcConfigPath())
		fatalIf(err.Trace(), "Unable to save config version ‘3’.")

		console.Infof("Successfully migrated %s from version ‘2’ to version ‘3’.\n", mustGetMcConfigPath())
	}
}

// Migrate from config version ‘3’ to ‘4’
func migrateConfigV3ToV4() {
	if !isMcConfigExists() {
		return
	}
	mcConfigV3, err := quick.Load(mustGetMcConfigPath(), newConfigV3())
	fatalIf(err.Trace(), "Unable to load mc config V2.")

	// update to newer version
	if mcConfigV3.Version() == "3" {
		confV4 := new(configV4)
		confV4.Aliases = mcConfigV3.Data().(*configV3).Aliases
		confV4.Hosts = make(map[string]struct {
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
			Signature       string `json:"signature"`
		})
		for host, hostConf := range mcConfigV3.Data().(*configV3).Hosts {
			confV4.Hosts[host] = struct {
				AccessKeyID     string `json:"accessKeyId"`
				SecretAccessKey string `json:"secretAccessKey"`
				Signature       string `json:"signature"`
			}{
				AccessKeyID:     hostConf.AccessKeyID,
				SecretAccessKey: hostConf.SecretAccessKey,
				Signature:       "v4",
			}
		}
		confV4.Version = "4"

		mcNewConfigV4, err := quick.New(confV4)
		fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘4’.")

		err = mcNewConfigV4.Save(mustGetMcConfigPath())
		fatalIf(err.Trace(), "Unable to save config version ‘4’.")

		console.Infof("Successfully migrated %s from version ‘3’ to version ‘4’.\n", mustGetMcConfigPath())
	}
}

// Migrate config version ‘4’ to ‘5’
func migrateConfigV4ToV5() {
	if !isMcConfigExists() {
		return
	}
	mcConfigV4, err := quick.Load(mustGetMcConfigPath(), newConfigV4())
	fatalIf(err.Trace(), "Unable to load mc config V2.")

	// update to newer version
	if mcConfigV4.Version() == "4" {
		confV5 := new(configV5)
		confV5.Aliases = mcConfigV4.Data().(*configV4).Aliases
		confV5.Hosts = make(map[string]hostConfig)
		for host, hostConf := range mcConfigV4.Data().(*configV4).Hosts {
			confV5.Hosts[host] = hostConfig{
				AccessKeyID:     hostConf.AccessKeyID,
				SecretAccessKey: hostConf.SecretAccessKey,
				API:             "S3v4",
			}
		}
		confV5.Version = globalMCConfigVersion

		mcNewConfigV5, err := quick.New(confV5)
		fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘5’.")

		err = mcNewConfigV5.Save(mustGetMcConfigPath())
		fatalIf(err.Trace(), "Unable to save config version ‘5’.")

		console.Infof("Successfully migrated %s from version ‘4’ to version ‘5’.\n", mustGetMcConfigPath())
	}
}

// Fix config version ‘3’, by removing broken struct tags
func fixConfigV3() {
	if !isMcConfigExists() {
		return
	}
	// brokenConfigV3 broken config between version 3
	type brokenConfigV3 struct {
		Version string
		ACL     string
		Access  string
		Aliases map[string]string
		Hosts   map[string]struct {
			AccessKeyID     string
			SecretAccessKey string
		}
	}
	conf := new(brokenConfigV3)
	conf.Aliases = make(map[string]string)
	conf.Hosts = make(map[string]struct {
		AccessKeyID     string
		SecretAccessKey string
	})

	mcConfigV3, err := quick.Load(mustGetMcConfigPath(), conf)
	fatalIf(err.Trace(), "Unable to load config.")

	// Update to newer version
	if len(mcConfigV3.Data().(*brokenConfigV3).Aliases) != 0 || mcConfigV3.Data().(*brokenConfigV3).ACL != "" || mcConfigV3.Data().(*brokenConfigV3).Access != "" && mcConfigV3.Version() == "3" {
		confV3 := new(configV3)
		confV3.Aliases = mcConfigV3.Data().(*brokenConfigV3).Aliases
		confV3.Hosts = make(map[string]struct {
			AccessKeyID     string `json:"access-key-id"`
			SecretAccessKey string `json:"secret-access-key"`
		})
		for host, hostConf := range mcConfigV3.Data().(*brokenConfigV3).Hosts {
			newHostConf := struct {
				AccessKeyID     string `json:"access-key-id"`
				SecretAccessKey string `json:"secret-access-key"`
			}{}
			newHostConf.AccessKeyID = hostConf.AccessKeyID
			newHostConf.SecretAccessKey = hostConf.SecretAccessKey
			confV3.Hosts[host] = newHostConf
		}
		confV3.Version = globalMCConfigVersion

		mcNewConfigV3, err := quick.New(confV3)
		fatalIf(err.Trace(), "Unable to initialize quick config for config version ‘3’.")

		err = mcNewConfigV3.Save(mustGetMcConfigPath())
		fatalIf(err.Trace(), "Unable to save config version ‘3’.")

		console.Infof("Successfully fixed %s broken config for version ‘3’.\n", mustGetMcConfigPath())
	}
}

// newConfigV1() - get new config version 1.0.0
func newConfigV1() *configV1 {
	conf := new(configV1)
	conf.Version = "1.0.0"
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]struct {
		AccessKeyID     string
		SecretAccessKey string
	})
	conf.Aliases = make(map[string]string)
	return conf
}

// newConfigV101() - get new config version 1.0.1
func newConfigV101() *configV101 {
	conf := new(configV101)
	conf.Version = "1.0.1"
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]struct {
		AccessKeyID     string
		SecretAccessKey string
	})
	conf.Aliases = make(map[string]string)
	return conf
}

// newConfigV2() - get new config version 2
func newConfigV2() *configV2 {
	conf := new(configV2)
	conf.Version = "2"
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]struct {
		AccessKeyID     string
		SecretAccessKey string
	})
	conf.Aliases = make(map[string]string)
	return conf
}

// newConfigV3 - get new config version 3
func newConfigV3() *configV3 {
	conf := new(configV3)
	conf.Version = "3"
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]struct {
		AccessKeyID     string `json:"access-key-id"`
		SecretAccessKey string `json:"secret-access-key"`
	})
	conf.Aliases = make(map[string]string)
	return conf
}

func newConfigV4() *configV4 {
	conf := new(configV4)
	conf.Version = "4"
	// make sure to allocate map's otherwise Golang
	// exits silently without providing any errors
	conf.Hosts = make(map[string]struct {
		AccessKeyID     string `json:"accessKeyId"`
		SecretAccessKey string `json:"secretAccessKey"`
		Signature       string `json:"signature"`
	})
	conf.Aliases = make(map[string]string)
	return conf
}

func newConfigV5() *configV5 {
	conf := new(configV5)
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
	config, err = quick.New(newConfigV5())
	if err != nil {
		return nil, err.Trace()
	}
	return config, nil
}
