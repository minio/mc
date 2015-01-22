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
	"github.com/minio-io/mc/pkg/s3"
)

func parseGetObjectInput(c *cli.Context) (bucket, key string, err error) {
	bucket = c.String("bucket")
	key = c.String("key")
	if bucket == "" {
		return "", "", bucketNameErr
	}
	if key == "" {
		return "", "", objectNameErr
	}

	return bucket, key, nil
}

func doGetObject(c *cli.Context) {
	var bucket, key string
	var err error
	var objectReader io.ReadCloser
	var objectSize int64

	var auth *s3.Auth
	auth, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	bucket, key, err = parseGetObjectInput(c)
	if err != nil {
		log.Fatal(err)
	}

	s3c := s3.NewS3Client(auth)
	objectReader, objectSize, err = s3c.Get(bucket, key)
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.CopyN(os.Stdout, objectReader, objectSize)
	if err != nil {
		log.Fatal(err)
	}
}
