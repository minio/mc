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
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
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

// diffMessage json container for diff messages
type diffMessage struct {
	Status    string       `json:"status"`
	FirstURL  string       `json:"first"`
	SecondURL string       `json:"second"`
	Diff      differType   `json:"diff"`
	Error     *probe.Error `json:"error,omitempty"`
}

// String colorized diff message
func (d diffMessage) String() string {
	msg := ""
	switch d.Diff {
	case differInFirst:
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffOnlyInFirst", " - only in first.")
	case differInType:
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffType", " - differ in type.")
	case differInSize:
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffSize", " - differ in size.")
	case differInTime:
		msg = console.Colorize("DiffMessage",
			"‘"+d.FirstURL+"’"+" and "+"‘"+d.SecondURL+"’") + console.Colorize("DiffTime", " - differ in modified time.")
	default:
		fatalIf(errDummy().Trace(d.FirstURL, d.SecondURL),
			"Unhandled difference between ‘"+d.FirstURL+"’ and ‘"+d.SecondURL+"’.")
	}
	return msg

}

// JSON jsonified diff message
func (d diffMessage) JSON() string {
	d.Status = "success"
	diffJSONBytes, e := json.Marshal(d)
	fatalIf(probe.NewError(e),
		"Unable to marshal diff message ‘"+d.FirstURL+"’, ‘"+d.SecondURL+"’ and ‘"+string(d.Diff)+"’.")
	return string(diffJSONBytes)
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
	URLs := ctx.Args()
	firstURL := URLs[0]
	secondURL := URLs[1]

	// Diff only works between two directories, verify them below.

	// Verify if firstURL is accessible.
	_, firstContent, err := url2Stat(firstURL)
	if err != nil {
		fatalIf(err.Trace(firstURL), fmt.Sprintf("Unable to stat '%s'.", firstURL))
	}

	// Verify if its a directory.
	if !firstContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(firstURL), fmt.Sprintf("‘%s’ is not a folder.", firstURL))
	}

	// Verify if secondURL is accessible.
	_, secondContent, err := url2Stat(secondURL)
	if err != nil {
		fatalIf(err.Trace(secondURL), fmt.Sprintf("Unable to stat '%s'.", secondURL))
	}

	// Verify if its a directory.
	if !secondContent.Type.IsDir() {
		fatalIf(errInvalidArgument().Trace(secondURL), fmt.Sprintf("‘%s’ is not a folder.", secondURL))
	}
}

// doDiffMain runs the diff.
func doDiffMain(firstURL, secondURL string) {
	// Source and targets are always directories
	sourceSeparator := string(newClientURL(firstURL).Separator)
	if !strings.HasSuffix(firstURL, sourceSeparator) {
		firstURL = firstURL + sourceSeparator
	}
	targetSeparator := string(newClientURL(secondURL).Separator)
	if !strings.HasSuffix(secondURL, targetSeparator) {
		secondURL = secondURL + targetSeparator
	}

	// Expand aliased urls.
	firstAlias, firstURL, _ := mustExpandAlias(firstURL)
	secondAlias, secondURL, _ := mustExpandAlias(secondURL)

	firstClient, err := newClientFromAlias(firstAlias, firstURL)
	if err != nil {
		fatalIf(err.Trace(firstAlias, firstURL, secondAlias, secondURL),
			fmt.Sprintf("Failed to diff '%s' and '%s'", firstURL, secondURL))
	}

	secondClient, err := newClientFromAlias(secondAlias, secondURL)
	if err != nil {
		fatalIf(err.Trace(firstAlias, firstURL, secondAlias, secondURL),
			fmt.Sprintf("Failed to diff '%s' and '%s'", firstURL, secondURL))
	}

	// Setup object difference function.
	difference := objectDifferenceFactory(secondClient)

	// Set default values for listing.
	isRecursive := true   // recursive is always true for diff.
	isIncomplete := false // we will not compare any incomplete objects.

	// List all the elements on first URL and compare with
	// 'difference' function.
	for firstContent := range firstClient.List(isRecursive, isIncomplete) {
		if firstContent.Err != nil {
			switch firstContent.Err.ToGoError().(type) {
			// Handle this specifically for filesystem related errors.
			case BrokenSymlink, TooManyLevelsSymlink, PathNotFound, PathInsufficientPermission:
				errorIf(firstContent.Err.Trace(firstURL, secondURL), fmt.Sprintf("Failed on '%s'", firstURL))
			default:
				fatalIf(firstContent.Err.Trace(firstURL, secondURL), fmt.Sprintf("Failed on '%s'", firstURL))
			}
			continue
		}
		if firstContent.Type.IsDir() {
			continue
		}
		firstSuffix := strings.TrimPrefix(firstContent.URL.String(), firstURL)
		differ, err := difference(secondURL, firstSuffix, firstContent.Type, firstContent.Size, firstContent.Time)
		if err != nil {
			errorIf(firstContent.Err.Trace(secondURL, firstSuffix),
				fmt.Sprintf("Failed on '%s'", urlJoinPath(secondURL, firstSuffix)))
			continue
		}
		if differ == differInNone {
			continue
		}
		printMsg(diffMessage{
			FirstURL:  firstContent.URL.String(),
			SecondURL: urlJoinPath(secondURL, firstSuffix),
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
	console.SetColor("DiffTime", color.New(color.FgYellow, color.Bold))

	URLs := ctx.Args()
	firstURL := URLs[0]
	secondURL := URLs[1]

	doDiffMain(firstURL, secondURL)
}
