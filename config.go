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
	"regexp"
	"runtime"

	"github.com/minio/minio-xl/pkg/probe"
	"github.com/minio/minio/pkg/user"
)

// mcCustomConfigDir contains the whole path to config dir. Only access via get/set functions.
var mcCustomConfigDir string

// mcCustomConfigPath contains the whole path to config file. Only access via get/set functions.
var mcCustomConfigPath string

// setMcConfigDir - set a custom minio client config folder.
func setMcConfigDir(configDir string) {
	mcCustomConfigDir = configDir
}

// getMcConfigDir - construct minio client config folder.
func getMcConfigDir() (string, *probe.Error) {
	if mcCustomConfigDir != "" {
		return mcCustomConfigDir, nil
	}
	homeDir, e := user.HomeDir()
	if e != nil {
		return "", probe.NewError(e)
	}
	// For windows the path is slightly different
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(homeDir, globalMCConfigWindowsDir), nil
	default:
		return filepath.Join(homeDir, globalMCConfigDir), nil
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
	if e := os.MkdirAll(p, 0700); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// getMcConfigPath - construct minio client configuration path
func getMcConfigPath() (string, *probe.Error) {
	if mcCustomConfigPath != "" {
		return mcCustomConfigPath, nil
	}
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

// newMcConfig - initializes a new version '6' config.
func newMcConfig() *configV7 {
	cfg := newConfigV7()
	cfg.loadDefaults()
	return cfg
}

// loadMcConfigCached - returns loadMcConfig with a closure for config cache.
func loadMcConfigFactory() func() (*configV7, *probe.Error) {
	// Load once and cache in a closure.
	cfgCache, err := loadConfigV7()

	// loadMcConfig - reads configuration file and returns config.
	return func() (*configV7, *probe.Error) {
		return cfgCache, err
	}
}

var loadMcConfig = loadMcConfigFactory()

// saveMcConfig - saves configuration file and returns error if any.
func saveMcConfig(config *configV7) *probe.Error {
	if config == nil {
		return errInvalidArgument().Trace()
	}

	err := createMcConfigDir()
	if err != nil {
		return err.Trace(mustGetMcConfigDir())
	}

	// Save the config.
	if err := saveConfigV7(config); err != nil {
		return err.Trace(mustGetMcConfigPath())
	}

	// Refresh the config cache.
	loadMcConfig = loadMcConfigFactory()
	return nil
}

// isMcConfigExists returns err if config doesn't exist.
func isMcConfigExists() bool {
	configFile, err := getMcConfigPath()
	if err != nil {
		return false
	}
	if _, e := os.Stat(configFile); e != nil {
		return false
	}
	return true
}

// isValidAlias - Check if alias valid.
func isValidAlias(alias string) bool {
	return regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-]+$").MatchString(alias)
}

// getHostConfig retrieves host specific configuration such as access keys, signature type.
func getHostConfig(alias string) (*hostConfigV7, *probe.Error) {
	mcCfg, err := loadMcConfig()
	if err != nil {
		return nil, err.Trace(alias)
	}

	// if host is exact return quickly.
	if _, ok := mcCfg.Hosts[alias]; ok {
		hostCfg := mcCfg.Hosts[alias]
		return &hostCfg, nil
	}

	// return error if cannot be matched.
	return nil, errNoMatchingHost(alias).Trace(alias)
}

// mustGetHostConfig retrieves host specific configuration such as access keys, signature type.
func mustGetHostConfig(alias string) *hostConfigV7 {
	hostCfg, _ := getHostConfig(alias)
	return hostCfg
}

// expandAlias expands aliased URL if any match is found, returns as is otherwise.
func expandAlias(aliasedURL string) (alias string, urlStr string, hostCfg *hostConfigV7, err *probe.Error) {
	// Extract alias from the URL.
	alias, path := url2Alias(aliasedURL)

	// Find the matching alias entry and expand the URL.
	if hostCfg = mustGetHostConfig(alias); hostCfg != nil {
		return alias, urlJoinPath(hostCfg.URL, path), hostCfg, nil
	}
	return "", aliasedURL, nil, nil // No matching entry found. Return original URL as is.
}

// mustExpandAlias expands aliased URL if any match is found, returns as is otherwise.
func mustExpandAlias(aliasedURL string) (alias string, urlStr string, hostCfg *hostConfigV7) {
	alias, urlStr, hostCfg, _ = expandAlias(aliasedURL)
	return alias, urlStr, hostCfg
}
