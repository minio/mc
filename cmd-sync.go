/*
 * Minio Client, (C) 2015 Minio, Inc.
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
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

func runSyncCmd(ctx *cli.Context) {
	if len(ctx.Args()) < 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "sync", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	config, err := getMcConfig()
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalf("Unable to read config file [%s]. Reason: [%s].\n", mustGetMcConfigPath(), iodine.ToError(err))
	}

	// Convert arguments to URLs: expand alias, fix format...
	_, err = getExpandedURLs(ctx.Args(), config.Aliases)
	if err != nil {
		switch e := iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("Unknown type of URL(s).\n")
		default:
			console.Debugln(iodine.New(err, nil))
			console.Fatalf("Unable to parse arguments. Reason: [%s].\n", e)
		}
	}
	//	runCopyCmdSingleSourceMultipleTargets(urls)
}

/*
func runCopyCmdSingleSourceMultipleTargets(urls []string) {
	sourceURL := urls[0]   // first arg is source
	targetURLs := urls[1:] // all other are targets

	recursive := isURLRecursive(sourceURL)
	// if recursive strip off the "..."
	if recursive {
		sourceURL = stripRecursiveURL(sourceURL)
	}
	sourceConfig, err := getHostConfig(sourceURL)
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalf("Unable to read host configuration for the source %s from config file [%s]. Reason: [%s].\n",
			sourceURL, mustGetMcConfigPath(), iodine.ToError(err))
	}
	targetURLConfigMap, err := getHostConfigs(targetURLs)
	if err != nil {
		console.Debugln(iodine.New(err, nil))
		console.Fatalf("Unable to read host configuration for the following targets [%s] from config file [%s]. Reason: [%s].\n",
			targetURLs, mustGetMcConfigPath(), iodine.ToError(err))
	}

	for targetURL, targetConfig := range targetURLConfigMap {
		err = doCopyRecursive(targetURL, targetConfig, sourceURL, sourceConfig)
		if err != nil {
			console.Debugln(err)
			console.Fatalf("Failed to copy from source [%s] to target %s. Reason: [%s].\n",
				sourceURL, targetURL, iodine.ToError(err))
		}
	}
}

*/
