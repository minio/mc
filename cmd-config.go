/*
 * Mini Copy, (C) 2014, 2015 Minio, Inc.
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
	"net/url"
	"os"
	"path"
	"runtime"
	"strings"

	"os/user"
	"path/filepath"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/mc/pkg/qdb"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	mcConfigDir        = ".mc/"
	mcConfigWindowsDir = "mc/"
	mcConfigFile       = "config.json"
)

var (
	mcCurrentConfigVersion = qdb.Version{Major: 1, Minor: 0, Patch: 0}
)

const (
	// do not pass accesskeyid and secretaccesskey through cli
	// users should manually edit them, add a stub code
	accessKeyID     = "YOUR-ACCESS-KEY-ID-HERE"
	secretAccesskey = "YOUR-SECRET-ACCESS-KEY-HERE"
)

func getMcConfigDir() (string, error) {
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

func getMcConfigPath() (string, error) {
	dir, err := getMcConfigDir()
	if err != nil {
		return "", iodine.New(err, nil)
	}
	return path.Join(dir, mcConfigFile), nil
}

func mustGetMcConfigPath() string {
	p, _ := getMcConfigPath()
	return p
}

// getMcConfig returns the config
func getMcConfig() (config qdb.Store, err error) {
	if !isMcConfigExist() {
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	if configStore := qdb.NewStore(mcCurrentConfigVersion); configStore != nil {
		if err := configStore.Load(configFile); err != nil {
			return nil, iodine.New(err, nil)
		}
		return configStore, nil
	}
	return nil, iodine.New(errInvalidArgument{}, nil)
}

// isMcConfigExist returns true/false if config exists
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

// writeConfig
func writeConfig(config qdb.Store) error {
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

// saveConfig writes configuration data in json format to config file.
func saveConfig(ctx *cli.Context) error {
	switch ctx.Args().Get(0) {
	case "generate":
		if isMcConfigExist() {
			return iodine.New(errConfigExists{}, nil)
		}
		err := writeConfig(newConfig())
		if err != nil {
			return iodine.New(err, nil)
		}
		return nil
	case "check":
		// verify if the binary can load config file
		configStore, err := getMcConfig()
		if err != nil || configStore == nil {
			return iodine.New(err, nil)
		}
		return nil
	default:
		configStore, err := parseConfigInput(ctx)
		if err != nil {
			return iodine.New(err, nil)
		}
		return writeConfig(configStore)
	}
}

func newConfig() (config qdb.Store) {
	configStore := qdb.NewStore(mcCurrentConfigVersion)
	s3Auth := make(map[string]string)
	localAuth := make(map[string]string)

	hosts := make(map[string]map[string]string)
	s3Auth["Auth.AccessKeyID"] = accessKeyID
	s3Auth["Auth.SecretAccessKey"] = secretAccesskey
	hosts["http*://s3*.amazonaws.com"] = s3Auth

	// local minio server can have this empty until webcli is ready
	// which would make it easier to generate accesskeys and manage
	localAuth["Auth.AccessKeyID"] = ""
	localAuth["Auth.SecretAccessKey"] = ""
	hosts["http*://localhost:*"] = localAuth

	configStore.SetMapMapString("Hosts", hosts)

	aliases := make(map[string]string)
	aliases["s3"] = "https://s3.amazonaws.com"
	aliases["localhost"] = "http://localhost:9000"

	configStore.SetMapString("Aliases", aliases)
	return configStore
}

func parseConfigInput(ctx *cli.Context) (config qdb.Store, err error) {
	configStore := qdb.NewStore(mcCurrentConfigVersion)
	configStore.Load(mcConfigFile)

	alias := strings.Fields(ctx.String("alias"))
	switch true {
	case len(alias) == 2:
		aliasName := alias[0]
		url := strings.TrimSuffix(alias[1], "/")
		if strings.HasPrefix(aliasName, "http") {
			return nil, iodine.New(errInvalidAliasName{name: aliasName}, nil)
		}
		if !strings.HasPrefix(url, "http") {
			return nil, iodine.New(errInvalidURL{url: url}, nil)
		}
		if !isValidAliasName(aliasName) {
			return nil, iodine.New(errInvalidAliasName{name: aliasName}, nil)
		}
		aliases := configStore.GetMapString("Aliases")
		if _, ok := aliases[aliasName]; ok {
			return nil, iodine.New(errAliasExists{name: aliasName}, nil)
		}
		aliases[aliasName] = url
		configStore.SetMapString("Aliases", aliases)
		return configStore, nil
	default:
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
}

// getHostURL -
func getHostURL(u *url.URL) string {
	return u.Scheme + "://" + u.Host
}

// getHostConfig retrieves host specific configuration such as access keys, certs.
func getHostConfig(requestURL string) (map[string]string, error) {
	u, err := url.Parse(requestURL)
	if err != nil {
		return nil, iodine.New(errInvalidURL{url: requestURL}, nil)
	}
	config, err := getMcConfig()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	for globURL, hostConfig := range config.GetMapMapString("Hosts") {
		match, err := filepath.Match(globURL, getHostURL(u))
		if err != nil {
			return nil, iodine.New(errInvalidGlobURL{glob: globURL, request: requestURL}, nil)
		}
		if match {
			// verify Auth key validity for all hosts other than localhost
			if !strings.Contains(getHostURL(u), "localhost") {
				if !isValidAccessKey(hostConfig["Auth.AccessKeyID"]) {
					return nil, iodine.New(errInvalidAuthKeys{}, nil)
				}
				if !isValidSecretKey(hostConfig["Auth.SecretAccessKey"]) {
					return nil, iodine.New(errInvalidAuthKeys{}, nil)
				}
			}
			return hostConfig, nil
		}
	}
	return nil, iodine.New(errNoMatchingHost{}, nil)
}

// doConfigCmd is the handler for "mc config" sub-command.
func doConfigCmd(ctx *cli.Context) {
	// show help if nothing is set
	if len(ctx.Args()) < 1 && !ctx.IsSet("completion") && !ctx.IsSet("alias") {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	configPath, err := getMcConfigPath()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: Unable to identify config file path")
	}
	err = saveConfig(ctx)
	if err != nil {
		switch iodine.ToError(err).(type) {
		case errConfigExists:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln("mc: Configuration file " + configPath + " already exists")
		default:
			// unexpected error
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln("DEBUG mc: Unable to generate config file", configPath)
		}
	}
	console.Infoln("mc: Configuration written to " + configPath + ". Please update your access credentials.")
}
