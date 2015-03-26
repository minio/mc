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
	"os"

	"net/url"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client/donut"
)

// doMakeDonutBucketCmd creates a new bucket
func doMakeDonutBucketCmd(c *cli.Context) {
	os.MkdirAll("testdir", 0755)
	urlArg1, err := url.Parse(c.Args().Get(0))
	if err != nil {
		fatal(err.Error())
	}
	d := donut.GetNewClient(urlArg1.Host)
	err = d.PutBucket(urlArg1.Path)
	if err != nil {
		fatal(err.Error())
	}
}
