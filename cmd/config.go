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

package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/minio/minio/pkg/probe"
	homedir "github.com/mitchellh/go-homedir"
)

// miniocCustomConfigDir contains the whole path to config dir. Only access via get/set functions.
var miniocCustomConfigDir string

// setMiniocConfigDir - set a custom minio client config folder.
func setMiniocConfigDir(configDir string) {
	miniocCustomConfigDir = configDir
}

// getMiniocConfigDir - construct minio client config folder.
func getMiniocConfigDir() (string, *probe.Error) {
	if miniocCustomConfigDir != "" {
		return miniocCustomConfigDir, nil
	}
	homeDir, e := homedir.Dir()
	if e != nil {
		return "", probe.NewError(e)
	}
	var configDir string
	// For windows the path is slightly different
	if runtime.GOOS == "windows" {
		configDir = filepath.Join(homeDir, globalMINIOCConfigWindowsDir)
	} else {
		configDir = filepath.Join(homeDir, globalMINIOCConfigDir)
	}
	return configDir, nil
}

// mustGetMiniocConfigDir - construct minio client config folder or fail
func mustGetMiniocConfigDir() (configDir string) {
	configDir, err := getMiniocConfigDir()
	fatalIf(err.Trace(), "Unable to get miniocConfigDir.")

	return configDir
}

// createMiniocConfigDir - create minio client config folder
func createMiniocConfigDir() *probe.Error {
	p, err := getMiniocConfigDir()
	if err != nil {
		return err.Trace()
	}
	if e := os.MkdirAll(p, 0700); e != nil {
		return probe.NewError(e)
	}
	return nil
}

// getMiniocConfigPath - construct minio client configuration path
func getMiniocConfigPath() (string, *probe.Error) {
	if miniocCustomConfigDir != "" {
		return filepath.Join(miniocCustomConfigDir, globalMINIOCConfigFile), nil
	}
	dir, err := getMiniocConfigDir()
	if err != nil {
		return "", err.Trace()
	}
	return filepath.Join(dir, globalMINIOCConfigFile), nil
}

// mustGetMiniocConfigPath - similar to getMiniocConfigPath, ignores errors
func mustGetMiniocConfigPath() string {
	path, err := getMiniocConfigPath()
	fatalIf(err.Trace(), "Unable to get miniocConfigPath.")

	return path
}

// newMiniocConfig - initializes a new version '6' config.
func newMiniocConfig() *configV8 {
	cfg := newConfigV8()
	cfg.loadDefaults()
	return cfg
}

// loadMiniocConfigCached - returns loadMiniocConfig with a closure for config cache.
func loadMiniocConfigFactory() func() (*configV8, *probe.Error) {
	// Load once and cache in a closure.
	cfgCache, err := loadConfigV8()

	// loadMiniocConfig - reads configuration file and returns config.
	return func() (*configV8, *probe.Error) {
		return cfgCache, err
	}
}

// loadMiniocConfig - returns configuration, initialized later.
var loadMiniocConfig func() (*configV8, *probe.Error)

// saveMiniocConfig - saves configuration file and returns error if any.
func saveMiniocConfig(config *configV8) *probe.Error {
	if config == nil {
		return errInvalidArgument().Trace()
	}

	err := createMiniocConfigDir()
	if err != nil {
		return err.Trace(mustGetMiniocConfigDir())
	}

	// Save the config.
	if err := saveConfigV8(config); err != nil {
		return err.Trace(mustGetMiniocConfigPath())
	}

	// Refresh the config cache.
	loadMiniocConfig = loadMiniocConfigFactory()
	return nil
}

// isMiniocConfigExists returns err if config doesn't exist.
func isMiniocConfigExists() bool {
	configFile, err := getMiniocConfigPath()
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
func getHostConfig(alias string) (*hostConfigV8, *probe.Error) {
	miniocCfg, err := loadMiniocConfig()
	if err != nil {
		return nil, err.Trace(alias)
	}

	// if host is exact return quickly.
	if _, ok := miniocCfg.Hosts[alias]; ok {
		hostCfg := miniocCfg.Hosts[alias]
		return &hostCfg, nil
	}

	// return error if cannot be matched.
	return nil, errNoMatchingHost(alias).Trace(alias)
}

// mustGetHostConfig retrieves host specific configuration such as access keys, signature type.
func mustGetHostConfig(alias string) *hostConfigV8 {
	hostCfg, _ := getHostConfig(alias)
	return hostCfg
}

// expandAlias expands aliased URL if any match is found, returns as is otherwise.
func expandAlias(aliasedURL string) (alias string, urlStr string, hostCfg *hostConfigV8, err *probe.Error) {
	// Extract alias from the URL.
	alias, path := url2Alias(aliasedURL)

	// Find the matching alias entry and expand the URL.
	if hostCfg = mustGetHostConfig(alias); hostCfg != nil {
		return alias, urlJoinPath(hostCfg.URL, path), hostCfg, nil
	}
	return "", aliasedURL, nil, nil // No matching entry found. Return original URL as is.
}

// mustExpandAlias expands aliased URL if any match is found, returns as is otherwise.
func mustExpandAlias(aliasedURL string) (alias string, urlStr string, hostCfg *hostConfigV8) {
	alias, urlStr, hostCfg, _ = expandAlias(aliasedURL)
	return alias, urlStr, hostCfg
}
