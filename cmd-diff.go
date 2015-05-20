/*
 * Minio Client (C) 2015 Minio, Inc.
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

// runDiffCmd - is a handler for mc diff command
func runDiffCmd(ctx *cli.Context) {
	if len(ctx.Args()) != 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "diff", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatalln("\"mc\" is not configured.  Please run \"mc config generate\".")
	}
	config, err := getMcConfig()
	if err != nil {
		console.Fatalf("Unable to read config file [%s]. Reason: [%s].\n", mustGetMcConfigPath(), iodine.ToError(err))
	}

	firstURL := ctx.Args().First()
	secondURL := ctx.Args()[1]

	firstURL, err = getExpandedURL(firstURL, config.Aliases)
	if err != nil {
		switch iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Fatalf("Unknown type of URL [%s].\n", firstURL)
		default:
			console.Fatalf("Unable to parse argument [%s]. Reason: [%s].\n", firstURL, iodine.ToError(err))
		}
	}

	_, err = getHostConfig(firstURL)
	if err != nil {
		console.Fatalf("Unable to read host configuration for [%s] from config file [%s]. Reason: [%s].\n",
			firstURL, mustGetMcConfigPath(), iodine.ToError(err))
	}

	_, err = getHostConfig(secondURL)
	if err != nil {
		console.Fatalf("Unable to read host configuration for [%s] from config file [%s]. Reason: [%s].\n",
			secondURL, mustGetMcConfigPath(), iodine.ToError(err))
	}

	secondURL, err = getExpandedURL(secondURL, config.Aliases)
	if err != nil {
		switch iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Fatalf("Unknown type of URL [%s].\n", secondURL)
		default:
			console.Fatalf("Unable to parse argument [%s]. Reason: [%s].\n", secondURL, iodine.ToError(err))
		}
	}

	for diff := range doDiffCmd(firstURL, secondURL) {
		if diff.err != nil {
			console.Fatalf(diff.message)
		}
		console.Infof(diff.message)
	}
}

func doDiffInRoutine(firstURL, secondURL string, ch chan diff) {
	_, firstContent, err := url2Stat(firstURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat " + firstURL + ". Reason: [" + iodine.ToError(err).Error() + "]\n",
			err:     iodine.New(err, nil),
		}
		return
	}
	_, secondContent, err := url2Stat(secondURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat " + secondURL + ". Reason: [" + iodine.ToError(err).Error() + "]\n",
			err:     iodine.New(err, nil),
		}
		return
	}
	if firstContent.Type.IsRegular() {
		if !secondContent.Type.IsRegular() {
			ch <- diff{
				message: firstURL + " and " + secondURL + " differs in type.\n",
				err:     nil,
			}
		}
		doDiffObjects(firstURL, secondURL, ch)
	}
	if firstContent.Type.IsDir() {
		if !secondContent.Type.IsDir() {
			ch <- diff{
				message: firstURL + " and " + secondURL + " differs in type.\n",
				err:     nil,
			}
		}
		doDiffDirs(firstURL, secondURL, ch)
	}
	close(ch)
}

// doDiffCmd - Execute the diff command
func doDiffCmd(firstURL, secondURL string) <-chan diff {
	ch := make(chan diff)
	go doDiffInRoutine(firstURL, secondURL, ch)
	return ch
}
