/*
 * Modern Copy, (C) 2014,2015 Minio, Inc.
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
	"strings"

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
		console.Fatalln("Unable to get config")
	}
	for _, arg := range ctx.Args() {
		urlStr, err := parseURL(arg, config.Aliases)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorln(err)
		}
		bucket, err := url2Bucket(urlStr)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorln(err)
		}
		clnt, err := getNewClient(urlStr, globalDebugFlag)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorln(err)
		}
		if !strings.HasPrefix(urlStr, "file:") {
			if !client.IsValidBucketName(bucket) {
				log.Debug.Println(iodine.New(err, nil))
				console.Errorln(errInvalidBucket{bucket: bucket})
			}
		}
		err = clnt.PutBucket(bucket)
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorln(err)
		}
	}
}
