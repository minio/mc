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
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// runListCmd - is a handler for mc ls command
func runListCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	config, err := getMcConfig()
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalf("Unable to read config file [%s]. Reason: [%s].\n", mustGetMcConfigPath(), iodine.ToError(err))
	}
	targetURLConfigMap := make(map[string]*hostConfig)
	for _, arg := range ctx.Args() {
		targetURL, err := getExpandedURL(arg, config.Aliases)
		if err != nil {
			switch iodine.ToError(err).(type) {
			case errUnsupportedScheme:
				console.Debugln(iodine.New(err, nil))
				console.Fatalf("Unknown type of URL [%s].\n", arg)
			default:
				console.Debugln(iodine.New(err, nil))
				console.Fatalf("Unable to parse argument [%s]. Reason: [%s].\n", arg, iodine.ToError(err))
			}
		}
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("Unable to read host configuration for [%s] from config file [%s]. Reason: [%s].\n",
				targetURL, mustGetMcConfigPath(), iodine.ToError(err))
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	for targetURL, targetConfig := range targetURLConfigMap {
		if isURLRecursive(targetURL) {
			// if recursive strip off the "..."
			targetURL = stripRecursiveURL(targetURL)
			err = doListRecursiveCmd(targetURL, targetConfig, globalDebugFlag)
			err = iodine.New(err, nil)
			if err != nil {
				console.Debugln(err)
				console.Fatalf("Failed to list [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
			}
		} else {
			err = doListCmd(targetURL, targetConfig, globalDebugFlag)
			if err != nil {
				if err != nil {
					console.Debugln(err)
					console.Fatalf("Failed to list [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
				}
			}
		}
	}
}

// doListCmd -
func doListCmd(targetURL string, targetConfig *hostConfig, debug bool) error {
	clnt, err := getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	err = doList(clnt, targetURL)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// doListRecursiveCmd -
func doListRecursiveCmd(targetURL string, targetConfig *hostConfig, debug bool) error {
	clnt, err := getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	err = doListRecursive(clnt, targetURL)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}
