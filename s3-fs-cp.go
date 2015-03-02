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
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
	"github.com/minio-io/mc/pkg/s3"
	"github.com/minio-io/mc/pkg/uri"
)

// TODO
//   - <S3Path> <S3Path>
//   - <S3Path> <S3Bucket>

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
			if uri.Path == "" {
				return fsOptions{}, fsKeyErr
			}
			fsoptions.key = strings.Trim(uri.Path, "/")
			if c.Args().Get(1) == "." {
				fsoptions.body = fsoptions.key
			} else {
				fsoptions.body = c.Args().Get(1)
			}
			fsoptions.isget = true
			fsoptions.isput = false
		} else if strings.HasPrefix(c.Args().Get(1), "s3://") {
			uri := uri.ParseURI(c.Args().Get(1))
			if uri.Scheme == "" {
				return fsOptions{}, fsUriErr
			}
			fsoptions.bucket = uri.Server
			if uri.Path == "" {
				fsoptions.key = c.Args().Get(0)
			} else {
				fsoptions.key = strings.Trim(uri.Path, "/")
			}
			fsoptions.body = c.Args().Get(0)
			fsoptions.isget = false
			fsoptions.isput = true
		}
	default:
		return fsOptions{}, fsPathErr
	}
	return
}

func startBar(size int64) *pb.ProgressBar {
	bar := pb.StartNew(int(size))
	bar.SetUnits(pb.U_BYTES)
	bar.SetRefreshRate(time.Millisecond * 10)
	bar.ShowSpeed = true
	return bar
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
		stat, err := os.Stat(fsoptions.body)
		if os.IsNotExist(err) {
			log.Fatal(err)
		}
		if stat.IsDir() {
			log.Fatal("Is a directory")
		}
		size := stat.Size()
		bodyFile, err = os.Open(fsoptions.body)
		defer bodyFile.Close()
		if err != nil {
			log.Fatal(err)
		}

		err = s3c.Put(fsoptions.bucket, fsoptions.key, nil, size, bodyFile)
		if err != nil {
			log.Fatal(err)
		}
	} else if fsoptions.isget {
		var objectReader io.ReadCloser
		var objectSize int64
		bodyFile, err = os.Create(fsoptions.body)
		defer bodyFile.Close()

		objectReader, objectSize, err = s3c.Get(fsoptions.bucket, fsoptions.key)
		if err != nil {
			log.Fatal(err)
		}

		// start progress bar
		bar := startBar(objectSize)

		// create multi writer to feed data
		writer := io.MultiWriter(bodyFile, bar)

		_, err = io.CopyN(writer, objectReader, objectSize)
		if err != nil {
			log.Fatal(err)
		}

		bar.FinishPrint("Done!")
	}
}
