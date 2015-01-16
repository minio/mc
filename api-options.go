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
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

var GetObject = cli.Command{
	Name:        "get-object",
	Usage:       "",
	Description: "",
	Action:      doGetObject,
}

var PutObject = cli.Command{
	Name:        "put-object",
	Usage:       "",
	Description: "",
	Action:      doPutObject,
}

var ListObjects = cli.Command{
	Name:        "list-objects",
	Usage:       "",
	Description: "",
	Action:      doListObjects,
}

var ListBuckets = cli.Command{
	Name:        "list-buckets",
	Usage:       "",
	Description: "",
	Action:      doListBuckets,
}

var Configure = cli.Command{
	Name:        "configure",
	Usage:       "",
	Description: "",
	Action:      doConfigure,
}

func doGetObject(c *cli.Context) {
	var bucket, key string
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if accessKey == "" && secretKey == "" {
		errstr := `You can configure your credentials by running "mc configure"`
		log.Fatal(errstr)
	}
	if accessKey == "" {
		errstr := `Partial credentials found in the env, missing : AWS_ACCESS_KEY_ID`
		log.Fatal(errstr)
	}

	if secretKey == "" {
		errstr := `Partial credentials found in the env, missing : AWS_SECRET_ACCESS_KEY`
		log.Fatal(errstr)
	}

	s3c := s3.NewS3Client(accessKey, secretKey)
	_, _, err := s3c.Get(bucket, key)
	if err != nil {
		log.Fatal(err)
	}
}

func doPutObject(c *cli.Context) {
}

func doListObject(c *cli.Context) {
}

func doListObjects(c *cli.Context) {
}

func doListBuckets(c *cli.Context) {
}

func doConfigure(c *cli.Context) {
}
