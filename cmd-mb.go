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
	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

// doMakeBucketCmd creates a new bucket
func doMakeBucketCmd(c *cli.Context) {
	args, err := parseArgs(c)
	if err != nil {
		fatal(err.Error())
	}

	s3c, err := getNewClient(c)

	if !s3.IsValidBucketName(args.source.bucket) {
		fatal(errInvalidbucket.Error())
	}
	s3c.Host = args.source.host
	s3c.Scheme = args.source.scheme

	err = s3c.PutBucket(args.source.bucket)
	if err != nil {
		fatal(err.Error())
	}
}
