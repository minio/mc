/*
 * Minio Client, (C) 2015 Minio, Inc.
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
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

func runAccessCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "access", 1) // last argument is exit code
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
			Message: "loading config file failed",
			Error:   iodine.New(err, nil),
		})
	}
	targetURLConfigMap := make(map[string]*hostConfig)
	targetURLs, err := getExpandedURLs(ctx.Args(), config.Aliases)
	if err != nil {
		console.Fatalln(console.ErrorMessage{
			Message: "Unknown type of URL ",
			Error:   iodine.New(err, nil),
		})
	}
	acl := bucketACL(ctx.Args().First())
	if !acl.isValidBucketACL() {
		console.Fatalln(console.ErrorMessage{
			Message: "Valid types are [private, public, readonly].",
			Error:   iodine.New(errors.New("Invalid ACL Type ‘"+acl.String()+"’"), nil),
		})
	}
	targetURLs = targetURLs[1:] // 1 or more target URLs
	for _, targetURL := range targetURLs {
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			console.Fatalln(console.ErrorMessage{
				Message: "Unable to read configuration for host " + "‘" + targetURL + "’",
				Error:   iodine.New(err, nil),
			})
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	for targetURL, targetConfig := range targetURLConfigMap {
		errorMsg, err := doUpdateAccessCmd(targetURL, acl.String(), targetConfig)
		if err != nil {
			console.Errorln(console.ErrorMessage{
				Message: errorMsg,
				Error:   iodine.New(err, nil),
			})
		}
	}
}

func doUpdateAccessCmd(targetURL, targetACL string, targetConfig *hostConfig) (string, error) {
	var err error
	var clnt client.Client
	clnt, err = getNewClient(targetURL, targetConfig)
	if err != nil {
		msg := fmt.Sprintf("Unable to initialize client for ‘%s’", targetURL)
		return msg, iodine.New(err, nil)
	}
	return doUpdateAccess(clnt, targetURL, targetACL)
}

func doUpdateAccess(clnt client.Client, targetURL, targetACL string) (string, error) {
	err := clnt.SetBucketACL(targetACL)
	for i := 0; i < globalMaxRetryFlag && err != nil && isValidRetry(err); i++ {
		console.Retry("Retrying ...", i)
		// Progressively longer delays
		time.Sleep(time.Duration(i*i) * time.Second)
		err = clnt.SetBucketACL(targetACL)
	}
	if err != nil {
		msg := fmt.Sprintf("Failed to add bucket access policy for URL ‘%s’", targetURL)
		return msg, iodine.New(err, nil)
	}
	return "", nil
}
