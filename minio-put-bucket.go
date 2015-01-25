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
	"log"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/minio"
)

func minioPutBucket(c *cli.Context) {
	var bucket, hostname string
	var err error

	hostname, err = getMinioEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	bucket = c.String("bucket")
	if bucket == "" {
		log.Fatal(bucketNameErr)
	}

	mc := minio.NewMinioClient(hostname)
	err = mc.PutBucket(bucket)
	if err != nil {
		log.Fatal(err)
	}
}
