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

	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
)

// TODO
//   - <S3Path> <S3Path>
//   - <S3Path> <S3Bucket>

func doFsCopy(c *cli.Context) {
	s3c, err := getNewClient(c)
	if err != nil {
		fatal(err.Error())
	}

	if len(c.Args()) != 2 {
		fatal("Invalid number of args")
	}

	var cmdoptions *cmdOptions
	cmdoptions, err = parseOptions(c)
	if err != nil {
		fatal(err.Error())
	}

	switch true {
	case cmdoptions.isput == true:
		stat, err := os.Stat(cmdoptions.body)
		if os.IsNotExist(err) {
			fatal(err.Error())
		}
		if stat.IsDir() {
			fatal("Is a directory")
		}
		size := stat.Size()
		bodyFile, err := os.Open(cmdoptions.body)
		defer bodyFile.Close()
		if err != nil {
			fatal(err.Error())
		}

		// s3://<bucket> is specified without key
		if cmdoptions.key == "" {
			cmdoptions.key = cmdoptions.body
		}

		err = s3c.Put(cmdoptions.bucket, cmdoptions.key, size, bodyFile)
		if err != nil {
			fatal(err.Error())
		}
		msg := fmt.Sprintf("%s uploaded -- to bucket:(%s)", cmdoptions.key, cmdoptions.bucket)
		info(msg)
	case cmdoptions.isget == true:
		var objectReader io.ReadCloser
		var objectSize, downloadedSize int64
		var bodyFile *os.File
		var err error
		var st os.FileInfo

		// Send HEAD request to validate if file exists.
		objectSize, _, err = s3c.Stat(cmdoptions.key, cmdoptions.bucket)
		if err != nil {
			fatal(err.Error())
		}

		var bar *pb.ProgressBar
		if !c.GlobalBool("quiet") {
			// get progress bar
			bar = startBar(objectSize)
		}

		// Check if the object already exists
		st, err = os.Stat(cmdoptions.body)
		switch os.IsNotExist(err) {
		case true:
			// Create if it doesn't exist
			bodyFile, err = os.Create(cmdoptions.body)
			defer bodyFile.Close()
			if err != nil {
				fatal(err.Error())
			}
			objectReader, _, err = s3c.Get(cmdoptions.bucket, cmdoptions.key)
			if err != nil {
				fatal(err.Error())
			}
		case false:
			downloadedSize = st.Size()
			// Verify if file is already downloaded
			if downloadedSize == objectSize {
				msg := fmt.Sprintf("%s object has been already downloaded", cmdoptions.body)
				fatal(msg)
			}

			bodyFile, err = os.OpenFile(cmdoptions.body, os.O_RDWR, 0600)
			defer bodyFile.Close()

			if err != nil {
				fatal(err.Error())
			}

			_, err := bodyFile.Seek(downloadedSize, os.SEEK_SET)
			if err != nil {
				fatal(err.Error())
			}

			remainingSize := objectSize - downloadedSize
			objectReader, objectSize, err = s3c.GetPartial(cmdoptions.bucket, cmdoptions.key, downloadedSize, remainingSize)
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
