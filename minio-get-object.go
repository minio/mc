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
	"io"
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/minio"
)

func minioGetObject(c *cli.Context) {
	var bucket, key string
	var err error
	var objectReader io.ReadCloser
	var objectSize int64
	var auth *minio.Auth

	auth, err = getMinioEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	bucket = c.String("bucket")
	key = c.String("key")
	if bucket == "" {
		log.Fatal(bucketNameErr)
	}
	if key == "" {
		log.Fatal(objectNameErr)
	}

	minio, _ := minio.NewMinioClient(auth)
	objectReader, objectSize, err = minio.Get(bucket, key)
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.CopyN(os.Stdout, objectReader, objectSize)
	if err != nil {
		log.Fatal(err)
	}
}
