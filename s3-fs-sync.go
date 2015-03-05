/*
 * Mini Object Storage, (C) 2015 Minio, Inc.
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
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func isValidBucketName(p string) {
	if path.IsAbs(p) {
		log.Fatal("directory bucketname cannot be absolute")
	}
	if strings.HasPrefix(p, "..") {
		log.Fatal("Relative directory references not supported")
	}
	if !s3.IsValidBucket(p) {
		log.Fatal(errInvalidbucket)
	}
}

type walk struct {
	s3 *s3.Client
}

func (w *walk) putWalk(p string, info os.FileInfo, err error) error {
	if info.IsDir() {
		return nil
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	parts := strings.SplitN(p, "/", 2)
	bucketname := parts[0]
	key := parts[1]

	bodyFile, err := os.Open(p)
	defer bodyFile.Close()
	err = w.s3.Put(bucketname, key, nil, info.Size(), bodyFile)
	if err != nil {
		return err
	}
	fmt.Printf("%s uploaded -- to bucket:%s\n", key, bucketname)
	return nil
}

func doFsSync(c *cli.Context) {
	var auth *s3.Auth
	var s3c *s3.Client
	var err error
	auth, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	s3c, err = getNewClient(auth)
	if err != nil {
		log.Fatal(err)
	}
	p := &walk{s3c}

	switch len(c.Args()) {
	case 1:
		input := path.Clean(c.Args().Get(0))
		isValidBucketName(input) // exit here if invalid

		fl, err := os.Stat(input)
		if os.IsNotExist(err) {
			log.Fatal(err)
		}
		if !fl.IsDir() {
			log.Fatal("Should be a directory")
		}
		// Create bucketname, before uploading files
		err = s3c.PutBucket(input)
		if err != nil {
			log.Fatal(err)
		}
		err = filepath.Walk(input, p.putWalk)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("Requires a directory name <Directory>")
	}
}
