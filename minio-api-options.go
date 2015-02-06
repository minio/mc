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

var MinioGetObject = cli.Command{
	Name:        "get-object",
	Usage:       "",
	Description: "Retrieves objects from Amazon S3.",
	Action:      minioGetObject,
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

var MinioPutBucket = cli.Command{
	Name:        "put-object",
	Usage:       "",
	Description: "Adds an object to a bucket.",
	Action:      minioPutObject,
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
			Usage: "Object blob",
		},
	},
}

var MinioPutObject = cli.Command{
	Name:        "put-bucket",
	Usage:       "",
	Description: "Creates a new bucket.",
	Action:      minioPutBucket,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bucket",
			Value: "",
			Usage: "bucket name",
		},
	},
}

var MinioListObjects = cli.Command{
	Name:  "list-objects",
	Usage: "",
	Description: `Returns some or all (up to 1000) of the objects in a bucket.
   You can use the request parameters as selection criteria to
   return a subset of the objects in a bucket.`,
	Action: minioListObjects,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "bucket",
			Value: "",
			Usage: "Bucket name",
		},
	},
}

var MinioListBuckets = cli.Command{
	Name:  "list-buckets",
	Usage: "",
	Description: `Returns a list of all buckets owned by the authenticated
   sender of the request.`,
	Action: minioListBuckets,
}

var MinioConfigure = cli.Command{
	Name:  "configure",
	Usage: "",
	Description: `Configure minio client configuration data. If your config
   file does not exist (the default location is ~/.auth), it will be
   automatically created for you. Note that the configure command only writes
   values to the config file. It does not use any configuration values from
   the environment variables.`,
	Action: doMinioConfigure,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "hostname",
			Value: "127.0.0.1:8080",
			Usage: "Minio object server",
		},
		cli.StringFlag{
			Name:  "accesskey",
			Value: "",
			Usage: "Minio access key",
		},
		cli.StringFlag{
			Name:  "secretKey",
			Value: "",
			Usage: "Minio secret key",
		},
		cli.StringFlag{
			Name:  "cacert",
			Value: "",
			Usage: "CA authority cert",
		},
		cli.StringFlag{
			Name:  "cert",
			Value: "",
			Usage: "Minio server certificate",
		},
		cli.StringFlag{
			Name:  "key",
			Value: "",
			Usage: "Minio server private key",
		},
	},
}

const (
	MINIO_AUTH = ".minioauth"
)
