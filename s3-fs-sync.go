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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func isValidBucketName(p string) {
	if path.IsAbs(p) {
		fatal("directory bucketname cannot be absolute")
	}
	if strings.HasPrefix(p, "..") {
		fatal("Relative directory references not supported")
	}
	if !s3.IsValidBucket(p) {
		fatal(errInvalidbucket.Error())
	}
}

type walk struct {
	s3 *s3.Client
}

func (w *walk) putWalk(p string, i os.FileInfo, err error) error {
	if i.IsDir() {
		return nil
	}
	if !i.Mode().IsRegular() {
		return nil
	}
	parts := strings.SplitN(p, "/", 2)
	bucketname := parts[0]
	key := parts[1]

	bodyFile, err := os.Open(p)
	defer bodyFile.Close()
	err = w.s3.Put(bucketname, key, i.Size(), bodyFile)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("%s uploaded -- to bucket:%s", key, bucketname)
	info(msg)

	return nil
}

func doFsSync(c *cli.Context) {
	var s3c *s3.Client
	var err error
	config, err := getMcConfig()
	if err != nil {
		fatal(err.Error())
	}
	s3c, err = getNewClient(config)
	if err != nil {
		fatal(err.Error())
	}
	p := &walk{s3c}

	switch len(c.Args()) {
	case 1:
		input := path.Clean(c.Args().Get(0))
		isValidBucketName(input) // exit here if invalid

		fl, err := os.Stat(input)
		if os.IsNotExist(err) {
			fatal(err.Error())
		}
		if !fl.IsDir() {
			fatal("Should be a directory")
		}
		// Create bucketname, before uploading files
		err = s3c.PutBucket(input)
		if err != nil {
			fatal(err.Error())
		}
		err = filepath.Walk(input, p.putWalk)
		if err != nil {
			fatal(err.Error())
		}
	default:
		fatal("Requires a directory name <Directory>")
	}
}
