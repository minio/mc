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

// diff specific flags.
var (
	diffFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "help, h",
			Usage: "Help of diff.",
		},
	}
)

// Compute differences between two files or folders.
var diffCmd = cli.Command{
	Name:        "diff",
	Usage:       "Compute differences between two folders.",
	Description: "Diff only lists missing objects or objects with size differences. It *DOES NOT* compare contents. i.e. Objects of same name and size, but differ in contents are not noticed.",
	Action:      mainDiff,
	Flags:       append(diffFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc {{.Name}} - {{.Usage}}

USAGE:
   mc {{.Name}} [FLAGS] FIRST SECOND

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
DESCRIPTION:
   {{.Description}}

EXAMPLES:
   1. Compare a local folder with a folder on Amazon S3 cloud storage.
      $ mc {{.Name}} ~/Photos s3/MyBucket/Photos

   2. Compare two different folders on a local filesystem.
      $ mc {{.Name}} ~/Photos /Media/Backup/Photos
`,
}

func checkDiffSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "diff", 1) // last argument is exit code
	}
	for _, arg := range ctx.Args() {
		if strings.TrimSpace(arg) == "" {
			fatalIf(errInvalidArgument().Trace(ctx.Args()...), "Unable to validate empty argument.")
		}
	}
}

// diffMessage json container for diff messages
type diffMessage struct {
	Status    string       `json:"status"`
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
		fatalIf(errDummy().Trace(d.FirstURL, d.SecondURL),
			"Unhandled difference between ‘"+d.FirstURL+"’ and ‘"+d.SecondURL+"’.")
	}
	return msg

}

// JSON jsonified diff message
func (d diffMessage) JSON() string {
	d.Status = "success"
	diffJSONBytes, err := json.Marshal(d)
	fatalIf(probe.NewError(err),
		"Unable to marshal diff message ‘"+d.FirstURL+"’, ‘"+d.SecondURL+"’ and ‘"+d.Diff+"’.")
	return string(diffJSONBytes)
}

// doDiffMain runs the diff.
func doDiffMain(firstURL, secondURL string) {
	// source and targets are always directories
	sourceSeparator := string(client.NewURL(firstURL).Separator)
	if !strings.HasSuffix(firstURL, sourceSeparator) {
		firstURL = firstURL + sourceSeparator
	}
	targetSeparator := string(client.NewURL(secondURL).Separator)
	if !strings.HasSuffix(secondURL, targetSeparator) {
		secondURL = secondURL + targetSeparator
	}

	firstClient, err := newClient(firstURL)
	if err != nil {
		fatalIf(err.Trace(firstURL, secondURL), fmt.Sprintf("Failed to diff '%s' and '%s'", firstURL, secondURL))
	}
	difference, err := objectDifferenceFactory(secondURL)
	if err != nil {
		fatalIf(err.Trace(firstURL, secondURL), fmt.Sprintf("Failed to diff '%s' and '%s'", firstURL, secondURL))
	}
	isRecursive := true
	isIncomplete := false
	for sourceContent := range firstClient.List(isRecursive, isIncomplete) {
		if sourceContent.Err != nil {
			switch sourceContent.Err.ToGoError().(type) {
			// handle this specifically for filesystem related errors.
			case client.BrokenSymlink, client.TooManyLevelsSymlink, client.PathNotFound, client.PathInsufficientPermission:
				errorIf(sourceContent.Err.Trace(firstURL, secondURL), fmt.Sprintf("Failed on '%s'", firstURL))
			default:
				fatalIf(sourceContent.Err.Trace(firstURL, secondURL), fmt.Sprintf("Failed on '%s'", firstURL))
			}
			continue
		}
		if sourceContent.Type.IsDir() {
			continue
		}
		suffix := strings.TrimPrefix(sourceContent.URL.String(), firstURL)
		differ, err := difference(suffix, sourceContent.Type, sourceContent.Size)
		if err != nil {
			errorIf(sourceContent.Err.Trace(secondURL, suffix),
				fmt.Sprintf("Failed on '%s'", urlJoinPath(secondURL, suffix)))
			continue
		}
		if differ == differNone {
			continue
		}
		printMsg(diffMessage{
			FirstURL:  sourceContent.URL.String(),
			SecondURL: urlJoinPath(secondURL, suffix),
			Diff:      differ,
		})
	}
}

// mainDiff main for 'diff'.
func mainDiff(ctx *cli.Context) {
	// Set global flags from context.
	setGlobalsFromContext(ctx)

	// check 'diff' cli arguments.
	checkDiffSyntax(ctx)

	// Additional command specific theme customization.
	console.SetColor("DiffMessage", color.New(color.FgGreen, color.Bold))
	console.SetColor("DiffOnlyInFirst", color.New(color.FgRed, color.Bold))
	console.SetColor("DiffType", color.New(color.FgYellow, color.Bold))
	console.SetColor("DiffSize", color.New(color.FgMagenta, color.Bold))

	URLs := ctx.Args()
	firstURL := URLs[0]
	secondURL := URLs[1]

	_, firstContent, err := url2Stat(firstURL)
	if err != nil {
		fatalIf(err.Trace(firstURL), fmt.Sprintf("Unable to stat '%s'.", firstURL))
	}
	if !firstContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(firstURL), fmt.Sprintf("‘%s’ is not a folder.", firstURL))
	}
	_, secondContent, err := url2Stat(secondURL)
	if err != nil {
		fatalIf(err.Trace(secondURL), fmt.Sprintf("Unable to stat '%s'.", secondURL))
	}
	if !secondContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(secondURL), fmt.Sprintf("‘%s’ is not a folder.", secondURL))
	}
	doDiffMain(firstURL, secondURL)
}
