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

	"github.com/minio-io/mc/pkg/s3"
)

type walk struct {
	s3   *s3.Client
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

	err = w.s3.Put(bucketname, key, i.Size(), bodyFile)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("%s uploaded -- to bucket:http://%s/%s", key, bucketname, key)
	info(msg)
	return nil
}

func isValidBucketName(p string) error {
	if path.IsAbs(p) {
		return fmt.Errorf("directory bucketname cannot be absolute")
	}
	if strings.HasPrefix(p, "..") {
		return fmt.Errorf("Relative directory references not supported")
	}
	if !s3.IsValidBucket(p) {
		return errInvalidbucket
	}
	return nil
}

func isBucketExists(name string, v []*s3.Bucket) bool {
	for _, b := range v {
		if name == b.Name {
			return true
		}
	}
	return false
}

func doRecursiveCp(s3c *s3.Client, args *cmdArgs) error {
	var err error
	var st os.FileInfo
	var buckets []*s3.Bucket

	p := &walk{s3c, args}

	switch true {
	case args.source.bucket == "":
		input := path.Clean(args.source.key)
		if err := isValidBucketName(input); err != nil {
			return err
		}
		st, err = os.Stat(input)
		if os.IsNotExist(err) {
			return err
		}
		if !st.IsDir() {
			return fmt.Errorf("Should be a directory")
		}

		buckets, err = s3c.ListBuckets()
		if !isBucketExists(args.destination.bucket, buckets) {
			// Create bucketname, before uploading files
			err = s3c.PutBucket(args.destination.bucket)
			if err != nil {
				return err
			}
		} else {
			items, _, err := s3c.ListObjects(args.destination.bucket, "", "", "", s3.MaxKeys)
			if err != nil {
				return err
			}
			if len(items) != 0 {
				return fmt.Errorf("destination bucket not empty")
			}
		}
		err = filepath.Walk(input, p.putWalk)
		if err != nil {
			return err
		}
	case args.destination.bucket == "":
		items, _, err := s3c.ListObjects(args.source.bucket, "", "", "", s3.MaxKeys)
		if err != nil {
			return err
		}
		root := args.destination.key
		writeObjects := func(v []*s3.Item) error {
			if len(v) > 0 {
				// Items are already sorted
				for _, b := range v {
					args.source.key = b.Key
					os.MkdirAll(path.Join(root, path.Dir(b.Key)), 0755)
					args.destination.key = path.Join(root, b.Key)
					err := secondMode(s3c, args)
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
