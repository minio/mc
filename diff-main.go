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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/client"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio-xl/pkg/probe"
)

// Compute differences between two files or folders.
var diffCmd = cli.Command{
	Name:        "diff",
	Usage:       "Compute differences between two folders.",
	Description: "NOTE: This command *DOES NOT* check for content similarity, which means objects with same size, but different content will not be spotted.",
	Action:      mainDiff,
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} FIRST SECOND

EXAMPLES:
   1. Compare two different folders on a local filesystem.
      $ mc {{.Name}} ~/Photos /Media/Backup/Photos

   2. Compare a local folder with a folder on cloud storage.
      $ mc {{.Name}} ~/Photos https://s3.amazonaws.com/MyBucket/Photos
`,
}

func checkDiffSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "diff", 1) // last argument is exit code
	}
	for _, arg := range ctx.Args() {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(), "Unable to validate empty argument.")
		}
	}
}

// diffMessage json container for diff messages
type diffMessage struct {
	FirstURL  string       `json:"first"`
	SecondURL string       `json:"second"`
	Diff      string       `json:"diff"`
	Error     *probe.Error `json:"error,omitempty"`
}

// String colorized diff message
func (d diffMessage) String() string {
	msg := ""
	switch d.Diff {
	case "only-in-first":
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffOnlyInFirst", " - only in first.")
	case "type":
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffType", " - differ in type.")
	case "size":
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffSize", " - differ in size.")
	default:
		fatalIf(errDummy().Trace(),
			"Unhandled difference between ‘"+d.FirstURL+"’ and ‘"+d.SecondURL+"’.")
	}
	return msg

}

// JSON jsonified diff message
func (d diffMessage) JSON() string {
	diffJSONBytes, err := json.Marshal(d)
	fatalIf(probe.NewError(err),
		"Unable to marshal diff message ‘"+d.FirstURL+"’, ‘"+d.SecondURL+"’ and ‘"+d.Diff+"’.")
	return string(diffJSONBytes)
}

// doDiffMain runs the diff.
func doDiffMain(firstURL, secondURL string) <-chan diffMessage {
	ch := make(chan diffMessage, 10000)
	go func() {
		defer close(ch)
		// source and targets are always directories
		sourceSeparator := string(client.NewURL(firstURL).Separator)
		if !strings.HasSuffix(firstURL, sourceSeparator) {
			firstURL = firstURL + sourceSeparator
		}
		targetSeparator := string(client.NewURL(secondURL).Separator)
		if !strings.HasSuffix(secondURL, targetSeparator) {
			secondURL = secondURL + targetSeparator
		}

		firstClient, err := url2Client(firstURL)
		if err != nil {
			ch <- diffMessage{Error: err.Trace(firstURL)}
			return
		}
		difference, err := objectDifferenceFactory(secondURL)
		if err != nil {
			ch <- diffMessage{Error: err.Trace(secondURL)}
			return
		}
		isRecursive := true
		for sourceContent := range firstClient.List(isRecursive, false) {
			if sourceContent.Err != nil {
				ch <- diffMessage{Error: sourceContent.Err.Trace()}
				continue
			}
			if sourceContent.Content.Type.IsDir() {
				continue
			}
			suffix := strings.TrimPrefix(sourceContent.Content.URL.String(), firstURL)
			differ, err := difference(suffix, sourceContent.Content.Type, sourceContent.Content.Size)
			if err != nil {
				ch <- diffMessage{Error: err.Trace()}
				continue
			}
			if differ == differNone {
				continue
			}
			ch <- diffMessage{
				FirstURL:  sourceContent.Content.URL.String(),
				SecondURL: urlJoinPath(secondURL, suffix),
				Diff:      differ,
			}
		}
	}()
	return ch
}

// mainDiff main for 'diff'.
func mainDiff(ctx *cli.Context) {
	checkDiffSyntax(ctx)

	// Additional command speific theme customization.
	console.SetColor("DiffMessage", color.New(color.FgGreen, color.Bold))
	console.SetColor("DiffOnlyInFirst", color.New(color.FgRed, color.Bold))
	console.SetColor("DiffType", color.New(color.FgYellow, color.Bold))
	console.SetColor("DiffSize", color.New(color.FgMagenta, color.Bold))

	config := mustGetMcConfig()
	firstArg := ctx.Args().First()
	secondArg := ctx.Args().Last()

	firstURL := getAliasURL(firstArg, config.Aliases)
	secondURL := getAliasURL(secondArg, config.Aliases)

	_, firstContent, err := url2Stat(firstURL)
	if err != nil {
		fatalIf(err.Trace(), fmt.Sprintf("Unable to stat '%s'", firstURL))
	}
	if !firstContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("‘%s’ is not a folder", firstURL))
	}
	_, secondContent, err := url2Stat(secondURL)
	if err != nil {
		fatalIf(err.Trace(), fmt.Sprintf("Unable to stat '%s'", secondURL))
	}
	if !secondContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("‘%s’ is not a folder", secondURL))
	}

	for diff := range doDiffMain(firstURL, secondURL) {
		if diff.Error != nil {
			// Print in new line and adjust to top so that we don't print over the ongoing scan bar
			if !globalQuietFlag && !globalJSONFlag {
				console.Eraseline()
			}
		}
		fatalIf(diff.Error.Trace(firstURL, secondURL), "Failed to diff ‘"+firstURL+"’ and ‘"+secondURL+"’.")
		printMsg(diff)
	}
	// Print in new line and adjust to top so that we don't print over the ongoing scan bar
	if !globalQuietFlag && !globalJSONFlag {
		console.Eraseline()
	}
}
