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
)

var cp = cli.Command{
	Name:        "cp",
	Usage:       "",
	Description: `Copies a local file or Object to another location locally or in S3.`,
	Action:      doFsCopy,
}

var ls = cli.Command{
	Name:        "ls",
	Usage:       "",
	Description: `List Objects and common prefixes under a prefix or all Buckets`,
	Action:      doFsList,
}

var mb = cli.Command{
	Name:        "mb",
	Usage:       "",
	Description: "Creates an S3 bucket",
	Action:      doFsMb,
}

var sync = cli.Command{
	Name:        "sync",
	Usage:       "",
	Description: "Syncs directories and S3 prefixes",
	Action:      doFsSync,
}

var configure = cli.Command{
	Name:  "configure",
	Usage: "",
	Description: `Configure minio client configuration data. If your config
   file does not exist (the default location is ~/.auth), it will be
   automatically created for you. Note that the configure command only writes
   values to the config file. It does not use any configuration values from
   the environment variables.`,
	Action: doConfigure,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "accesskey, a",
			Value: "",
			Usage: "AWS access key id",
		},
		cli.StringFlag{
			Name:  "secretkey, s",
			Value: "",
			Usage: "AWS secret key id",
		},
		cli.StringFlag{
			Name:  "endpoint, e",
			Value: "s3.amazonaws.com",
			Usage: "S3 Endpoint URL default is 's3.amazonaws.com'",
		},
		cli.BoolFlag{
			Name:  "pathstyle, p",
			Usage: "Force path style API requests",
		},
	},
}

type fsOptions struct {
	bucket string
	body   string
	key    string
	isget  bool
	isput  bool
}

var options = []cli.Command{
	cp,
	ls,
	mb,
	sync,
	configure,
}
