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
	"fmt"
	"time"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

// doMakeBucketCmd creates a new bucket
func runMakeBucketCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "mb", 1) // last argument is exit code
	}
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read config file [%s].\n", mustGetMcConfigPath())
	}
	targetURLConfigMap := make(map[string]*hostConfig)
	targetURLs, err := getURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("Unknown URL type [%s] passed. Reason: [%s].\n", e.url, e)
		default:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("Error in parsing path or URL. Reason: [%s].\n", e)
		}
	}
	acl := bucketACL(ctx.Args().First())
	if !acl.isValidBucketACL() {
		log.Debug.Println(iodine.New(errInvalidACL{acl: acl.String()}, nil))
		console.Fatalf("Access type [%s] is not supported.  Valid types are [private, private, read-only].\n", acl)
	}
	targetURLs = targetURLs[1:] // 1 or more target URLs
	for _, targetURL := range targetURLs {
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("Unable to read configuration for host [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	for targetURL, targetConfig := range targetURLConfigMap {
		errorMsg, err := doMakeBucketCmd(mcClientManager{}, targetURL, acl.String(), targetConfig, globalDebugFlag)
		err = iodine.New(err, nil)
		if err != nil {
			if errorMsg == "" {
				errorMsg = "Empty error message.  Please rerun this command with --debug and file a bug report."
			}
			log.Debug.Println(err)
			console.Errorf("%s", errorMsg)
		}
	}
}

func doMakeBucketCmd(manager clientManager, targetURL, targetACL string, targetConfig *hostConfig, debug bool) (string, error) {
	var err error
	var clnt client.Client
	clnt, err = manager.getNewClient(targetURL, targetConfig, debug)
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("Unable to initialize client for [%s]. Reason: [%s].\n",
			targetURL, iodine.ToError(err))
		return msg, err
	}
	return doMakeBucket(clnt, targetURL, targetACL)
}

func doMakeBucket(clnt client.Client, targetURL, targetACL string) (string, error) {
	err := clnt.PutBucket(targetACL)
	if err != nil && isValidRetry(err) {
		console.Infof("Retrying ...")
	}
	for i := 0; i < globalMaxRetryFlag && err != nil && isValidRetry(err); i++ {
		err = clnt.PutBucket(targetACL)
		console.Errorf(" %d", i)
		// Progressively longer delays
		time.Sleep(time.Duration(i*i) * time.Second)
	}
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("Failed to create bucket for URL [%s]. Reason: [%s].\n", targetURL, iodine.ToError(err))
		return msg, err
	}
	return "", nil
}
