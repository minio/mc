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
	"os"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
)

// TODO
//   - <S3Path> <S3Path>
//   - <S3Path> <S3Bucket>

func startBar(size int64) *pb.ProgressBar {
	bar := pb.New(int(size))
	bar.SetUnits(pb.U_BYTES)
	bar.SetRefreshRate(time.Millisecond * 10)
	bar.NotPrint = true
	bar.ShowSpeed = true
	bar.Callback = func(s string) {
		// Colorize
		infoCallback(s)
	}
	// Feels like wget
	bar.Format("[=> ]")
	return bar
}

func doFsCopy(c *cli.Context) {
	s3c, err := getNewClient(c)
	if err != nil {
		fatal(err.Error())
	}

	if len(c.Args()) != 2 {
		fatal("Invalid number of args")
	}

	var fsoptions *fsOptions
	fsoptions, err = parseOptions(c)
	if err != nil {
		fatal(err.Error())
	}

	switch true {
	case fsoptions.isput == true:
		stat, err := os.Stat(fsoptions.body)
		if os.IsNotExist(err) {
			fatal(err.Error())
		}
		if stat.IsDir() {
			fatal("Is a directory")
		}
		size := stat.Size()
		bodyFile, err := os.Open(fsoptions.body)
		defer bodyFile.Close()
		if err != nil {
			fatal(err.Error())
		}

		// s3://<bucket> is specified without key
		if fsoptions.key == "" {
			fsoptions.key = fsoptions.body
		}

		err = s3c.Put(fsoptions.bucket, fsoptions.key, size, bodyFile)
		if err != nil {
			fatal(err.Error())
		}
		msg := fmt.Sprintf("%s uploaded -- to bucket:(%s)", fsoptions.key, fsoptions.bucket)
		info(msg)
	case fsoptions.isget == true:
		var objectReader io.ReadCloser
		var objectSize, downloadedSize int64
		var bodyFile *os.File
		var err error
		var st os.FileInfo

		// Send HEAD request to validate if file exists.
		objectSize, _, err = s3c.Stat(fsoptions.key, fsoptions.bucket)
		if err != nil {
			fatal(err.Error())
		}

		var bar *pb.ProgressBar
		if !c.GlobalBool("quiet") {
			// get progress bar
			bar = startBar(objectSize)
		}

		// Check if the object already exists
		st, err = os.Stat(fsoptions.body)
		switch os.IsNotExist(err) {
		case true:
			// Create if it doesn't exist
			bodyFile, err = os.Create(fsoptions.body)
			defer bodyFile.Close()
			if err != nil {
				fatal(err.Error())
			}
			objectReader, _, err = s3c.Get(fsoptions.bucket, fsoptions.key)
			if err != nil {
				fatal(err.Error())
			}
		case false:
			downloadedSize = st.Size()
			// Verify if file is already downloaded
			if downloadedSize == objectSize {
				msg := fmt.Sprintf("%s object has been already downloaded", fsoptions.body)
				fatal(msg)
			}

			bodyFile, err = os.OpenFile(fsoptions.body, os.O_RDWR, 0600)
			defer bodyFile.Close()

			if err != nil {
				fatal(err.Error())
			}

			_, err := bodyFile.Seek(downloadedSize, os.SEEK_SET)
			if err != nil {
				fatal(err.Error())
			}

			remainingSize := objectSize - downloadedSize
			objectReader, objectSize, err = s3c.GetPartial(fsoptions.bucket, fsoptions.key, downloadedSize, remainingSize)
			if err != nil {
				fatal(err.Error())
			}

			if !c.GlobalBool("quiet") {
				bar.Set(int(downloadedSize))
			}
		}

		writer := io.Writer(bodyFile)
		if !c.GlobalBool("quiet") {
			// Start the bar now
			bar.Start()
			// create multi writer to feed data
			writer = io.MultiWriter(bodyFile, bar)
		}
		_, err = io.CopyN(writer, objectReader, objectSize)
		if err != nil {
			fatal(err.Error())
		}

		bar.Finish()
		info("Success!")
	}
}
