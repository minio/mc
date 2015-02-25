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
	"log"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

const (
	recvFormat  = "2006-01-02T15:04:05.000Z"
	printFormat = "2006-01-02 15:04:05"
)

func parseTime(t string) string {
	ti, _ := time.Parse(recvFormat, t)
	return ti.Format(printFormat)
}

func parseLastModified(t string) string {
	ti, _ := time.Parse(time.RFC1123, t)
	return ti.Format(printFormat)
}

func printBuckets(v []*s3.Bucket) {
	for _, b := range v {
		fmt.Printf("%s %s\n", parseTime(b.CreationDate), b.Name)
	}
}

func printObjects(v []*s3.Item) {
	sort.Sort(s3.BySize(v))
	for _, b := range v {
		fmt.Printf("%s   %d %s\n", parseTime(b.LastModified), b.Size, b.Key)
	}
}

func printObject(v int64, date, key string) {
	fmt.Printf("%s   %d %s\n", parseLastModified(date), v, key)
}

func getBucketAndObject(p string) (bucket, object string) {
	parts := strings.Split(path.Clean(p), "/")
	switch true {
	case len(parts) == 2:
		bucket = parts[0]
		object = parts[1]
	case len(parts) < 2:
		bucket = parts[0]
		object = ""
	case len(parts) > 2:
		bucket = parts[0]
		for _, v := range parts[1 : len(parts)-1] {
			object = object + "/" + v
		}
	}
	return
}

func doFsList(c *cli.Context) {
	var err error
	var auth *s3.Auth
	var items []*s3.Item
	auth, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	var buckets []*s3.Bucket
	s3c := s3.NewS3Client(auth)
	switch len(c.Args()) {
	case 1:
		input := c.Args().Get(0)
		if path.IsAbs(input) {
			log.Fatal("Invalid bucket style")
		}
		bucket, object := getBucketAndObject(input)
		if object == "" {
			items, err = s3c.GetBucket(bucket, "", s3.MAX_OBJECT_LIST)
			if err != nil {
				log.Fatal(err)
			}
			printObjects(items)
		} else {
			var date string
			var size int64
			size, date, err = s3c.Stat(object, bucket)
			if err != nil {
				log.Fatal(err)
			}
			printObject(size, date, object)
		}
	default:
		buckets, err = s3c.Buckets()
		if err != nil {
			log.Fatal(err)
		}
		printBuckets(buckets)
	}
}
