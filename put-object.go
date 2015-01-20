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
	"bytes"
	"crypto/md5"
	"errors"
	"hash"
	"io"
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
)

func getPutMetadata(reader io.Reader) (md5hash hash.Hash, bodyBuf io.Reader, size int64, err error) {
	md5hash = md5.New()
	var length int
	var bodyBuffer bytes.Buffer

	for err == nil {
		byteBuffer := make([]byte, 1024*1024)
		length, err = reader.Read(byteBuffer)
		// While hash.Write() wouldn't mind a Nil byteBuffer
		// It is necessary for us to verify this and break
		if length == 0 {
			break
		}
		byteBuffer = byteBuffer[0:length]
		_, err = bodyBuffer.Write(byteBuffer)
		if err != nil {
			break
		}
		md5hash.Write(byteBuffer)
	}
	if err != io.EOF {
		return nil, nil, 0, err
	}
	return md5hash, &bodyBuffer, int64(bodyBuffer.Len()), nil
}

func parsePutObjectInput(c *cli.Context) (bucket, key, body string, err error) {
	bucket = c.String("bucket")
	key = c.String("key")
	body = c.String("body")

	if bucket == "" {
		return "", "", "", errors.New("bucket name is mandatory")
	}

	if key == "" {
		return "", "", "", errors.New("object name is mandatory")
	}

	if body == "" {
		return "", "", "", errors.New("object blob is mandatory")
	}

	return bucket, key, body, nil
}

func doPutObject(c *cli.Context) {
	var err error
	var md5hash hash.Hash
	var bucket, key, body string
	var auth *s3.Auth
	auth, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	bucket, key, body, err = parsePutObjectInput(c)
	if err != nil {
		log.Fatal(err)
	}
	s3c := s3.NewS3Client(auth)
	var bodyFile *os.File
	bodyFile, err = os.Open(body)
	if err != nil {
		log.Fatal(err)
	}

	var bodyBuffer io.Reader
	var size int64
	md5hash, bodyBuffer, size, err = getPutMetadata(bodyFile)
	if err != nil {
		log.Fatal(err)
	}

	err = s3c.Put(bucket, key, md5hash, size, bodyBuffer)
	if err != nil {
		log.Fatal(err)
	}
}
