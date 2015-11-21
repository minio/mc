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
	"sync"

	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio-xl/pkg/quick"
)

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
	conf, err := loadConfigV6()
	if err != nil {
		return nil, err.Trace()
	}
	return conf, nil
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

// newConfig - get new config interface.
func newConfig() (config quick.Config, err *probe.Error) {
	config, err = quick.New(newConfigV6())
	if err != nil {
		return nil, err.Trace()
	}
	return config, nil
}
