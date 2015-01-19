/*
 * Mini Object Storage, (C) 2014 Minio, Inc.
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
	"errors"
	"log"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func parseListObjectsInput(c *cli.Context) (bucket string, err error) {
	bucket = c.String("bucket")
	if bucket == "" {
		return "", errors.New("bucket name is mandatory")
	}
	return bucket, nil
}

func doListObjects(c *cli.Context) {
	var err error
	var auth *s3.Auth
	var bucket string

	auth, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	bucket, err = parseListObjectsInput(c)
	if err != nil {
		log.Fatal(err)
	}

	var items []*s3.Item
	s3c := s3.NewS3Client(auth)
	// Gets 1000 maxkeys supported with GET Bucket API
	items, err = s3c.GetBucket(bucket, "", s3.MAX_OBJECT_LIST)
	if err != nil {
		log.Fatal(err)
	}

	for _, item := range items {
		log.Println(item)
	}
}
