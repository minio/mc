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
	"time"

	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	printDate = "2006-01-02 15:04:05 MST"
)

// printItem prints item meta-data
func printItem(date time.Time, v int64, name string) {
	fmt.Printf(console.Time("[%s] ", date.Local().Format(printDate)))
	fmt.Printf(console.Size("%6s ", humanize.IBytes(uint64(v))))
	fmt.Println(console.File("%s", name))
}

func doList(clnt client.Client, targetURL string) (string, error) {
	var err error
	for itemCh := range clnt.List() {
		if itemCh.Err != nil {
			err = itemCh.Err
			break
		}
		printItem(itemCh.Item.Time, itemCh.Item.Size, itemCh.Item.Name)
	}
	if err != nil {
		err = iodine.New(err, nil)
		msg := fmt.Sprintf("mc: listing objects for URL [%s] failed with following reason: [%s]\n", targetURL, iodine.ToError(err))
		return msg, err
	}
	return "", nil
}

// runListCmd lists objects inside a bucket
func runListCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 1 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: Error reading config file. Reason: %s\n", iodine.ToError(err))
	}
	targetURLConfigMap := make(map[string]*hostConfig)
	for _, arg := range ctx.Args() {
		targetURL, err := getURL(arg, config.Aliases)
		if err != nil {
			switch iodine.ToError(err).(type) {
			case errUnsupportedScheme:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("mc: Unknown type of URL [%s].\n", arg)
			default:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("mc: Unknown type of URL [%s]. Reason: %s\n", arg, iodine.ToError(err))
			}
		}
		targetConfig, err := getHostConfig(targetURL)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: Error reading config for URL [%s]. Reason: %s\n", targetURL, iodine.ToError(err))
		}
		targetURLConfigMap[targetURL] = targetConfig
	}
	for targetURL, targetConfig := range targetURLConfigMap {
		errorMsg, err := doListCmd(mcClientManager{}, targetURL, targetConfig, globalDebugFlag)
		err = iodine.New(err, nil)
		if err != nil {
			if errorMsg == "" {
				errorMsg = "mc: List command failed. Please re-run with --debug and report this bug."
			}
			log.Debug.Println(err)
			console.Errorf("%s", errorMsg)
		}
	}
}

func doListCmd(manager clientManager, targetURL string, targetConfig *hostConfig, debug bool) (string, error) {
	clnt, err := manager.getNewClient(targetURL, targetConfig, globalDebugFlag)
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("mc: instantiating a new client for URL [%s] failed with following reason: [%s]\n",
			targetURL, iodine.ToError(err))
		return msg, err
	}
	return doList(clnt, targetURL)
}
