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
	"strings"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

type MinioClient struct {
	bucketName string
	keyName    string
	body       string
	bucketAcls string
	policy     string
	region     string
	query      string // TODO
}

var Options = []cli.Command{
	GetObject,
	PutObject,
	ListObjects,
	ListBuckets,
	Configure,
}

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

func parseInput(c *cli.Context) string {
	var commandName string
	switch len(c.Args()) {
	case 1:
		commandName = c.Args()[0]
	default:
		log.Fatal("command name must not be blank\n")
	}

	var inputOptions []string
	if c.String("bucket") != "" {
		inputOptions = strings.Split(c.String("options"), ",")
	}

	if inputOptions[0] == "" {
		log.Fatal("options cannot be empty with a command name")
	}
	return commandName
}

func doGetObject(c *cli.Context) {
	var bucket, key string
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_ACCESS_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		log.Fatal("no AWS_ACCESS_KEY_ID or AWS_ACCESS_KEY_SECRET set in environment")
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
