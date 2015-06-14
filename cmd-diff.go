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
	"fmt"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// Help message.
var diffCmd = cli.Command{
	Name:        "diff",
	Usage:       "Compute differences between two files or folders",
	Description: "NOTE: This command *DOES NOT* check for content similarity, which means objects with same size, but different content will not be spotted",
	Action:      runDiffCmd,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} FIRST SECOND {{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if .Flags}}

FLAGS:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}

EXAMPLES:
   1. Compare foo.ogg on a local filesystem with bar.ogg on Amazon AWS cloud storage.
      $ mc {{.Name}} foo.ogg  https://s3.amazonaws.com/jukebox/bar.ogg

   2. Compare two different directories on a local filesystem.
      $ mc {{.Name}} ~/Photos /Media/Backup/Photos

`,
}

// runDiffCmd - is a handler for mc diff command
func runDiffCmd(ctx *cli.Context) {
	if len(ctx.Args()) != 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "diff", 1) // last argument is exit code
	}
	if !isMcConfigExist() {
		console.Fatals(ErrorMessage{
			Message: "Please run \"mc config generate\"",
			Error:   iodine.New(errNotConfigured{}, nil),
		})
	}
	config, err := getMcConfig()
	if err != nil {
		console.Fatals(ErrorMessage{
			Message: fmt.Sprintf("Unable to read config file ‘%s’", mustGetMcConfigPath()),
			Error:   err,
		})
	}

	firstURL := ctx.Args().First()
	secondURL := ctx.Args()[1]

	firstURL, err = getExpandedURL(firstURL, config.Aliases)
	if err != nil {
		switch iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Fatals(ErrorMessage{
				Message: fmt.Sprintf("Unknown type of URL ‘%s’", firstURL),
				Error:   err,
			})
		default:
			console.Fatals(ErrorMessage{
				Message: fmt.Sprintf("Unable to parse argument ‘%s’", firstURL),
				Error:   err,
			})
		}
	}
	secondURL, err = getExpandedURL(secondURL, config.Aliases)
	if err != nil {
		switch iodine.ToError(err).(type) {
		case errUnsupportedScheme:
			console.Fatals(ErrorMessage{
				Message: fmt.Sprintf("Unknown type of URL ‘%s’", secondURL),
				Error:   err,
			})
		default:
			console.Fatals(ErrorMessage{
				Message: fmt.Sprintf("Unable to parse argument ‘%s’", secondURL),
				Error:   err,
			})
		}
	}
	if isURLRecursive(secondURL) {
		console.Fatals(ErrorMessage{
			Message: "Second URL cannot be recursive, diff command is unidirectional",
			Error:   errInvalidArgument{},
		})
	}
	newFirstURL := stripRecursiveURL(firstURL)
	for diff := range doDiffCmd(newFirstURL, secondURL, isURLRecursive(firstURL)) {
		if diff.err != nil {
			console.Fatals(ErrorMessage{
				Message: diff.message,
				Error:   diff.err,
			})
		}
		console.Infoln(diff.message)
	}
}

func doDiffInRoutine(firstURL, secondURL string, recursive bool, ch chan diff) {
	defer close(ch)
	_, firstContent, err := url2Stat(firstURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + firstURL + "’",
			err:     iodine.New(err, nil),
		}
		return
	}
	_, secondContent, err := url2Stat(secondURL)
	if err != nil {
		ch <- diff{
			message: "Failed to stat ‘" + secondURL + "’",
			err:     iodine.New(err, nil),
		}
		return
	}
	if firstContent.Type.IsRegular() {
		switch {
		case secondContent.Type.IsDir():
			newSecondURL, err := urlJoinPath(secondURL, firstURL)
			if err != nil {
				ch <- diff{
					message: "Unable to construct new URL from ‘" + secondURL + "’ using ‘" + firstURL,
					err:     iodine.New(err, nil),
				}
				return
			}
			doDiffObjects(firstURL, newSecondURL, ch)
		case !secondContent.Type.IsRegular():
			ch <- diff{
				message: "‘" + firstURL + "’ and " + "‘" + secondURL + "’ differs in type.",
				err:     nil,
			}
			return
		case secondContent.Type.IsRegular():
			doDiffObjects(firstURL, secondURL, ch)
		}
	}
	if firstContent.Type.IsDir() {
		switch {
		case !secondContent.Type.IsDir():
			ch <- diff{
				message: "‘" + firstURL + "’ and " + "‘" + secondURL + "’ differs in type.",
				err:     nil,
			}
			return
		default:
			doDiffDirs(firstURL, secondURL, recursive, ch)
		}
	}
}

// doDiffCmd - Execute the diff command
func doDiffCmd(firstURL, secondURL string, recursive bool) <-chan diff {
	ch := make(chan diff)
	go doDiffInRoutine(firstURL, secondURL, recursive, ch)
	return ch
}
