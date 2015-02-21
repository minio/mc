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
	"hash"
	"io"
	"log"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
	"github.com/minio-io/mc/pkg/uri"
)

// TODO
//   - <S3Path> <S3Path>
//   - <S3Path> <S3Bucket>
//   - <LocalDir> <S3Bucket>

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

func parseCpOptions(c *cli.Context) (fsoptions fsOptions, err error) {
	switch len(c.Args()) {
	case 1:
		return fsOptions{}, fsPathErr
	case 2:
		if strings.HasPrefix(c.Args().Get(0), "s3://") {
			uri := uri.ParseURI(c.Args().Get(0))
			if uri.Scheme == "" {
				return fsOptions{}, fsUriErr
			}
			fsoptions.bucket = uri.Server
			fsoptions.key = strings.Trim(uri.Path, "/")
			fsoptions.body = c.Args().Get(1)
			fsoptions.isget = true
			fsoptions.isput = false
		} else if strings.HasPrefix(c.Args().Get(1), "s3://") {
			uri := uri.ParseURI(c.Args().Get(1))
			if uri.Scheme == "" {
				return fsOptions{}, fsUriErr
			}
			fsoptions.bucket = uri.Server
			fsoptions.key = strings.Trim(uri.Path, "/")
			fsoptions.body = c.Args().Get(0)
			fsoptions.isget = false
			fsoptions.isput = true
		}
	default:
		return fsOptions{}, fsPathErr
	}
	return
}

func doFsCopy(c *cli.Context) {
	var auth *s3.Auth
	var err error
	var bodyFile *os.File
	auth, err = getAWSEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	s3c := s3.NewS3Client(auth)

	var fsoptions fsOptions
	fsoptions, err = parseCpOptions(c)
	if err != nil {
		log.Fatal(err)
	}

	if fsoptions.isput {
		bodyFile, err = os.Open(fsoptions.body)
		defer bodyFile.Close()
		if err != nil {
			log.Fatal(err)
		}

		var bodyBuffer io.Reader
		var size int64
		var md5hash hash.Hash
		md5hash, bodyBuffer, size, err = getPutMetadata(bodyFile)
		if err != nil {
			log.Fatal(err)
		}

		err = s3c.Put(fsoptions.bucket, fsoptions.key, md5hash, size, bodyBuffer)
		if err != nil {
			log.Fatal(err)
		}
	} else if fsoptions.isget {
		var objectReader io.ReadCloser
		var objectSize int64
		bodyFile, err = os.OpenFile(fsoptions.body, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		defer bodyFile.Close()

		objectReader, objectSize, err = s3c.Get(fsoptions.bucket, fsoptions.key)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.CopyN(bodyFile, objectReader, objectSize)
		if err != nil {
			log.Fatal(err)
		}
	}
}
