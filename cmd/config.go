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
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/minio/go-homedir"
	"github.com/minio/mc/pkg/probe"
)

// mcCustomConfigDir contains the whole path to config dir. Only access via get/set functions.
var mcCustomConfigDir string

// setMcConfigDir - set a custom minio client config folder.
func setMcConfigDir(configDir string) {
	mcCustomConfigDir = configDir
}

// getMcConfigDir - construct minio client config folder.
func getMcConfigDir() (string, *probe.Error) {
	if mcCustomConfigDir != "" {
		return mcCustomConfigDir, nil
	}
	homeDir, e := homedir.Dir()
	if e != nil {
		return "", probe.NewError(e)
	}
	var configDir string
	// For windows the path is slightly different
	if runtime.GOOS == "windows" {
		configDir = filepath.Join(homeDir, globalMCConfigWindowsDir)
	} else {
		configDir = filepath.Join(homeDir, globalMCConfigDir)
	}
	return configDir, nil
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
	if mcCustomConfigDir != "" {
		return filepath.Join(mcCustomConfigDir, globalMCConfigFile), nil
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

// newMcConfig - initializes a new version '9' config.
func newMcConfig() *configV9 {
	cfg := newConfigV9()
	cfg.loadDefaults()
	return cfg
}

// loadMcConfigCached - returns loadMcConfig with a closure for config cache.
func loadMcConfigFactory() func() (*configV9, *probe.Error) {
	// Load once and cache in a closure.
	cfgCache, err := loadConfigV9()

	// loadMcConfig - reads configuration file and returns config.
	return func() (*configV9, *probe.Error) {
		return cfgCache, err
	}
}

// loadMcConfig - returns configuration, initialized later.
var loadMcConfig func() (*configV9, *probe.Error)

// saveMcConfig - saves configuration file and returns error if any.
func saveMcConfig(config *configV9) *probe.Error {
	if config == nil {
		return errInvalidArgument().Trace()
	}

	err := createMcConfigDir()
	if err != nil {
		return err.Trace(mustGetMcConfigDir())
	}

	// Save the config.
	if err := saveConfigV9(config); err != nil {
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
	return regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-_]+$").MatchString(alias)
}

// getHostConfig retrieves host specific configuration such as access keys, signature type.
func getHostConfig(alias string) (*hostConfigV9, *probe.Error) {
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
func mustGetHostConfig(alias string) *hostConfigV9 {
	hostCfg, _ := getHostConfig(alias)
	return hostCfg
}

// parse url usually obtained from env.
func parseEnvURL(envURL string) (*url.URL, string, string, *probe.Error) {
	u, e := url.Parse(envURL)
	if e != nil {
		return nil, "", "", probe.NewError(e).Trace(envURL)
	}

	var accessKey, secretKey string
	// Check if username:password is provided in URL, with no
	// access keys or secret we proceed and perform anonymous
	// requests.
	if u.User != nil {
		accessKey = u.User.Username()
		secretKey, _ = u.User.Password()
	}

	// Look for if URL has invalid values and return error.
	if !((u.Scheme == "http" || u.Scheme == "https") &&
		(u.Path == "/" || u.Path == "") && u.Opaque == "" &&
		u.ForceQuery == false && u.RawQuery == "" && u.Fragment == "") {
		return nil, "", "", errInvalidArgument().Trace(u.String())
	}

	// Now that we have validated the URL to be in expected style.
	u.User = nil

	return u, accessKey, secretKey, nil
}

const mcEnvHostsPrefix = "MC_HOSTS_"

func expandAliasFromEnv(envURL string) (*hostConfigV9, *probe.Error) {
	u, accessKey, secretKey, err := parseEnvURL(envURL)
	if err != nil {
		return nil, err.Trace(envURL)
	}

	return &hostConfigV9{
		URL:       u.String(),
		API:       "S3v4",
		AccessKey: accessKey,
		SecretKey: secretKey,
	}, nil
}

// expandAlias expands aliased URL if any match is found, returns as is otherwise.
func expandAlias(aliasedURL string) (alias string, urlStr string, hostCfg *hostConfigV9, err *probe.Error) {
	// Extract alias from the URL.
	alias, path := url2Alias(aliasedURL)

	if envConfig, ok := os.LookupEnv(mcEnvHostsPrefix + alias); ok {
		hostCfg, err = expandAliasFromEnv(envConfig)
		if err != nil {
			return "", "", nil, err.Trace(aliasedURL)
		}
		return alias, urlJoinPath(hostCfg.URL, path), hostCfg, nil
	}

	// Find the matching alias entry and expand the URL.
	if hostCfg = mustGetHostConfig(alias); hostCfg != nil {
		return alias, urlJoinPath(hostCfg.URL, path), hostCfg, nil
	}
	return "", aliasedURL, nil, nil // No matching entry found. Return original URL as is.
}

// mustExpandAlias expands aliased URL if any match is found, returns as is otherwise.
func mustExpandAlias(aliasedURL string) (alias string, urlStr string, hostCfg *hostConfigV9) {
	alias, urlStr, hostCfg, _ = expandAlias(aliasedURL)
	return alias, urlStr, hostCfg
}
