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
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
)

// TODO
//   - <S3Path> <S3Path>
//   - <S3Path> <S3Bucket>

func startBar(size int64) *pb.ProgressBar {
	bar := pb.StartNew(int(size))
	bar.SetUnits(pb.U_BYTES)
	bar.SetRefreshRate(time.Millisecond * 10)
	bar.ShowSpeed = true
	return bar
}

func doFsCopy(c *cli.Context) {
	mcConfig, err := getMcConfig()
	if err != nil {
		log.Fatal(err)
	}
	s3c, err := getNewClient(mcConfig)
	if err != nil {
		log.Fatal(err)
	}

	var fsoptions *fsOptions
	fsoptions, err = parseOptions(c)
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
		bodyFile, err := os.Open(fsoptions.body)
		defer bodyFile.Close()
		if err != nil {
			log.Fatal(err)
		}

		// s3://<bucket> is specified without key
		if fsoptions.key == "" {
			fsoptions.key = fsoptions.body
		}

		err = s3c.Put(fsoptions.bucket, fsoptions.key, size, bodyFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s uploaded -- to bucket:%s\n", fsoptions.key, fsoptions.bucket)
	} else if fsoptions.isget {
		var objectReader io.ReadCloser
		var objectSize int64

		objectReader, objectSize, err = s3c.Get(fsoptions.bucket, fsoptions.key)
		if err != nil {
			log.Fatal(err)
		}
		bodyFile, err := os.Create(fsoptions.body)
		defer bodyFile.Close()

		// start progress bar
		bar := startBar(objectSize)

		// create multi writer to feed data
		writer := io.MultiWriter(bodyFile, bar)

		_, err = io.CopyN(writer, objectReader, objectSize)
		if err != nil {
			log.Fatal(err)
		}

		bar.FinishPrint(fsoptions.body + " downloaded!")
	}
}
