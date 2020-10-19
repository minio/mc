/*
 * MinIO Client (C) 2015-2020 MinIO, Inc.
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
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/minio/mc/pkg/probe"

	"github.com/mitchellh/go-homedir"
)

// mcCustomConfigDir contains the whole path to config dir. Only access via get/set functions.
var mcCustomConfigDir string

// setMcConfigDir - set a custom MinIO Client config folder.
func setMcConfigDir(configDir string) {
	mcCustomConfigDir = configDir
}

// getMcConfigDir - construct MinIO Client config folder.
func getMcConfigDir() (string, *probe.Error) {
	if mcCustomConfigDir != "" {
		return mcCustomConfigDir, nil
	}
	homeDir, e := homedir.Dir()
	if e != nil {
		return "", probe.NewError(e)
	}
	configDir := filepath.Join(homeDir, defaultMCConfigDir())
	return configDir, nil
}

// Return default default mc config directory.
// Generally you want to use getMcConfigDir which returns custom overrides.
func defaultMCConfigDir() string {
	if runtime.GOOS == "windows" {
		// For windows the path is slightly different
		cmd := filepath.Base(os.Args[0])
		if strings.HasSuffix(strings.ToLower(cmd), ".exe") {
			cmd = cmd[:strings.LastIndex(cmd, ".")]
		}
		return fmt.Sprintf("%s\\", cmd)
	}
	return fmt.Sprintf(".%s/", filepath.Base(os.Args[0]))
}

// mustGetMcConfigDir - construct MinIO Client config folder or fail
func mustGetMcConfigDir() (configDir string) {
	configDir, err := getMcConfigDir()
	fatalIf(err.Trace(), "Unable to get mcConfigDir.")

	return configDir
}

// createMcConfigDir - create MinIO Client config folder
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

// getMcConfigPath - construct MinIO Client configuration path
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

// newMcConfig - initializes a new version '10' config.
func newMcConfig() *configV10 {
	cfg := newConfigV10()
	cfg.loadDefaults()
	return cfg
}

// loadMcConfigCached - returns loadMcConfig with a closure for config cache.
func loadMcConfigFactory() func() (*configV10, *probe.Error) {
	// Load once and cache in a closure.
	cfgCache, err := loadConfigV10()

	// loadMcConfig - reads configuration file and returns config.
	return func() (*configV10, *probe.Error) {
		return cfgCache, err
	}
}

// loadMcConfig - returns configuration, initialized later.
var loadMcConfig func() (*configV10, *probe.Error)

// saveMcConfig - saves configuration file and returns error if any.
func saveMcConfig(config *configV10) *probe.Error {
	if config == nil {
		return errInvalidArgument().Trace()
	}

	err := createMcConfigDir()
	if err != nil {
		return err.Trace(mustGetMcConfigDir())
	}

	// Save the config.
	if err := saveConfigV10(config); err != nil {
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

// cleanAlias removes any forbidden trailing slashes or backslashes
// before any validation to avoid annoying mc complaints.
func cleanAlias(s string) string {
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimSuffix(s, "\\")
	return s
}

// isValidAlias - Check if alias valid.
func isValidAlias(alias string) bool {
	return regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-_]+$").MatchString(alias)
}

// getAliasConfig retrieves host specific configuration such as access keys, signature type.
func getAliasConfig(alias string) (*aliasConfigV10, *probe.Error) {
	mcCfg, err := loadMcConfig()
	if err != nil {
		return nil, err.Trace(alias)
	}

	// if host is exact return quickly.
	if _, ok := mcCfg.Aliases[alias]; ok {
		hostCfg := mcCfg.Aliases[alias]
		return &hostCfg, nil
	}

	// return error if cannot be matched.
	return nil, errNoMatchingHost(alias).Trace(alias)
}

// mustGetHostConfig retrieves host specific configuration such as access keys, signature type.
func mustGetHostConfig(alias string) *aliasConfigV10 {
	aliasCfg, _ := getAliasConfig(alias)
	// If alias is not found,
	// look for it in the environment variable.
	if aliasCfg == nil {
		if envConfig, ok := os.LookupEnv(mcEnvHostPrefix + alias); ok {
			aliasCfg, _ = expandAliasFromEnv(envConfig)
		}
	}
	if aliasCfg == nil {
		if envConfig, ok := os.LookupEnv(mcEnvHostsDeprecatedPrefix + alias); ok {
			errorIf(errInvalidArgument().Trace(mcEnvHostsDeprecatedPrefix+alias), "`MC_HOSTS_<alias>` environment variable is deprecated. Please use `MC_HOST_<alias>` instead for the same functionality.")
			aliasCfg, _ = expandAliasFromEnv(envConfig)
		}
	}
	return aliasCfg
}

// Parse MC_HOST url obtained from the environment
// - It supports the following formats:
//      http(s)://access-key:secret-key@host:port/
//    AND
///     http(s)://access-key:secret-key:session-token@host:port/
// - Users need to manually escape any odds characters (except for @)
//   in order for this to work correctly.
func parseEnvURLStr(envURL string) (endpoint, accessKey, secretKey, sessionToken string, err *probe.Error) {
	u, e := url.Parse(envURL)
	if e != nil {
		return "", "", "", "", probe.NewError(e)
	}

	endpoint = u.Scheme + "://" + u.Host
	accessKey = u.User.Username()

	pwd, _ := u.User.Password()
	pwdFields := strings.Split(pwd, ":")
	secretKey = pwdFields[0]
	if len(pwdFields) > 1 {
		sessionToken = pwdFields[1]
	}

	// Look for if URL has invalid values and return error.
	if !((u.Scheme == "http" || u.Scheme == "https") &&
		(u.Path == "/" || u.Path == "") && u.Opaque == "" &&
		!u.ForceQuery && u.RawQuery == "" && u.Fragment == "") {
		return "", "", "", "", errInvalidArgument().Trace(u.String())
	}

	return
}

const (
	mcEnvHostPrefix            = "MC_HOST_"
	mcEnvHostsDeprecatedPrefix = "MC_HOSTS_"
)

func expandAliasFromEnv(envURL string) (*aliasConfigV10, *probe.Error) {
	endpoint, accessKey, secretKey, sessionToken, err := parseEnvURLStr(envURL)
	if err != nil {
		return nil, err.Trace(envURL)
	}

	return &aliasConfigV10{
		URL:          endpoint,
		API:          "S3v4",
		AccessKey:    accessKey,
		SecretKey:    secretKey,
		SessionToken: sessionToken,
	}, nil
}

// expandAlias expands aliased URL if any match is found, returns as is otherwise.
func expandAlias(aliasedURL string) (alias string, urlStr string, aliasCfg *aliasConfigV10, err *probe.Error) {
	// Extract alias from the URL.
	alias, path := url2Alias(aliasedURL)

	var envConfig string
	var ok bool

	if envConfig, ok = os.LookupEnv(mcEnvHostPrefix + alias); !ok {
		envConfig, ok = os.LookupEnv(mcEnvHostsDeprecatedPrefix + alias)
		if ok {
			errorIf(errInvalidArgument().Trace(mcEnvHostsDeprecatedPrefix+alias), "`MC_HOSTS_<alias>` environment variable is deprecated. Please use `MC_HOST_<alias>` instead for the same functionality.")
		}
	}

	if ok {
		aliasCfg, err = expandAliasFromEnv(envConfig)
		if err != nil {
			return "", "", nil, err.Trace(aliasedURL)
		}
		return alias, urlJoinPath(aliasCfg.URL, path), aliasCfg, nil
	}

	// Find the matching alias entry and expand the URL.
	if aliasCfg = mustGetHostConfig(alias); aliasCfg != nil {
		return alias, urlJoinPath(aliasCfg.URL, path), aliasCfg, nil
	}
	return "", aliasedURL, nil, nil // No matching entry found. Return original URL as is.
}

// mustExpandAlias expands aliased URL if any match is found, returns as is otherwise.
func mustExpandAlias(aliasedURL string) (alias string, urlStr string, aliasCfg *aliasConfigV10) {
	alias, urlStr, aliasCfg, _ = expandAlias(aliasedURL)
	return alias, urlStr, aliasCfg
}
