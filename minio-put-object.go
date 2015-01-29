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
	"io"
	"log"
	"os"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/minio"
)

func minioPutMetadata(reader io.Reader) (bodyBuf io.Reader, size int64, err error) {
	var length int
	var bodyBuffer bytes.Buffer

	for err == nil {
		byteBuffer := make([]byte, 1024*1024)
		length, err = reader.Read(byteBuffer)
		// It is necessary for us to verify this and break
		if length == 0 {
			break
		}
		byteBuffer = byteBuffer[0:length]
		_, err = bodyBuffer.Write(byteBuffer)
		if err != nil {
			break
		}
	}

	if err != io.EOF {
		return nil, 0, err
	}
	return &bodyBuffer, int64(bodyBuffer.Len()), nil
}

func minioParsePutObjectInput(c *cli.Context) (bucket, key, body string, err error) {
	bucket = c.String("bucket")
	key = c.String("key")
	body = c.String("body")

	if bucket == "" {
		return "", "", "", bucketNameErr
	}

	if key == "" {
		return "", "", "", objectNameErr
	}

	if body == "" {
		return "", "", "", objectBlobErr
	}

	return bucket, key, body, nil
}

func minioPutObject(c *cli.Context) {
	var err error
	var bucket, key, body string
	var auth *minio.Auth
	auth, err = getMinioEnvironment()
	if err != nil {
		log.Fatal(err)
	}

	bucket, key, body, err = minioParsePutObjectInput(c)
	if err != nil {
		log.Fatal(err)
	}
	mc, _ := minio.NewMinioClient(auth)
	var bodyFile *os.File
	bodyFile, err = os.Open(body)
	if err != nil {
		log.Fatal(err)
	}

	var bodyBuffer io.Reader
	var size int64
	bodyBuffer, size, err = minioPutMetadata(bodyFile)
	if err != nil {
		log.Fatal(err)
	}

	err = mc.Put(bucket, key, nil, size, bodyBuffer)
	if err != nil {
		log.Fatal(err)
	}
}
