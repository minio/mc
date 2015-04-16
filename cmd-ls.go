/*
 * Modern Copy, (C) 2014,2015 Minio, Inc.
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

// printBuckets lists buckets and its metadata
func printBuckets(v []*client.Bucket) {
	for _, b := range v {
		console.Infof("%23s %13s %s\n", b.CreationDate.Local().Format(printDate), "", b.Name)
	}
}

// printObjects prints a metadata of a list of objects
func printObjects(v []*client.Item) {
	if len(v) > 0 {
		// Items are already sorted
		for _, b := range v {
			printObject(b.LastModified, b.Size, b.Key)
		}
	}
}

// printObject prints object meta-data
func printObject(date time.Time, v int64, key string) {
	console.Infof("%23s %13s %s\n", date.Local().Format(printDate), pb.FormatBytes(v), key)
}

// doListCmd lists objects inside a bucket
func doListCmd(ctx *cli.Context) {
	var items []*client.Item

	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: Unable to get config")
	}
	for _, arg := range ctx.Args() {
		targetURLParser, err := parseURL(arg, config.Aliases)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln("mc: Unable to parse URL")
		}
		client, err := getNewClient(targetURLParser, globalDebugFlag)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln("mc: Unable to initiate new client")
		}

		// ListBuckets() will not be called for fsClient() as its not needed.
		if targetURLParser.bucketName == "" && targetURLParser.urlType != urlFile {
			buckets, err := client.ListBuckets()
			if err != nil {
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalln("mc: Unable to list buckets for ", targetURLParser.String())
			}
			printBuckets(buckets)
		} else {
			items, err = client.ListObjects(targetURLParser.bucketName, targetURLParser.objectName)
			if err != nil {
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalln("mc: Unable to list objects for ", targetURLParser.String())
			}
			printObjects(items)
		}
	}
}
