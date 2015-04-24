/*
 * Mini Copy, (C) 2014, 2015 Minio, Inc.
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
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/crypto/md5"
	"github.com/minio-io/minio/pkg/utils/log"
)

func runCatCmd(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "cat", 1) // last argument is exit code
	}

	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: loading config file failed with following reason: [%s]\n", iodine.ToError(err))
	}

	// Convert arguments to URLs: expand alias, fix format...
	urls, err := getURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: reading URL [%s] failed with following reason: [%s]\n", e.url, e)
		default:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("mc: reading URLs failed with following reason: [%s]\n", e)
		}
	}

	sourceURL := urls[0] // First arg is source
	recursive := isURLRecursive(sourceURL)
	// if recursive strip off the "..."
	if recursive {
		sourceURL = strings.TrimSuffix(sourceURL, recursiveSeparator)
	}

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig, err := getHostConfig(sourceURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("mc: reading host config for URL [%s] failed with following reason: [%s]\n", sourceURL, iodine.ToError(err))
	}
	sourceURLConfigMap[sourceURL] = sourceConfig

	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		os.Exit(1)
	}
	humanReadable, err := doCatCmd(mcClientManager{}, os.Stdout, sourceURLConfigMap, globalDebugFlag)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln(humanReadable)
	}
}

func doCatCmd(manager clientManager, writer io.Writer, sourceURLConfigMap map[string]*hostConfig, debug bool) (string, error) {
	for url, config := range sourceURLConfigMap {
		clnt, err := manager.getNewClient(url, config, debug)
		if err != nil {
			return "Unable to create client: " + url, iodine.New(err, nil)
		}
		reader, size, expectedMd5, err := clnt.Get()
		if err != nil {
			return "Unable to retrieve file: " + url, iodine.New(err, nil)
		}
		defer reader.Close()
		var teeReader io.Reader
		var actualMd5 []byte
		if client.GetType(url) != client.Filesystem {
			wg := &sync.WaitGroup{}
			md5Reader, md5Writer := io.Pipe()
			wg.Add(1)
			go func() {
				actualMd5, _ = md5.Sum(md5Reader)
				// drop error, we'll catch later on if it fails
				wg.Done()
			}()
			teeReader = io.TeeReader(reader, md5Writer)
			_, err = io.CopyN(writer, teeReader, size)
			md5Writer.Close()
			wg.Wait()
			if err != nil {
				return "Copying data from source failed: " + url, iodine.New(errors.New("Copy data from source failed"), nil)
			}
			actualMd5String := hex.EncodeToString(actualMd5)
			if expectedMd5 != actualMd5String {
				return "Copying data from source was corrupted in transit: " + url,
					iodine.New(errors.New("Data copied from source was corrupted in transit"), nil)
			}
			return "", nil
		}
		teeReader = reader
		_, err = io.CopyN(writer, teeReader, size)
		if err != nil {
			return "Copying data from source failed: " + url, iodine.New(errors.New("Copy data from source failed"), nil)
		}
	}
	return "", nil
}
