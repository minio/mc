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
	"github.com/minio-io/mc/pkg/s3"
)

// Different modes of cp operation
const (
	first  = iota // <Path> <S3Path> or <Path> <S3Bucket>
	second        // <S3Path> <Path> or <S3Path> .
	third         // <S3Path> <S3Path> or <S3Path> <S3Bucket>
	fourth        // <S3Bucket> <S3Bucket> - TODO
	invalid
)

func getMode(options *cmdOptions) int {
	switch true {
	// <Path> <S3Path> or <Path> <S3Bucket>
	case options.source.bucket == "" && options.destination.bucket != "":
		return first
	// <S3Path> <Path> or <S3Path> .
	case options.source.bucket != "" && options.destination.bucket == "":
		return second
	// <S3Path> <S3Path> or <S3Path> <S3Bucket>
	case options.source.bucket != "" && options.destination.bucket != "" && options.source.key != "":
		return third
	}
	return invalid
}

// First mode <Path> <S3Path> or <Path> <S3Bucket>
func firstMode(s3c *s3.Client, options *cmdOptions) error {
	if options.source.key == "" {
		return fmt.Errorf("invalid")
	}
	st, err := os.Stat(options.source.key)
	if os.IsNotExist(err) {
		return err
	}
	if st.IsDir() {
		fmt.Errorf("Is a directory")
	}
	size := st.Size()
	source, err := os.Open(options.source.key)
	defer source.Close()
	if err != nil {
		return err
	}

	// s3://<bucket> is specified without key
	if options.destination.key == "" {
		options.destination.key = options.source.key
	}

	err = s3c.Put(options.destination.bucket, options.destination.key, size, source)
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("%s uploaded -- to bucket:(s3://%s/%s)", options.source.key,
		options.destination.bucket, options.destination.key)
	info(msg)
	return nil
}

// Second mode <S3Path> <Path> or <S3Path> .
func secondMode(s3c *s3.Client, options *cmdOptions) error {
	var objectReader io.ReadCloser
	var objectSize, downloadedSize int64
	var destination *os.File
	var err error
	var st os.FileInfo

	// Send HEAD request to validate if file exists.
	objectSize, _, err = s3c.Stat(options.source.bucket, options.source.key)
	if err != nil {
		return err
	}

	if options.destination.key == "." {
		options.destination.key = options.source.key
	}

	var bar *pb.ProgressBar
	if !options.quiet {
		// get progress bar
		bar = startBar(objectSize)
	}

	// Check if the object already exists
	st, err = os.Stat(options.destination.key)
	switch os.IsNotExist(err) {
	case true:
		// Create if it doesn't exist
		destination, err = os.Create(options.destination.key)
		defer destination.Close()
		if err != nil {
			return err
		}
		objectReader, _, err = s3c.Get(options.source.bucket, options.source.key)
		if err != nil {
			return err
		}
	case false:
		downloadedSize = st.Size()
		// Verify if file is already downloaded
		if downloadedSize == objectSize {
			return fmt.Errorf("%s object has been already downloaded", options.destination.key)
		}

		destination, err = os.OpenFile(options.destination.key, os.O_RDWR, 0600)
		defer destination.Close()
		if err != nil {
			return err
		}

		_, err := destination.Seek(downloadedSize, os.SEEK_SET)
		if err != nil {
			return err
		}

		remainingSize := objectSize - downloadedSize
		objectReader, objectSize, err = s3c.GetPartial(options.source.bucket,
			options.source.key, downloadedSize, remainingSize)
		if err != nil {
			return err
		}

		if !options.quiet {
			bar.Set(int(downloadedSize))
		}
	}

	writer := io.Writer(destination)
	if !options.quiet {
		// Start the bar now
		bar.Start()
		// create multi writer to feed data
		writer = io.MultiWriter(destination, bar)
	}

	_, err = io.CopyN(writer, objectReader, objectSize)
	if err != nil {
		return err
	}

	bar.Finish()
	info("Success!")
	return nil
}

// <S3Path> <S3Path> or <S3Path> <S3Bucket>
func thirdMode(s3c *s3.Client, options *cmdOptions) error {
	var objectReader io.ReadCloser
	var objectSize int64
	var err error

	// Send HEAD request to validate if file exists.
	objectSize, _, err = s3c.Stat(options.source.bucket, options.source.key)
	if err != nil {
		return err
	}

	if options.destination.key == "" {
		options.destination.key = options.source.key
	}

	// Check if the object already exists
	_, _, err = s3c.Stat(options.destination.bucket, options.destination.key)
	switch os.IsNotExist(err) {
	case true:
		objectReader, _, err = s3c.Get(options.source.bucket, options.source.key)
		if err != nil {
			return err
		}
		err = s3c.Put(options.destination.bucket, options.destination.key, objectSize, objectReader)
		if err != nil {
			return err
		}
	case false:
		return fmt.Errorf("Ranges not supported")
	}

	msg := fmt.Sprintf("s3://%s/%s uploaded -- to bucket:(s3://%s/%s)", options.source.bucket, options.source.key,
		options.destination.bucket, options.destination.key)
	info(msg)
	return nil
}

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

	switch getMode(cmdoptions) {
	case first:
		err := firstMode(s3c, cmdoptions)
		if err != nil {
			fatal(err.Error())
		}
	case second:
		err := secondMode(s3c, cmdoptions)
		if err != nil {
			fatal(err.Error())
		}
	case third:
		err := thirdMode(s3c, cmdoptions)
		if err != nil {
			fatal(err.Error())
		}
	default:
		fatal("Invalid request")
	}
}
