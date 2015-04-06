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
	"github.com/minio-io/cli"
	"github.com/minio-io/iodine"
	"github.com/minio-io/mc/pkg/client/s3"
	"github.com/minio-io/minio/pkg/utils/log"
)

// doMakeBucketCmd creates a new bucket
func doMakeBucketCmd(ctx *cli.Context) {
	urlStr, err := parseURL(ctx.Args().First())
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		fatal(err.Error())
	}

	bucket, err := url2Bucket(urlStr)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		fatal(err.Error())
	}

	clnt, err := getNewClient(globalDebugFlag, urlStr)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		fatal(err.Error())
	}

	if !s3.IsValidBucketName(bucket) {
		log.Debug.Println(iodine.New(err, nil))
		fatal(errInvalidbucket.Error())
	}

	err = clnt.PutBucket(bucket)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		fatal(err.Error())
	}
}
