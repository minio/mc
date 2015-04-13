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
	"fmt"
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
		// TODO: escape sequences internally in color package clobber the output, format this prior
		// find a way out
		msg := fmt.Sprintf("%23s %13s %s", b.CreationDate.Local().Format(printDate), "", b.Name)
		console.Infoln(msg)
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
	// TODO: escape sequences internally in color package clobber the output, format this prior
	// find a way out
	msg := fmt.Sprintf("%23s %13s %s", date.Local().Format(printDate), pb.FormatBytes(v), key)
	console.Infoln(msg)
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
		console.Fatalln("Unable to get config")
	}
	for _, arg := range ctx.Args() {
		urlp, err := parseURL(arg, config.Aliases)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln(err)
		}
		client, err := getNewClient(urlp, globalDebugFlag)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalln(err)
		}
		if urlp.bucketName == "" { // List all buckets
			buckets, err := client.ListBuckets()
			if err != nil {
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalln(err)
			}
			printBuckets(buckets)
		} else {
			items, err = client.ListObjects(urlp.bucketName, urlp.objectName)
			if err != nil {
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalln(err)
			}
			printObjects(items)
		}
	}
}
