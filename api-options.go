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
	"github.com/codegangsta/cli"
)

var GetObject = cli.Command{
	Name:        "get-object",
	Usage:       "",
	Description: "",
	Action:      doGetObject,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bucket",
			Value: "",
			Usage: "bucket name",
		},
		cli.StringFlag{
			Name:  "key",
			Value: "",
			Usage: "path to Object",
		},
	},
}

var PutObject = cli.Command{
	Name:        "put-object",
	Usage:       "",
	Description: "",
	Action:      doPutObject,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bucket",
			Value: "",
			Usage: "bucket name",
		},
		cli.StringFlag{
			Name:  "key",
			Value: "",
			Usage: "Object name",
		},
		cli.StringFlag{
			Name:  "body",
			Value: "",
			Usage: "Object blobx",
		},
	},
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

func doListObject(c *cli.Context) {
}

func doListObjects(c *cli.Context) {
}

func doListBuckets(c *cli.Context) {
}

func doConfigure(c *cli.Context) {
}
