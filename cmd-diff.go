/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"path/filepath"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
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

	doDiffCmd(firstURL, secondURL)
}

// urlJoinPath - Join a path to existing URL
func urlJoinPath(urlStr string, path string) (newURLStr string, err error) {
	u, err := client.Parse(urlStr)
	if err != nil {
		return "", iodine.New(err, nil)
	}

	u.Path = filepath.Join(u.Path, path)
	newURLStr = u.String()
	return newURLStr, nil
}

// doDiffCmd - Execute the diff command
func doDiffCmd(firstURL string, secondURL string) {
	_, firstContent, err := URL2Stat(firstURL)
	if err != nil {
		console.Fatalf("Failed to stat ‘%s’. Reason: [%s].\n", firstURL, iodine.ToError(err))
	}

	_, secondContent, err := URL2Stat(secondURL)
	if err != nil {
		console.Fatalf("Failed to stat ‘%s’. Reason: [%s].\n", secondURL, iodine.ToError(err))
	}

	if firstContent.Type.IsRegular() {
		if !secondContent.Type.IsRegular() {
			console.Infof("‘%s’ and ‘%s’ differs in type.\n", firstURL, secondURL)
			return
		}
		doDiffCmdObjects(firstURL, secondURL)
		return
	}

	if firstContent.Type.IsDir() {
		if !secondContent.Type.IsDir() {
			console.Infof("‘%s’ and ‘%s’ differs in type.\n", firstURL, secondURL)
			return
		}
		doDiffCmdDirs(firstURL, secondURL)
		return
	}

	console.Fatalf("‘%s’ is of unknown type.\n", firstURL)
}

// doDiffCmdObjects - Diff two object URLs
func doDiffCmdObjects(firstURL string, secondURL string) {
	if firstURL == secondURL { // kind of lame :p
		return
	}
	_, firstContent, err := URL2Stat(firstURL)
	if err != nil {
		console.Fatalf("Failed to stat ‘%s’. Reason: [%s].\n", firstURL, iodine.ToError(err))
	}

	_, secondContent, err := URL2Stat(secondURL)
	if err != nil {
		console.Fatalf("Failed to stat ‘%s’. Reason: [%s].\n", secondURL, iodine.ToError(err))
	}

	if firstContent.Type.IsRegular() {
		if !secondContent.Type.IsRegular() {
			console.Infof("‘%s’ and ‘%s’ differs in type.\n", firstURL, secondURL)
			return
		}
	} else {
		console.Fatalf("‘%s’ is not an object. Please report this bug with ‘--debug’ option\n.", firstURL)
	}

	if firstContent.Size != secondContent.Size {
		console.Infof("‘%s’ and ‘%s’ differs in size.\n", firstURL, secondURL)
	}
}

// doDiffCmdDirs - Diff two Dir URLs
func doDiffCmdDirs(firstURL string, secondURL string) {
	firstClnt, firstContent, err := URL2Stat(firstURL)
	if err != nil {
		console.Fatalf("Failed to stat ‘%s’. Reason: [%s].\n", firstURL, iodine.ToError(err))
	}

	_, secondContent, err := URL2Stat(secondURL)
	if err != nil {
		console.Fatalf("Failed to stat ‘%s’. Reason: [%s].\n", secondURL, iodine.ToError(err))
	}

	if firstContent.Type.IsDir() {
		if !secondContent.Type.IsDir() {
			console.Infof("‘%s’ and ‘%s’ differs in type.\n", firstURL, secondURL)
			return
		}
	} else {
		console.Fatalf("‘%s’ is not a directory. Please report this bug with ‘--debug’ option\n.", firstURL)
	}

	for contentCh := range firstClnt.List() {
		if contentCh.Err != nil {
			console.Fatalf("Failed to list ‘%s’. Reason: [%s].\n", firstURL, iodine.ToError(contentCh.Err))
		}

		newFirstURL, err := urlJoinPath(firstURL, contentCh.Content.Name)
		if err != nil {
			console.Fatalf("Unable to construct new URL from ‘%s’ using ‘%s’. Reason: [%s].\n", firstURL, contentCh.Content.Name, iodine.ToError(err))
		}

		newSecondURL, err := urlJoinPath(secondURL, contentCh.Content.Name)
		if err != nil {
			console.Fatalf("Unable to construct new URL from ‘%s’ using ‘%s’. Reason: [%s].\n", secondURL, contentCh.Content.Name, iodine.ToError(err))
		}

		_, newFirstContent, err := URL2Stat(newFirstURL)
		if err != nil {
			console.Fatalf("Failed to stat ‘%s’. Reason: [%s].\n", newFirstURL, iodine.ToError(err))
		}

		_, newSecondContent, err := URL2Stat(newSecondURL)
		if err != nil {
			console.Infof("‘%s’ only in ‘%s’.\n", filepath.Base(newFirstContent.Name), firstURL)
			continue
		}

		if newFirstContent.Type.IsDir() {
			if !newSecondContent.Type.IsDir() {
				console.Infof("‘%s’ and ‘%s’ differs in type.\n", newFirstURL, newSecondURL)
				continue
			}
		} else if newFirstContent.Type.IsRegular() {
			if !newSecondContent.Type.IsRegular() {
				console.Infof("‘%s’ and ‘%s’ differs in type.\n", newFirstURL, newSecondURL)
				continue
			}
			doDiffCmdObjects(newFirstURL, newSecondURL)
		}
	} // End of for-loop
}
