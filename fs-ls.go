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

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func printBuckets(v []*s3.Bucket) {
	for _, b := range v {
		fmt.Printf("%s %s\n", b.CreationDate, b.Name)
	}
}

func printObjects(v []*s3.Item) {
	for _, b := range v {
		fmt.Printf("%s %s\n", b.LastModified, b.Key)
	}
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
		items, err = s3c.GetBucket(c.Args().Get(0), "", s3.MAX_OBJECT_LIST)
		if err != nil {
			log.Fatal(err)
		}
		printObjects(items)
	default:
		buckets, err = s3c.Buckets()
		printBuckets(buckets)
	}
}
