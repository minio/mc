/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/quick"
	"github.com/minio/minio/pkg/iodine"
)

// runConfigCmd is the handle for "mc config" sub-command
func runConfigCmd(ctx *cli.Context) {
	// show help if nothing is set
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "config", 1) // last argument is exit code
	}
	arg := ctx.Args().First()
	tailArgs := ctx.Args().Tail()
	if len(tailArgs) > 2 {
		console.Fatalln("Incorrect number of arguments, please use \"mc config help\"")
	}
	msg, err := doConfig(arg, tailArgs)
	if err != nil {
		console.Fatalln(msg)
	}
	console.Infoln(msg)
}

// saveConfig writes configuration data in json format to config file.
func saveConfig(arg string, aliases []string) error {
	switch arg {
	case "generate":
		if isMcConfigExist() {
			return iodine.New(errConfigExists{}, nil)
		}

		config, err := newConfig()
		if err != nil {
			return iodine.New(err, nil)
		}

		err = writeConfig(config)
		if err != nil {
			return iodine.New(err, nil)
		}
		return nil
	case "alias":
		config, err := addAlias(aliases)
		if err != nil {
			return iodine.New(err, nil)
		}
		return writeConfig(config)
	default:
		return iodine.New(errInvalidArgument{}, nil)
	}
}

// doConfig is the handler for "mc config" sub-command.
func doConfig(arg string, aliases []string) (string, error) {
	configPath, err := getMcConfigPath()
	if err != nil {
		return "Unable to determine config file path.", iodine.New(err, nil)
	}
	err = saveConfig(arg, aliases)
	if err != nil {
		switch iodine.ToError(err).(type) {
		case errConfigExists:
			return "Configuration file [" + configPath + "] already exists.", iodine.New(err, nil)
		case errInvalidArgument:
			return "Incorrect usage, please use \"mc config help\" ", iodine.New(err, nil)
		case errAliasExists:
			return "Alias [" + aliases[0] + "] already exists", iodine.New(err, nil)
		case errInvalidAliasName:
			return "Alias [" + aliases[0] + "] is reserved word or invalid", iodine.New(err, nil)
		case errInvalidURL:
			return "Alias [" + aliases[1] + "] is invalid URL", iodine.New(err, nil)
		default:
			// unexpected error
			return "Unable to generate config file [" + configPath + "].", iodine.New(err, nil)
		}
	}
	return "Configuration written to [" + configPath + "]. Please update your access credentials.", nil
}

// addAlias - add new aliases
func addAlias(aliases []string) (quick.Config, error) {
	if len(aliases) < 2 {
		return nil, iodine.New(errInvalidArgument{}, nil)
	}
	conf := newConfigV1()
	config, err := quick.New(conf)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	config.Load(mustGetMcConfigPath())

	aliasName := aliases[0]
	url := strings.TrimSuffix(aliases[1], "/")
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
	newConfig, err := quick.New(newConf)
	if err != nil {
		return nil, iodine.New(err, nil)
	}
	return newConfig, nil
}
