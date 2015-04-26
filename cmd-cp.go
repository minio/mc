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
	"strings"

	"github.com/minio-io/cli"
	"github.com/minio-io/mc/pkg/console"
	"github.com/minio-io/minio/pkg/iodine"
	"github.com/minio-io/minio/pkg/utils/log"
)

func runCopyCmdMultipleSources(methods clientMethods, urls []string) {
	sourceURLs := urls[:len(urls)-1] // All args are source except the last one
	targetURL := urls[len(urls)-1]   // Last one is target
	targetConfig, err := getHostConfig(targetURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read host configuration for the following targets %s from config file [%s]. Reason: [%s].\n",
			targetURL, mustGetMcConfigPath(), iodine.ToError(err))
	}
	sourceURLConfigMap, err := getHostConfigs(sourceURLs)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read host configuration for source [%s] from config file [%s]. Reason: [%s].\n",
			sourceURLs, mustGetMcConfigPath(), iodine.ToError(err))
	}
	err = doCopyMultipleSources(methods, sourceURLConfigMap, targetURL, targetConfig)
	if err != nil {
		log.Debug.Println(err)
		console.Fatalf("Failed to copy from source [%s] to target %s. Reason: [%s].\n", sourceURLs, targetURL, iodine.ToError(err))
	}

}

func runCopyCmdSingleSource(methods clientMethods, urls []string) {
	sourceURL := urls[0]
	targetURL := urls[1]
	targetConfig, err := getHostConfig(targetURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read host configuration for the following targets %s from config file [%s]. Reason: [%s].\n",
			targetURL, mustGetMcConfigPath(), iodine.ToError(err))
	}
	recursive := isURLRecursive(sourceURL)
	// if recursive strip off the "..."
	if recursive {
		sourceURL = strings.TrimSuffix(sourceURL, recursiveSeparator)
	}
	sourceConfig, err := getHostConfig(sourceURL)
	if err != nil {
		log.Debug.Println(iodine.New(err, nil))
		console.Fatalf("Unable to read host configuration for the following targets %s from config file [%s]. Reason: [%s].\n",
			sourceURL, mustGetMcConfigPath(), iodine.ToError(err))
	}
	if recursive {
		err = doCopySingleSourceRecursive(methods, sourceURL, targetURL, sourceConfig, targetConfig)
		if err != nil {
			log.Debug.Println(err)
			console.Fatalf("Failed to copy from source [%s] to target %s. Reason: [%s].\n", sourceURL, targetURL, iodine.ToError(err))
		}
		return
	}
	err = doCopySingleSource(methods, sourceURL, targetURL, sourceConfig, targetConfig)
	if err != nil {
		log.Debug.Println(err)
		console.Fatalf("Failed to copy from source [%s] to target %s. Reason: [%s].\n", sourceURL, targetURL, iodine.ToError(err))
	}

}

func runCopyCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
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

	methods := mcClientMethods{}
	switch len(urls) {
	case 2:
		runCopyCmdSingleSource(methods, urls)
	default:
		runCopyCmdMultipleSources(methods, urls)
	}
}
