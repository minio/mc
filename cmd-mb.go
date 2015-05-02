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
	"time"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
)

// runMakeBucketCmd is the handler for mc mb command
func runMakeBucketCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "mb", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	config, err := getMcConfig()
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalf("Unable to read config file [%s].\n", mustGetMcConfigPath())
	}
	targetURLConfigMap := make(map[string]*hostConfig)
	targetURLs, err := getExpandedURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("Unknown URL type [%s] passed. Reason: [%s].\n", e.url, e)
		default:
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("Error in parsing path or URL. Reason: [%s].\n", e)
		}
	}
	for _, targetURL := range targetURLs {
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("Unable to read configuration for host [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	for targetURL, targetConfig := range targetURLConfigMap {
		errorMsg, err := doMakeBucketCmd(targetURL, targetConfig, globalDebugFlag)
		err = iodine.New(err, nil)
		if err != nil {
			if errorMsg == "" {
				errorMsg = "Empty error message.  Please rerun this command with --debug and file a bug report."
			}
			console.Debugln(err)
			console.Errorf("%s", errorMsg)
		}
	}
}

// doMakeBucketCmd -
func doMakeBucketCmd(targetURL string, targetConfig *hostConfig, debug bool) (string, error) {
	var err error
	var clnt client.Client
	clnt, err = getNewClient(targetURL, targetConfig, debug)
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("Unable to initialize client for [%s]. Reason: [%s].\n",
			targetURL, iodine.ToError(err))
		return msg, err
	}
	return doMakeBucket(clnt, targetURL)
}

// doMakeBucket - wrapper around PutBucket() API
func doMakeBucket(clnt client.Client, targetURL string) (string, error) {
	err := clnt.PutBucket()
	for i := 0; i < globalMaxRetryFlag && err != nil && isValidRetry(err); i++ {
		fmt.Println(console.Retry("Retrying ... %d", i))
		// Progressively longer delays
		time.Sleep(time.Duration(i*i) * time.Second)
		err = clnt.PutBucket()
	}
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("Failed to create bucket for URL [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
		return msg, err
	}
	return "", nil
}
