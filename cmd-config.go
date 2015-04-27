/*
 * Mini Copy (C) 2014, 2015 Minio, Inc.
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
	"path"
	"runtime"
	"strings"

	"os/user"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/mc/pkg/quick"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
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

func addAlias(alias []string) (quick.Config, error) {
	if len(alias) < 2 {
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
	conf := newConfigV1()
	config := quick.New(conf)
	config.Load(mustGetMcConfigPath())

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
	newConf := config.Data().(*configV1)
	if _, ok := newConf.Aliases[aliasName]; ok {
		return nil, iodine.New(errAliasExists{name: aliasName}, nil)
	}
	newConf.Aliases[aliasName] = url
	newConfig := quick.New(newConf)
	return newConfig, nil
}

// runConfigCmd is the handle for "mc config" sub-command
func runConfigCmd(ctx *cli.Context) {
	// show help if nothing is set
	if !ctx.Args().Present() && !ctx.IsSet("alias") || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	arg := ctx.Args().First()
	alias := strings.Fields(ctx.String("alias"))
	msg, err := doConfig(arg, alias)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln(msg)
	}
}

// saveConfig writes configuration data in json format to config file.
func saveConfig(arg string, alias []string) error {
	switch arg {
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
		config, err := addAlias(alias)
		if err != nil {
			return iodine.New(err, nil)
		}
		return writeConfig(config)
	}
}

// doConfig is the handler for "mc config" sub-command.
func doConfig(arg string, alias []string) (string, error) {
	configPath, err := getMcConfigPath()
	if err != nil {
		return "Unable to determine config file path.", iodine.New(err, nil)
	}
	err = saveConfig(arg, alias)
	if err != nil {
		switch iodine.ToError(err).(type) {
		case errConfigExists:
			return "Configuration file [" + configPath + "] already exists.", iodine.New(err, nil)
		case errInvalidArgument:
			return "Incorrect usage, please use \"help\" ", iodine.New(err, nil)
		case errAliasExists:
			return "Alias [" + alias[0] + "] already exists", iodine.New(err, nil)
		case errInvalidAliasName:
			return "Alias [" + alias[0] + "] is reserved word or invalid", iodine.New(err, nil)
		case errInvalidURL:
			return "Alias [" + alias[1] + "] is invalid URL", iodine.New(err, nil)
		default:
			// unexpected error
			return "Unable to generate config file [" + configPath + "].", iodine.New(err, nil)
		}
	}
	return "Configuration written to [" + configPath + "]. Please update your access credentials.", nil
}
