/*
 * Minimalist Object Storage, (C) 2015 Minio, Inc.
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
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/s3"
)

type walk struct {
	s3   client.Client
	args *cmdArgs
}

func (w *walk) putWalk(p string, i os.FileInfo, err error) error {
	if i.IsDir() {
		return nil
	}
	if !i.Mode().IsRegular() {
		return nil
	}
	parts := strings.SplitN(p, "/", 2)
	bucketname := w.args.destination.bucket
	key := parts[1]

	bodyFile, err := os.Open(p)
	defer bodyFile.Close()

	var size int64
	size, _, err = w.s3.Stat(bucketname, key)
	if os.IsExist(err) || size != 0 {
		msg := fmt.Sprintf("%s is already uploaded -- to bucket:%s/%s/%s",
			key, w.args.destination.host, bucketname, key)
		info(msg)
		return nil
	}
	err = w.s3.Put(bucketname, key, i.Size(), bodyFile)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("%s uploaded -- to bucket:%s/%s/%s",
		key, w.args.destination.host, bucketname, key)
	info(msg)
	return nil
}

// isBucketExist checks if a bucket exists
func isBucketExist(bucketName string, v []*client.Bucket) bool {
	for _, b := range v {
		if bucketName == b.Name {
			return true
		}
	}
	return false
}

// doRecursiveCP recursively copies objects from source to destination
func doRecursiveCP(c *cli.Context, args *cmdArgs) error {
	var err error
	var st os.FileInfo
	var buckets []*client.Bucket

	switch true {
	case args.source.bucket == "":
		input := path.Clean(args.source.key)
		if s3.IsValidBucketName(input) {
			return fmt.Errorf("Invalid input bucket name [%s]", input)
		}
		st, err = os.Stat(input)
		if os.IsNotExist(err) {
			return err
		}
		if !st.IsDir() {
			return errors.New("Should be a directory")
		}
		s3c, err := getNewClient(c.GlobalBool("debug"), args.destination.url.String())
		if err != nil {
			return err
		}
		p := &walk{s3c, args}
		buckets, err = s3c.ListBuckets()
		if !isBucketExist(args.destination.bucket, buckets) {
			// Create bucketname, before uploading files
			err = s3c.PutBucket(args.destination.bucket)
			if err != nil {
				return err
			}
		}
		err = filepath.Walk(input, p.putWalk)
		if err != nil {
			return err
		}
	case args.destination.bucket == "":
		s3c, err := getNewClient(c.GlobalBool("debug"), args.source.url.String())
		if err != nil {
			return err
		}
		items, _, err := s3c.ListObjects(args.source.bucket, "", "", "", s3.MaxKeys)
		if err != nil {
			return err
		}
		root := args.destination.key
		writeObjects := func(v []*client.Item) error {
			if len(v) > 0 {
				// Items are already sorted
				for _, b := range v {
					args.source.key = b.Key
					os.MkdirAll(path.Join(root, path.Dir(b.Key)), 0755)
					args.destination.key = path.Join(root, b.Key)
					err := secondMode(c, args)
					if err != nil {
						return err
					}
				}
			}
			return nil
		}
		err = writeObjects(items)
		if err != nil {
			return err
		}
	}
	return nil
}
