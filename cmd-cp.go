/*
 * Modern Copy, (C) 2014,2015 Minio, Inc.
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

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

func getSourceReader(sourceURLParser *urlParser) (reader io.ReadCloser, length int64, md5hex string, err error) {
	sourceClnt, err := getNewClient(sourceURLParser, globalDebugFlag)
	if err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"sourceURL": sourceURLParser.String()})
	}
	// Get a reader for the source object
	sourceBucket := sourceURLParser.bucketName
	// check if the bucket is valid
	if err := sourceClnt.StatBucket(sourceBucket); err != nil {
		return nil, 0, "", iodine.New(err, map[string]string{"sourceURL": sourceURLParser.String()})
	}
	sourceObject := sourceURLParser.objectName
	return sourceClnt.Get(sourceBucket, sourceObject)
}

func getTargetWriter(targetURLParser *urlParser, md5Hex string, length int64) (io.WriteCloser, error) {
	targetClnt, err := getNewClient(targetURLParser, globalDebugFlag)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"failedURL": targetURLParser.String()})
	}
	targetBucket := targetURLParser.bucketName
	// check if bucket is valid
	if err := targetClnt.StatBucket(targetBucket); err != nil {
		return nil, iodine.New(err, map[string]string{"failedURL": targetURLParser.String()})
	}
	targetObject := targetURLParser.objectName
	return targetClnt.Put(targetBucket, targetObject, md5Hex, length)
}

func getTargetWriters(targetURLParsers []*urlParser, md5Hex string, length int64) ([]io.WriteCloser, error) {
	var targetWriters []io.WriteCloser
	for _, targetURLParser := range targetURLParsers {
		writer, err := getTargetWriter(targetURLParser, md5Hex, length)
		if err != nil {
			// close all writers
			for _, targetWriter := range targetWriters {
				targetWriter.Close()
			}
			return nil, iodine.New(err, map[string]string{"failedURL": targetURLParser.String()})
		}
		targetWriters = append(targetWriters, writer)
	}
	return targetWriters, nil
}

// doCopyCmd copies objects into and from a bucket or between buckets
func doCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
	}
	// Convert arguments to URLs: expand alias, fix format...
	urlParsers, err := parseURLs(ctx)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to parse URL")
	}
	sourceURLParser := urlParsers[0]   // First arg is source
	targetURLsParser := urlParsers[1:] // 1 or more targets

	reader, length, hexMd5, err := getSourceReader(sourceURLParser)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to read source")
	}

	writeClosers, err := getTargetWriters(targetURLsParser, hexMd5, length)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to open targets for writing")
	}
	// set up progress bar
	var writers []io.Writer
	for _, writer := range writeClosers {
		writers = append(writers, writer)
	}
	bar := startBar(length)
	bar.Start()
	writers = append(writers, bar)

	// write progress bar
	multiWriter := io.MultiWriter(writers...)

	// copy data to writers
	_, err = io.CopyN(multiWriter, reader, length)

	// close writers
	for _, writer := range writeClosers {
		err := writer.Close()
		if err != nil {
			log.Debug.Println(iodine.New(err, nil))
			console.Errorln("Unable to close writer, object may not of written properly.")
		}
	}
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to write to target")
	}
}
