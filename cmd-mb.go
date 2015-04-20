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
	for _, arg := range ctx.Args() {
		u, err := getURL(arg, config.GetMapString("Aliases"))
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
		errorMsg, err := doMakeBucketCmd(mcClientManager{}, u, globalDebugFlag)
		err = iodine.New(err, nil)
		if err != nil {
			if errorMsg == "" {
				errorMsg = "No error message present, please rerun with --debug and report a bug."
			}
			log.Debug.Println(err)
			console.Fatalf("%s", errorMsg)
		}
	}
}

func doMakeBucketCmd(manager clientManager, u string, debug bool) (string, error) {
	var err error
	var clnt client.Client

	clnt, err = manager.getNewClient(u, debug)
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("mc: instantiating a new client for URL [%s] failed with following reason: [%s]\n", u, iodine.ToError(err))
		return msg, err
	}
	err = clnt.PutBucket()
	if err != nil {
		console.Infof("Retrying ...")
	}
	for i := 1; i <= globalMaxRetryFlag && err != nil; i++ {
		err = clnt.PutBucket()
		console.Errorf(" %d", i)
		// Progressively longer delays
		time.Sleep(time.Duration(i*i) * time.Second)
	}
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("\nmc: Creating bucket failed for URL [%s] with following reason: [%s]\n", u, iodine.ToError(err))
		return msg, err
	}
	return "", nil
}
