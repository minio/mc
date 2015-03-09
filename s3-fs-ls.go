/*
 * Mini Object Storage, (C) 2014,2015 Minio, Inc.
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
	"sort"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func printBuckets(v []*s3.Bucket) {
	for _, b := range v {
		msg := fmt.Sprintf("%v %13s %s", b.CreationDate.Local(), "", b.Name)
		info(msg)
	}
}

func printObjects(v []*s3.Item) {
	if len(v) > 0 {
		sort.Sort(s3.BySize(v))
		for _, b := range v {
			printObject(b.LastModified.Time, b.Size, b.Key)
		}
	}
}

func printObject(date time.Time, v int64, key string) {
	msg := fmt.Sprintf("%v  %13s %s", date.Local(), pb.FormatBytes(v), key)
	info(msg)
}

func doFsList(c *cli.Context) {
	var items []*s3.Item

	config, err := getMcConfig()
	if err != nil {
		fatal(err.Error())
	}

	s3c, err := getNewClient(config)
	if err != nil {
		fatal(err.Error())
	}
	fsoptions, err := parseOptions(c)
	if err != nil {
		fatal(err.Error())
	}
	switch true {
	case fsoptions.bucket == "": // List all buckets
		buckets, err := s3c.Buckets()
		if err != nil {
			fatal(err.Error())
		}
		printBuckets(buckets)
	case fsoptions.key == "": // List the objects in a bucket
		items, _, err = s3c.GetBucket(fsoptions.bucket, "", "", "", s3.MaxKeys)
		if err != nil {
			fatal(err.Error())
		}
		printObjects(items)
	case fsoptions.key != "": // List objects matching the key prefix
		var date time.Time
		var size int64

		size, date, err = s3c.Stat(fsoptions.key, fsoptions.bucket)
		switch err {
		case nil: // List a single object. Exact key prefix match
			printObject(date, size, fsoptions.key)
		case os.ErrNotExist:
			// List all objects matching the key prefix
			items, _, err = s3c.GetBucket(fsoptions.bucket, "", fsoptions.key, "", s3.MaxKeys)
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
	default:
		fatal(err.Error())
	}
}
