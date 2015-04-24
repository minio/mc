/*
 * Mini Copy (C) 2014, 2015 Minio, Inc.
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
	"strings"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

func runCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 {
		cli.ShowCommandHelpAndExit(ctx, "cp", 1) // last argument is exit code
	}

	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read config file [%s]. Reason: [%s].\n", mustGetMcConfigPath(), iodine.ToError(err))
	}

	// Convert arguments to URLs: expand alias, fix format...
	urls, err := getURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("Unknown type of URL(s).\n")
		default:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("Unable to parse arguments. Reason: [%s].\n", e)
		}
	}

	sourceURL := urls[0] // First arg is source
	recursive := isURLRecursive(sourceURL)
	// if recursive strip off the "..."
	if recursive {
		sourceURL = strings.TrimSuffix(sourceURL, recursiveSeparator)
	}
	targetURLs := urls[1:] // 1 or more targets

	sourceURLConfigMap := make(map[string]*hostConfig)
	sourceConfig, err := getHostConfig(sourceURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read host configuration for source [%s] from config file [%s]. Reason: [%s].\n",
			sourceURL, mustGetMcConfigPath(), iodine.ToError(err))
	}
	sourceURLConfigMap[sourceURL] = sourceConfig

	targetURLConfigMap, err := getHostConfigs(targetURLs)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read host configuration for the following targets %s from config file [%s]. Reason: [%s].\n",
			targetURLs, mustGetMcConfigPath(), iodine.ToError(err))
	}

	// perform recursive
	if recursive {
		err := doCopyCmdRecursive(mcClientManager{}, sourceURLConfigMap, targetURLConfigMap)
		err = iodine.New(err, nil)
		if err != nil {
			log.Debug.Println(err)
			console.Fatalf("Failed to copy recursively. Reason: [%s].\n", iodine.ToError(err))
		}
		return
	}
	err = doCopyCmd(mcClientManager{}, sourceURLConfigMap, targetURLConfigMap)
	err = iodine.New(err, nil)
	if err != nil {
		log.Debug.Println(err)
		console.Fatalf("Failed to copy from source [%s] to target %s. Reason: [%s].\n", sourceURL, targetURLs, iodine.ToError(err))
	}
}

// doCopyCmd copies objects into and from a bucket or between buckets
func doCopyCmd(manager clientManager, sourceURLConfigMap map[string]*hostConfig, targetURLConfigMap map[string]*hostConfig) error {
	for sourceURL, sourceConfig := range sourceURLConfigMap {
		reader, length, hexMd5, err := manager.getSourceReader(sourceURL, sourceConfig)
		if err != nil {
			return iodine.New(err, map[string]string{"Source": sourceURL})
		}
		defer reader.Close()

		writeClosers, err := getTargetWriters(manager, targetURLConfigMap, hexMd5, length)
		if err != nil {
			return iodine.New(err, nil)
		}

		var writers []io.Writer
		for _, writer := range writeClosers {
			writers = append(writers, writer)
		}

		// set up progress bar
		var bar *pb.ProgressBar
		if !globalQuietFlag {
			bar = startBar(length)
			bar.Start()
			writers = append(writers, bar)
		}

		// write progress bar
		multiWriter := io.MultiWriter(writers...)

		// copy data to writers
		_, copyErr := io.CopyN(multiWriter, reader, length)
		// close writers
		for _, writer := range writeClosers {
			// it is the target's responsibility to notice if a close is premature.
			// on fs, we handle in fs client
			// over the wire, the server is responsible
			err = writer.Close()
			// don't return an error here, this error may be caused by a previous error.
			// we check for this again after copyErr
		}
		// write copy errors if present
		if copyErr != nil {
			return iodine.New(copyErr, map[string]string{"Source": sourceURL})
		}
		// write close errors if present after checking copyErr
		if err != nil {
			return iodine.New(err, map[string]string{"Source": sourceURL})
		}
		if !globalQuietFlag {
			bar.Finish()
			// console.Infoln("Success!")
		}
	}
	return nil
}
