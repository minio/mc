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

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	printDate = "2006-01-02 15:04:05 MST"
)

// printItems prints a metadata of a list of items
func printItems(v []*client.Item) {
	if len(v) > 0 {
		// Items are already sorted
		for _, b := range v {
			printItem(b.Time, b.Size, b.Name)
		}
	}
}

// printItem prints item meta-data
func printItem(date time.Time, v int64, name string) {
	console.Infof("%23s %13s %s\n", date.Local().Format(printDate), pb.FormatBytes(v), name)
}

func doList(clnt client.Client, urlStr string) (string, error) {
	var err error
	var items []*client.Item

	items, err = clnt.List()
	for i := 0; i < globalMaxRetryFlag && err != nil; i++ {
		items, err = clnt.List()
		time.Sleep(time.Duration(i*i) * time.Second)
	}
	if err != nil {
		err = iodine.New(err, nil)
		msg := fmt.Sprintf("mc: listing objects for URL [%s] failed with following reason: [%s]\n", urlStr, iodine.ToError(err))
		return msg, err
	}
	printItems(items)
	return "", nil
}

// runListCmd lists objects inside a bucket
func runListCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: reading config file failed with following reason: [%s]\n", iodine.ToError(err))
	}
	for _, arg := range ctx.Args() {
		u, err := parseURL(arg, config.GetMapString("Aliases"))
		if err != nil {
			switch iodine.ToError(err).(type) {
			case errUnsupportedScheme:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("mc: reading URL [%s] failed with invalid scheme, %s\n", arg, client.GuessPossibleURL(arg))
			default:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("mc: reading URL [%s] failed with following reason: [%s]\n", arg, iodine.ToError(err))
			}
		}
		doListCmd(mcClientManager{}, u, globalDebugFlag)
	}
}

func doListCmd(manager clientManager, u string, debug bool) (string, error) {
	clnt, err := manager.getNewClient(u, globalDebugFlag)
	if err != nil {
		err := iodine.New(err, nil)
		msg := fmt.Sprintf("mc: instantiating a new client for URL [%s] failed with following reason: [%s]\n",
			u, iodine.ToError(err))
		return msg, err
	}

	return doList(clnt, u)

}
