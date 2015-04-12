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

type message struct {
	err error
}

//func multiTargetGoroutine(targetURL string, md5Hex string, sourceSize int64, targetReader io.Reader, ch chan message) {
//	errParams := map[string]string{
//		"url": targetURL,
//	}
//	targetBucket, targetObject, err := url2Object(targetURL)
//	if err != nil {
//		ch <- message{err: iodine.New(err, errParams)}
//	}
//	targetClnt, err := getNewClient(globalDebugFlag, targetURL)
//	if err != nil {
//		ch <- message{err: iodine.New(err, errParams)}
//	}
//	if err := targetClnt.Put(targetBucket, targetObject, md5Hex, sourceSize, targetReader); err != nil {
//		ch <- message{err: iodine.New(err, errParams)}
//	}
//	ch <- message{err: nil}
//	close(ch)
//}

//func doPutMultiTarget(targetURLs []string, md5Hex string, sourceSize int64, targetReaders []io.Reader) <-chan message {
//	ch := make(chan message)
//	for i := 0; i < len(targetURLs); i++ {
//		go multiTargetGoroutine(targetURLs[i], md5Hex, sourceSize, targetReaders[i], ch)
//	}
//	return ch
//}

func getClientReader(sourceURL string) (reader io.ReadCloser, length int64, md5hex string, err error) {
	errParams := map[string]string{"sourceURL": sourceURL}
	// Parse URL to bucket and object names
	sourceBucket, sourceObject, err := url2Object(sourceURL)
	if err != nil {
		return nil, 0, "", iodine.New(err, errParams)
	}

	sourceClnt, err := getNewClient(globalDebugFlag, sourceURL)
	if err != nil {
		return nil, 0, "", iodine.New(err, errParams)
	}

	// Get a reader for the source object
	return sourceClnt.Get(sourceBucket, sourceObject)
}

// doCopyCmd copies objects into and from a bucket or between buckets
func doCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
	}
	// Convert arguments to URLs: expand alias, fix format...
	urlList, err := parseURLs(ctx)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln(err)
	}
	sourceURL := urlList[0]   // First arg is source
	targetURLs := urlList[1:] // 1 or more targets

	reader, length, hexMd5, err := getClientReader(sourceURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to read source")
	}

	writeClosers, err := getWriters(targetURLs, hexMd5, length)
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
		}
	}

	//	time.Sleep(5 * time.Second)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln("Unable to write to target")
	}
}

func getWriter(targetURL string, md5Hex string, length int64) (io.WriteCloser, error) {
	client, err := getNewClient(false, targetURL)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"failedURL": targetURL})
	}
	targetBucket, targetObject, err := url2Object(targetURL)
	if err != nil {
		return nil, iodine.New(err, map[string]string{"failedURL": targetURL})
	}
	return client.Put(targetBucket, targetObject, md5Hex, length)
}

func getWriters(urls []string, md5Hex string, length int64) ([]io.WriteCloser, error) {
	var targetWriters []io.WriteCloser
	for _, targetURL := range urls {
		writer, err := getWriter(targetURL, md5Hex, length)
		if err != nil {
			// close all writers
			for _, targetWriter := range targetWriters {
				targetWriter.Close()
			}
			return nil, iodine.New(err, map[string]string{"failedURL": targetURL})
		}
		targetWriters = append(targetWriters, writer)
	}
	return targetWriters, nil
}
