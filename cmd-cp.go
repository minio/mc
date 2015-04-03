/*
 * Minimalist Object Storage, (C) 2014,2015 Minio, Inc.
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
	"io/ioutil"
	"os"

	"github.com/cheggaaa/pb"
	"github.com/minio-io/cli"
)

// doCopyCmd copies objects into and from a bucket or between buckets
func doCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelp(ctx, "cp")
		os.Exit(1)
	}

	// var recursiveMode = ctx.Bool("recursive")
	sourceURL, err := parseURL(ctx.Args().First())
	if err != nil {
		fatal(err.Error())
		return
	}

	targetURL, err := parseURL(ctx.Args()[1])
	if err != nil {
		fatal(err.Error())
		return
	}

	sourceBucket, sourceObject, err := url2Object(sourceURL)
	if err != nil {
		fatal(err.Error())
		return
	}

	targetBucket, targetObject, err := url2Object(targetURL)
	if err != nil {
		fatal(err.Error())
		return
	}

	sourceClnt, err := getNewClient(globalDebugFlag, sourceURL)
	if err != nil {
		fatal(err.Error())
		return
	}

	targetClnt, err := getNewClient(globalDebugFlag, targetURL)
	if err != nil {
		fatal(err.Error())
		return
	}

	sourceReader, sourceSize, err := sourceClnt.Get(sourceBucket, sourceObject)
	if err != nil {
		fatal(err.Error())
		return
	}

	var bar *pb.ProgressBar
	if !globalQuietFlag {
		bar = startBar(sourceSize)
		bar.Start()
		sourceReader = ioutil.NopCloser(io.TeeReader(sourceReader, bar))
	}

	if err = targetClnt.Put(targetBucket, targetObject, sourceSize, sourceReader); err != nil {
		fatal(err.Error())
		return
	}

	if !globalQuietFlag {
		bar.Finish()
		info("Success!")
	}

	return
}
