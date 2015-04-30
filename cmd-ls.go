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
	"fmt"
	"strings"
	"time"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
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
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read config file [%s]. Reason: [%s].\n", mustGetMcConfigPath(), iodine.ToError(err))
	}
	targetURLConfigMap := make(map[string]*hostConfig)
	for _, arg := range ctx.Args() {
		targetURL, err := getExpandedURL(arg, config.Aliases)
		if err != nil {
			switch iodine.ToError(err).(type) {
			case errUnsupportedScheme:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("Unknown type of URL [%s].\n", arg)
			default:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("Unable to parse argument [%s]. Reason: [%s].\n", arg, iodine.ToError(err))
			}
		}
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("Unable to read host configuration for [%s] from config file [%s]. Reason: [%s].\n",
				targetURL, mustGetMcConfigPath(), iodine.ToError(err))
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	for targetURL, targetConfig := range targetURLConfigMap {
		if isURLRecursive(targetURL) {
			// if recursive strip off the "..."
			targetURL = strings.TrimSuffix(targetURL, recursiveSeparator)
			err = doListRecursiveCmd(mcClientMethods{}, targetURL, targetConfig, globalDebugFlag)
			err = iodine.New(err, nil)
			if err != nil {
				log.Debug.Println(err)
				console.Fatalf("Failed to list [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
			}
		} else {
			err = doListCmd(mcClientMethods{}, targetURL, targetConfig, globalDebugFlag)
			if err != nil {
				if err != nil {
					log.Debug.Println(err)
					console.Fatalf("Failed to list [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
				}
			}
		}
	}
}

// doListCmd -
func doListCmd(methods clientMethods, targetURL string, targetConfig *hostConfig, debug bool) error {
	clnt, err := methods.getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	err = doList(clnt, targetURL)
	for i := 0; i < globalMaxRetryFlag && err != nil && isValidRetry(err); i++ {
		fmt.Println(console.Retry("Retrying ... %d", i))
		// Progressively longer delays
		time.Sleep(time.Duration(i*i) * time.Second)
		err = doList(clnt, targetURL)
	}
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}

// doListRecursiveCmd -
func doListRecursiveCmd(methods clientMethods, targetURL string, targetConfig *hostConfig, debug bool) error {
	clnt, err := methods.getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		return iodine.New(err, map[string]string{"Target": targetURL})
	}
	err = doListRecursive(clnt, targetURL)
	for i := 0; i < globalMaxRetryFlag && err != nil && isValidRetry(err); i++ {
		fmt.Println(console.Retry("Retrying ... %d", i))
		// Progressively longer delays
		time.Sleep(time.Duration(i*i) * time.Second)
		err = doListRecursive(clnt, targetURL)
	}
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}
