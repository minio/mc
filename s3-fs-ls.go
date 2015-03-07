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
	"bufio"
	"bytes"
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

const (
	delimiter = '/'
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
		fmt.Printf("%s   %s\n", parseTime(b.CreationDate), b.Name)
	}
}

func printObjects(v []*s3.Item) {
	if len(v) > 0 {
		sort.Sort(s3.BySize(v))
		for _, b := range v {
			fmt.Printf("%s   %d %s\n", parseTime(b.LastModified), b.Size, b.Key)
		}
	}
}

func printPrefixes(v []*s3.Prefix) {
	if len(v) > 0 {
		for _, b := range v {
			fmt.Printf("                      PRE %s\n", b.Prefix)
		}
	}
}

func printObject(v int64, date, key string) {
	fmt.Printf("%s   %d %s\n", parseLastModified(date), v, key)
}

func getBucketAndObject(p string) (bucket, object string) {
	readBuffer := bytes.NewBufferString(p)
	reader := bufio.NewReader(readBuffer)
	pathPrefix, _ := reader.ReadString(byte(delimiter))
	bucket = path.Clean(pathPrefix)
	object = strings.TrimPrefix(p, pathPrefix)
	// if object is equal to bucket, set object to be empty
	if path.Clean(object) == bucket {
		object = ""
		return
	}
	return
}

func doFsList(c *cli.Context) {
	var items []*s3.Item
	var prefixes []*s3.Prefix

	config, err := getMcConfig()
	if err != nil {
		log.Fatal(err)
	}

	s3c, err := getNewClient(config)
	if err != nil {
		log.Fatal(err)
	}
	switch len(c.Args()) {
	case 1:
		input := c.Args().Get(0)
		if path.IsAbs(input) {
			log.Fatal("Invalid bucket style")
		}
		bucket, object := getBucketAndObject(input)
		if object == "" {
			items, prefixes, err = s3c.GetBucket(bucket, "", "", string(delimiter), s3.MaxKeys)
			if err != nil {
				log.Fatal(err)
			}
			printPrefixes(prefixes)
			printObjects(items)
		} else {
			var date string
			var size int64
			size, date, err = s3c.Stat(object, bucket)
			if err != nil {
				items, prefixes, err = s3c.GetBucket(bucket, "", object, string(delimiter), s3.MaxKeys)
				if err != nil {
					log.Fatal(err)
				}
				printPrefixes(prefixes)
				printObjects(items)
			} else {
				printObject(size, date, object)
			}
		}
	default:
		buckets, err := s3c.Buckets()
		if err != nil {
			log.Fatal(err)
		}
		printBuckets(buckets)
	}
}
