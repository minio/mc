/*
 * Mini Copy, (C) 2014,2015 Minio, Inc.
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
	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "mb", 1) // last argument is exit code
	}
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: Unable to read config")
	}

	targetURLConfigMap := make(map[string]*hostConfig)
	for _, arg := range ctx.Args() {
		targetURL, err := getURL(arg, config.Aliases)
		if err != nil {
			switch iodine.ToError(err).(type) {
			case errUnsupportedScheme:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("mc: reading URL [%s] failed, %s\n", arg, client.GuessPossibleURL(arg))
			default:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("mc: reading URL [%s] failed with following reason: [%s]\n", arg, iodine.ToError(err))
			}
		}
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: reading config URL [%s] failed with following reason: [%s]\n", targetURL, iodine.ToError(err))
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	errorMsg, err := doMakeBucketCmd(mcClientManager{}, targetURLConfigMap, globalDebugFlag)
	err = iodine.New(err, nil)
	if err != nil {
		if errorMsg == "" {
			errorMsg = "No error message present, please rerun with --debug and report a bug."
		}
		log.Debug.Println(err)
		console.Fatalf("%s", errorMsg)
	}
}

func doMakeBucketCmd(manager clientManager, targetURLConfigMap map[string]*hostConfig, debug bool) (string, error) {
	var err error
	var clnt client.Client

	for targetURL, targetConfig := range targetURLConfigMap {
		clnt, err = manager.getNewClient(targetURL, targetConfig, debug)
		if err != nil {
			err := iodine.New(err, nil)
			msg := fmt.Sprintf("mc: instantiating a new client for URL [%s] failed with following reason: [%s]\n",
				targetURL, iodine.ToError(err))
			return msg, err
		}
		err = clnt.PutBucket()
		if err != nil {
			console.Infof("Retrying ...")
		}
		for i := 0; i < globalMaxRetryFlag && err != nil; i++ {
			err = clnt.PutBucket()
			console.Errorf(" %d", i)
			// Progressively longer delays
			time.Sleep(time.Duration(i*i) * time.Second)
		}
		if err != nil {
			err := iodine.New(err, nil)
			msg := fmt.Sprintf("\nmc: Creating bucket failed for URL [%s] with following reason: [%s]\n", targetURL, iodine.ToError(err))
			return msg, err
		}
		console.Infoln()
	}
	return "", nil
}
