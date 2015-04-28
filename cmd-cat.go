/*
 * Mini Copy, (C) 2015 Minio, Inc.
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
	"errors"
	"io"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

const (
	standardOutput = "/dev/stdout"
)

func runCatCmd(ctx *cli.Context) {
	if !ctx.Args().Present() || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "cat", 1) // last argument is exit code
	}
	config, err := getMcConfig()
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("loading config file failed with following reason: [%s]\n", iodine.ToError(err))
	}

	// Convert arguments to URLs: expand alias, fix format...
	urls, err := getURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("reading URL [%s] failed with following reason: [%s]\n", e.url, e)
		default:
			log.Debug.Println(iodine.New(err, nil))
			console.Fatalf("reading URLs failed with following reason: [%s]\n", e)
		}
	}

	sourceURLs := urls
	sourceURLConfigMap, err := getHostConfigs(sourceURLs)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("reading host config for URL [%s] failed with following reason: [%s]\n", sourceURLs, iodine.ToError(err))
	}
	humanReadable, err := doCatCmd(mcClientMethods{}, sourceURLConfigMap, standardOutput, globalDebugFlag)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalln(humanReadable)
	}
}

func doCatCmd(methods clientMethods, sourceURLConfigMap map[string]*hostConfig, targetURL string, debug bool) (string, error) {
	for url, config := range sourceURLConfigMap {
		sourceClnt, err := methods.getNewClient(url, config, debug)
		if err != nil {
			return "Unable to create client: " + url, iodine.New(err, nil)
		}
		reader, size, expectedMd5, err := sourceClnt.Get()
		if err != nil {
			return "Unable to retrieve file: " + url, iodine.New(err, nil)
		}
		defer reader.Close()

		stdOutClnt, err := methods.getNewClient(targetURL, &hostConfig{}, debug)
		if err != nil {
			return "Unable to create client: " + url, iodine.New(err, nil)
		}
		stdOutWriter, err := stdOutClnt.Put(expectedMd5, size)
		if err != nil {
			return "Unable to retrieve file: " + url, iodine.New(err, nil)
		}
		defer stdOutWriter.Close()
		_, err = io.CopyN(stdOutWriter, reader, size)
		if err != nil {
			return "Reading data from source failed: " + url, iodine.New(errors.New("Copy data from source failed"), nil)
		}
	}
	return "", nil
}
