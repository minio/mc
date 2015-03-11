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
	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func doFsMb(c *cli.Context) {
	switch len(c.Args()) {
	case 1:
		if !s3.IsValidBucket(c.Args().Get(0)) {
			fatal(errInvalidbucket.Error())
		}
	default:
		fatal("Needs an argument <BucketName>")
	}
	bucketName := c.Args().Get(0)

	s3c, err := getNewClient(c)
	err = s3c.PutBucket(bucketName)
	if err != nil {
		fatal(err.Error())
	}
}
