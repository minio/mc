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
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/mc/pkg/quick"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	mcConfigDir        = ".mc/"
	mcConfigWindowsDir = "mc/"
	mcConfigFile       = "config.json"
)

type hostConfig struct {
	AccessKeyID     string
	SecretAccessKey string
}

type configV1 struct {
	Version string
	Aliases map[string]string
	Hosts   map[string]*hostConfig
}

var (
	mcCurrentConfigVersion = "1.0.0"
)

const (
	// do not pass accesskeyid and secretaccesskey through cli
	// users should manually edit them, add a stub entry
	globalAccessKeyID     = "YOUR-ACCESS-KEY-ID-HERE"
	globalSecretAccessKey = "YOUR-SECRET-ACCESS-KEY-HERE"
)

const (
	exampleHostURL = "YOUR-EXAMPLE.COM"
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
func getMcConfig() (config *configV1, err error) {
	if !isMcConfigExist() {
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
	configFile, err := getMcConfigPath()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	conf := newConfigV1()
	if config := quick.New(conf); config != nil {
		if err := config.Load(configFile); err != nil {
			return nil, iodine.New(err, nil)
		}
		return config.Data().(*configV1), nil
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
func writeConfig(config quick.Config) error {
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
	default:
		config, err := parseConfigInput(ctx)
		if err != nil {
			return iodine.New(err, nil)
		}
		return writeConfig(config)
	}
}

func newConfigV1() *configV1 {
	conf := new(configV1)
	conf.Version = mcCurrentConfigVersion
	// make sure to allocate map's otherwise Golang
	// exists silently without providing any errors
	conf.Hosts = make(map[string]*hostConfig)
	conf.Aliases = make(map[string]string)
	return conf
}

func newConfig() (config quick.Config) {
	conf := newConfigV1()
	s3HostConf := new(hostConfig)
	s3HostConf.AccessKeyID = globalAccessKeyID
	s3HostConf.SecretAccessKey = globalSecretAccessKey

	// Your example host config
	exampleHostConf := new(hostConfig)
	exampleHostConf.AccessKeyID = globalAccessKeyID
	exampleHostConf.SecretAccessKey = globalSecretAccessKey

	conf.Hosts[exampleHostURL] = exampleHostConf
	conf.Hosts["http*://s3*.amazonaws.com"] = s3HostConf

	aliases := make(map[string]string)
	aliases["s3"] = "https://s3.amazonaws.com"
	aliases["localhost"] = "http://localhost:9000"
	conf.Aliases = aliases
	config = quick.New(conf)

	return config
}

func parseConfigInput(ctx *cli.Context) (config quick.Config, err error) {
	conf := newConfigV1()
	config = quick.New(&conf)
	config.Load(mcConfigFile)

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
		// convert interface{} back to its original struct
		newConf := config.Data().(configV1)
		if _, ok := newConf.Aliases[aliasName]; ok {
			return nil, iodine.New(errAliasExists{name: aliasName}, nil)
		}
		newConf.Aliases[aliasName] = url
		newConfig := quick.New(newConf)
		return newConfig, nil
	default:
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
}

// getHostURL -
func getHostURL(u *url.URL) string {
	return u.Scheme + "://" + u.Host
}

func getHostConfigs(requestURLs []string) (hostConfigs map[string]*hostConfig, err error) {
	hostConfigs = make(map[string]*hostConfig)
	for _, requestURL := range requestURLs {
		hostConfigs[requestURL], err = getHostConfig(requestURL)
		if err != nil {
			return nil, iodine.New(err, nil)
		}
	}
	return hostConfigs, nil
}

// getHostConfig retrieves host specific configuration such as access keys, certs.
func getHostConfig(requestURL string) (*hostConfig, error) {
	config, err := getMcConfig()
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	u, err := url.Parse(requestURL)
	if err != nil {
		return nil, iodine.New(errInvalidURL{url: requestURL}, nil)
	}
	// No host matching or keys needed for filesystem requests
	if client.GetType(requestURL) == client.Filesystem {
		hostCfg := &hostConfig{
			AccessKeyID:     "",
			SecretAccessKey: "",
		}
		return hostCfg, nil
	}

	// No host matching or keys needed for localhost and 127.0.0.1 URL's skip them
	if strings.Contains(getHostURL(u), "localhost") || strings.Contains(getHostURL(u), "127.0.0.1") {
		hostCfg := &hostConfig{
			AccessKeyID:     "",
			SecretAccessKey: "",
		}
		return hostCfg, nil
	}
	for globURL, hostCfg := range config.Hosts {
		match, err := filepath.Match(globURL, getHostURL(u))
		if err != nil {
			return nil, iodine.New(errInvalidGlobURL{glob: globURL, request: requestURL}, nil)
		}
		if match {
			if hostCfg == nil {
				return nil, iodine.New(errInvalidAuth{}, nil)
			}
			// verify Auth key validity for all hosts
			if !client.IsValidAccessKey(hostCfg.AccessKeyID) || !client.IsValidSecretKey(hostCfg.SecretAccessKey) {
				return nil, iodine.New(errInvalidAuthKeys{}, nil)
			}
			return hostCfg, nil
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
			console.Fatalln("mc: Unable to generate config file", configPath)
		}
	}
	console.Infoln("mc: Configuration written to " + configPath + ". Please update your access credentials.")
}
