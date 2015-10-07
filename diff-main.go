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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
)

// Compute differences between two files or folders.
var diffCmd = cli.Command{
	Name:        "diff",
	Usage:       "Compute differences between two files or folders.",
	Description: "NOTE: This command *DOES NOT* check for content similarity, which means objects with same size, but different content will not be spotted.",
	Action:      mainDiff,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} FIRST SECOND

EXAMPLES:
   1. Compare foo.ogg on a local filesystem with bar.ogg on Amazon AWS cloud storage.
      $ mc {{.Name}} foo.ogg https://s3.amazonaws.com/jukebox/bar.ogg

   2. Compare two different folders on a local filesystem.
      $ mc {{.Name}} ~/Photos /Media/Backup/Photos
`,
}

func checkDiffSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 || ctx.Args().First() == "help" {
		cli.ShowCommandHelpAndExit(ctx, "diff", 1) // last argument is exit code
	}
	for _, arg := range ctx.Args() {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
		}
	}
	if isURLRecursive(ctx.Args().Last()) {
		fatalIf(errInvalidArgument().Trace(), "Second argument ‘"+ctx.Args().Last()+"’ cannot be recursive.")
	}
}

// mainDiff - is a handler for mc diff command
func mainDiff(ctx *cli.Context) {
	checkDiffSyntax(ctx)

	console.SetCustomPalette(map[string]*color.Color{
		"DiffMessage":     color.New(color.FgGreen, color.Bold),
		"DiffOnlyInFirst": color.New(color.FgRed, color.Bold),
		"DiffType":        color.New(color.FgYellow, color.Bold),
		"DiffSize":        color.New(color.FgMagenta, color.Bold),
	})

	config := mustGetMcConfig()
	firstArg := ctx.Args().First()
	secondArg := ctx.Args().Last()

	firstURL := getAliasURL(firstArg, config.Aliases)
	secondURL := getAliasURL(secondArg, config.Aliases)

	newFirstURL := stripRecursiveURL(firstURL)
	for diff := range doDiff(newFirstURL, secondURL, isURLRecursive(firstURL)) {
		fatalIf(diff.Error.Trace(newFirstURL, secondURL), "Failed to diff ‘"+firstURL+"’ and ‘"+secondURL+"’.")

		Prints("%s\n", diff)
	}
}

// doDiff - Execute the diff command
func doDiff(firstURL, secondURL string, recursive bool) <-chan DiffMessage {
	ch := make(chan DiffMessage, 10000)
	go doDiffInRoutine(firstURL, secondURL, recursive, ch)
	return ch
}
