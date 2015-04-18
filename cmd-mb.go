/*
 * Mini Copy, (C) 2014,2015 Minio, Inc.
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
	"time"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

// doMakeBucketCmd creates a new bucket
func doMakeBucketCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		cli.ShowCommandHelpAndExit(ctx, "mb", 1) // last argument is exit code
	}
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("mc: Unable to read config")
	}
	for _, arg := range ctx.Args() {
		var err error
		var clnt client.Client

		u, err := parseURL(arg, config.GetMapString("Aliases"))
		if err != nil {
			switch iodine.ToError(err).(type) {
			case errUnsupportedScheme:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("mc: Unable to parse URL [%s], %s\n", arg, client.GuessPossibleURL(arg))
			default:
				log.Debug.Println(iodine.New(err, nil))
				console.Fatalf("mc: Unable to parse URL [%s]\n", arg)
			}
		}

		manager := mcClientManager{}
		clnt, err = manager.getNewClient(u, globalDebugFlag)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: instantiating a new client for URL [%s] failed with following reason: [%s]\n", u, iodine.ToError(err))
		}

		bucket, _, err := client.URL2Object(u)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: decoding bucket and object from URL [%s] failed\n", u)
		}

		// this is handled differently since http based URLs cannot have
		// nested directories as buckets, buckets are a unique alphanumeric
		// name having subdirectories is only supported for fsClient
		if client.GetURLType(u) != client.URLFilesystem {
			if bucket == "" {
				err := iodine.New(errBucketNameEmpty{}, nil)
				log.Debug.Println(err)
				console.Fatalf("mc: Creating bucket failed for URL [%s] with following reason: [%s]\n", u, iodine.ToError(err))
			}
			if !client.IsValidBucketName(bucket) {
				err := iodine.New(errInvalidBucketName{bucket: bucket}, nil)
				log.Debug.Println(err)
				console.Fatalf("mc: Creating bucket failed for URL [%s] with following reason: [%s]\n", u, iodine.ToError(err))
			}
		}

		err = clnt.PutBucket(bucket)
		for i := 0; i < globalMaxRetryFlag && err != nil; i++ {
			err = clnt.PutBucket(bucket)
			// Progressively longer delays
			time.Sleep(time.Duration(i*i) * time.Second)
		}
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: Creating bucket failed for URL [%s] with following reason: [%s]\n", u, iodine.ToError(err))
		}
	}
}
