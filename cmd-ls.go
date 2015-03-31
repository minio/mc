/*
 * Minimalist Object Storage, (C) 2014,2015 Minio, Inc.
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
	"os"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
)

const (
	printDate = "2006-01-02 15:04:05 MST"
)

// printBuckets lists buckets and its meta-dat
func printBuckets(v []*client.Bucket) {
	for _, b := range v {
		msg := fmt.Sprintf("%23s %13s %s", b.CreationDate.Time.Local().Format(printDate), "", b.Name)
		info(msg)
	}
}

// printObjects prints a meta-data of a list of objects
func printObjects(v []*client.Item) {
	if len(v) > 0 {
		// Items are already sorted
		for _, b := range v {
			printObject(b.LastModified.Time, b.Size, b.Key)
		}
	}
}

// printObject prints object meta-data
func printObject(date time.Time, v int64, key string) {
	msg := fmt.Sprintf("%23s %13s %s", date.Local().Format(printDate), pb.FormatBytes(v), key)
	info(msg)
}

// listObjectPrefix prints matching key prefix
func listObjectPrefix(s3c client.Client, bucketName, objectName string, maxkeys int) {
	var date time.Time
	var size int64
	var err error

	size, date, err = s3c.Stat(bucketName, objectName)
	var items []*client.Item
	switch err {
	case nil: // List a single object. Exact key
		printObject(date, size, objectName)
	case os.ErrNotExist:
		// List all objects matching the key prefix
		items, _, err = s3c.ListObjects(bucketName, "", objectName, "", maxkeys)
		if err != nil {
			fatal(err.Error())
		}
		if len(items) > 0 {
			printObjects(items)
		} else {
			fatal(os.ErrNotExist.Error())
		}
	default: // Error
		fatal(err.Error())
	}
}

// doListCmd lists objects inside a bucket
func doListCmd(c *cli.Context) {
	var items []*client.Item
	// quiet := c.GlobalBool("quiet")

	urlStr, err := parseURL(c.Args().First())
	if err != nil {
		fatal(err.Error())
	}

	bucketName, objectName, err := url2Object(urlStr)
	if err != nil {
		fatal(err.Error())
	}

	s3c, err := getNewClient(c.GlobalBool("debug"), urlStr)
	if err != nil {
		fatal(err.Error())
	}

	switch true {
	case bucketName == "": // List all buckets
		buckets, err := s3c.ListBuckets()
		if err != nil {
			fatal(err.Error())
		}
		printBuckets(buckets)
	case objectName == "": // List objects in a bucket
		items, _, err = s3c.ListObjects(bucketName, "", "", "", client.Maxkeys)
		if err != nil {
			fatal(err.Error())
		}
		printObjects(items)
	case objectName != "": // List objects matching the key prefix
		listObjectPrefix(s3c, bucketName, objectName, client.Maxkeys)
	default:
		fatal(err.Error())
	}
}
