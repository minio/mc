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
	"errors"
	"fmt"

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
		console.Fatalln(console.ErrorMessage{
			Message: "Please run \"mc config generate\"",
			Error:   iodine.New(errors.New("\"mc\" is not configured"), nil),
		})
	}

	config, err := getMcConfig()
	if err != nil {
		console.Fatalln(console.ErrorMessage{
			Message: fmt.Sprintf("Unable to read config file ‘%s’", mustGetMcConfigPath()),
			Error:   iodine.New(err, nil),
		})
	}
	targetURLConfigMap := make(map[string]*hostConfig)
	for _, arg := range ctx.Args() {
		targetURL, err := getExpandedURL(arg, config.Aliases)
		if err != nil {
			switch e := iodine.ToError(err).(type) {
			case errUnsupportedScheme:
				console.Fatalln(console.ErrorMessage{
					Message: fmt.Sprintf("Unknown type of URL ‘%s’", e.url),
					Error:   iodine.New(e, nil),
				})
			default:
				console.Fatalln(console.ErrorMessage{
					Message: fmt.Sprintf("Unable to parse argument ‘%s’", arg),
					Error:   iodine.New(err, nil),
				})
			}
		}
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			console.Fatalln(console.ErrorMessage{
				Message: fmt.Sprintf("Unable to read host configuration for ‘%s’ from config file ‘%s’", targetURL, mustGetMcConfigPath()),
				Error:   iodine.New(err, nil),
			})
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	for targetURL, targetConfig := range targetURLConfigMap {
		// if recursive strip off the "..."
		newTargetURL := stripRecursiveURL(targetURL)
		err = doListCmd(newTargetURL, targetConfig, isURLRecursive(targetURL))
		if err != nil {
			console.Fatalln(console.ErrorMessage{
				Message: fmt.Sprintf("Failed to list ‘%s’", targetURL),
				Error:   iodine.New(err, nil),
			})
		}
	}
}

// doListCmd -
func doListCmd(targetURL string, targetConfig *hostConfig, recursive bool) error {
	clnt, err := getNewClient(targetURL, targetConfig)
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	err = doList(clnt, targetURL, recursive)
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}
