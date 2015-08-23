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
	"github.com/minio/minio/pkg/probe"
)

// Help message.
var diffCmd = cli.Command{
	Name:        "diff",
	Usage:       "Compute differences between two files or folders",
	Description: "NOTE: This command *DOES NOT* check for content similarity, which means objects with same size, but different content will not be spotted",
	Action:      mainDiff,
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

   2. Compare two different folders on a local filesystem.
      $ mc {{.Name}} ~/Photos /Media/Backup/Photos

`,
}

// mainDiff - is a handler for mc diff command
func mainDiff(ctx *cli.Context) {
	if len(ctx.Args()) != 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "diff", 1) // last argument is exit code
	}

	config := mustGetMcConfig()
	firstArg := ctx.Args().First()
	secondArg := ctx.Args()[1]

	var err *probe.Error
	firstURL, err := getCanonicalizedURL(firstArg, config.Aliases)
	fatalIf(err.Trace(firstArg), "Unable to parse first argument ‘"+firstArg+"’.")

	secondURL, err := getCanonicalizedURL(secondArg, config.Aliases)
	fatalIf(err.Trace(secondArg), "Unable to parse second argument ‘"+secondArg+"’.")

	if isURLRecursive(secondURL) {
		fatalIf(probe.NewError(errInvalidArgument), "Second argument ‘"+secondURL+"’ cannot be recursive.")
	}
	newFirstURL := stripRecursiveURL(firstURL)
	for diff := range doDiffCmd(newFirstURL, secondURL, isURLRecursive(firstURL)) {
		fatalIf(diff.err.Trace(newFirstURL, secondURL), "Failed to diff ‘"+firstURL+"’ and ‘"+secondURL+"’.")
		fmt.Println(diff.message)
	}
}

// doDiffCmd - Execute the diff command
func doDiffCmd(firstURL, secondURL string, recursive bool) <-chan diff {
	ch := make(chan diff, 10000)
	go doDiffInRoutine(firstURL, secondURL, recursive, ch)
	return ch
}
