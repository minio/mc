/*
 * MinIO Client, (C) 2020 MinIO, Inc.
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

package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var (
	undoFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "recursive, r",
			Usage: "undo last S3 put/delete operations",
		},
		cli.BoolFlag{
			Name:  "force",
			Usage: "force recursive operation",
		},
		cli.IntFlag{
			Name:  "last",
			Usage: "undo N last changes",
			Value: 1,
		},
		cli.BoolFlag{
			Name:  "dry-run",
			Usage: "fake an undo operation",
		},
	}
)

var undoCmd = cli.Command{
	Name:   "undo",
	Usage:  "undo PUT/DELETE operations",
	Action: mainUndo,
	Before: setGlobalsFromContext,
	Flags:  append(undoFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] SOURCE [SOURCE...]
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}

EXAMPLES:
  1. Undo the last 3 uploads and/or removals of a particular object
     {{.Prompt}} {{.HelpName}} s3/backups/file.zip --last 3

  2. Undo the last upload/removal change of all objects under a prefix
     {{.Prompt}} {{.HelpName}} --recursive --force s3/backups/prefix/
`,
}

// undoMessage container for undo message structure.
type undoMessage struct {
	Status         string `json:"status"`
	URL            string `json:"url,omitempty"`
	Key            string `json:"key,omitempty"`
	VersionID      string `json:"versionId,omitempty"`
	IsDeleteMarker bool   `json:"isDeleteMarker,omitempty"`
}

// String colorized string message.
func (c undoMessage) String() string {
	var msg string
	fmt.Print(color.GreenString("\u2713 "))
	yellow := color.New(color.FgYellow).SprintFunc()
	if c.IsDeleteMarker {
		msg += "Last " + color.RedString("delete") + " of `" + yellow(c.Key) + "` is reverted"
	} else {
		msg += "Last " + color.BlueString("upload") + " of `" + yellow(c.Key) + "` (vid=" + c.VersionID + ") is reverted"
	}
	msg += "."
	return msg
}

// JSON jsonified content message.
func (c undoMessage) JSON() string {
	c.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// parseUndoSyntax performs command-line input validation for cat command.
func parseUndoSyntax(ctx *cli.Context) (targetAliasedURL string, last int, recursive, dryRun bool) {
	targetAliasedURL = ctx.Args().Get(0)
	if targetAliasedURL == "" {
		fatalIf(errInvalidArgument().Trace(), "The argument should not be empty")
	}

	last = ctx.Int("last")
	if last < 1 {
		fatalIf(errInvalidArgument().Trace(), "--last value should be a positive integer")
	}

	recursive = ctx.Bool("recursive")
	force := ctx.Bool("force")
	if recursive && !force {
		fatalIf(errInvalidArgument().Trace(), "This is a dangerous operation, you need to provide --force flag as well")
	}

	dryRun = ctx.Bool("dry-run")
	return
}

func undoLastNOperations(ctx context.Context, clnt Client, objectVersions []*ClientContent, last int, dryRun bool) (exitErr error) {
	if last == 0 {
		return
	}

	sortObjectVersions(objectVersions)

	if len(objectVersions) > last {
		objectVersions = objectVersions[:last]
	}

	contentCh := make(chan *ClientContent)
	errorCh := clnt.Remove(ctx, false, false, false, contentCh)

	prefixPath := clnt.GetURL().Path
	prefixPath = filepath.ToSlash(prefixPath)
	if !strings.HasSuffix(prefixPath, "/") {
		prefixPath = prefixPath[:strings.LastIndex(prefixPath, "/")+1]
	}
	prefixPath = strings.TrimPrefix(prefixPath, "./")

	go func() {
		for _, objectVersion := range objectVersions {
			if !dryRun {
				contentCh <- objectVersion
			}

			// Convert any os specific delimiters to "/".
			contentURL := filepath.ToSlash(objectVersion.URL.Path)
			// Trim prefix path from the content path.
			keyName := strings.TrimPrefix(contentURL, prefixPath)

			printMsg(undoMessage{
				Status:         "success",
				Key:            getOSDependantKey(keyName, objectVersion.Type.IsDir()),
				URL:            objectVersion.URL.String(),
				VersionID:      objectVersion.VersionID,
				IsDeleteMarker: objectVersion.IsDeleteMarker,
			})

		}
		close(contentCh)
	}()

	for err := range errorCh {
		if err != nil {
			errorIf(err.Trace(), "Unable to undo")
			exitErr = exitStatus(globalErrorExitStatus) // Set the exit status.
		}
	}

	return
}

func undoURL(ctx context.Context, aliasedURL string, last int, recursive, dryRun bool) (exitErr error) {
	clnt, err := newClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize target `"+aliasedURL+"`.")

	alias, _, _ := mustExpandAlias(aliasedURL)

	var (
		lastObjectPath        string
		perObjectVersions     []*ClientContent
		atLeastOneUndoApplied bool
	)

	for content := range clnt.List(ctx, ListOptions{
		IsRecursive:       recursive,
		WithOlderVersions: true,
		WithDeleteMarkers: true,
		ShowDir:           DirNone,
	}) {
		if content.Err != nil {
			fatalIf(content.Err.Trace(clnt.GetURL().String()), "Unable to list folder.")
		}

		if content.StorageClass == s3StorageClassGlacier {
			continue
		}

		if !recursive {
			if alias+getKey(content) != getStandardizedURL(aliasedURL) {
				break
			}
		}

		if lastObjectPath != content.URL.Path {
			// Print any object in the current list before reinitializing it
			exitErr = undoLastNOperations(ctx, clnt, perObjectVersions, last, dryRun)
			lastObjectPath = content.URL.Path
			perObjectVersions = []*ClientContent{}
		}

		perObjectVersions = append(perObjectVersions, content)
		atLeastOneUndoApplied = true
	}

	// Undo the remaining versions found if any
	exitErr = undoLastNOperations(ctx, clnt, perObjectVersions, last, dryRun)

	if !atLeastOneUndoApplied {
		errorIf(errDummy().Trace(clnt.GetURL().String()), "Unable to find any object version to undo.")
		exitErr = exitStatus(globalErrorExitStatus) // Set the exit status.
	}

	return
}

func checkIfBucketIsVersioned(ctx context.Context, aliasedURL string) (versioned bool) {
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to parse `%s`", aliasedURL)

	versioningConfig, err := client.GetVersion(ctx)
	if err != nil {
		if errors.As(err.ToGoError(), &APINotImplemented{}) {
			return false
		}
		fatalIf(err.Trace(), "Unable to get bucket versioning info")
	}

	if versioningConfig.Status == "Enabled" {
		return true
	}
	return false
}

// mainUndo is the main entry point for undo command.
func mainUndo(cliCtx *cli.Context) error {
	ctx, cancelCat := context.WithCancel(globalContext)
	defer cancelCat()

	console.SetColor("Success", color.New(color.FgGreen, color.Bold))

	// check 'undo' cli arguments.
	targetAliasedURL, last, recursive, dryRun := parseUndoSyntax(cliCtx)

	if !checkIfBucketIsVersioned(ctx, targetAliasedURL) {
		fatalIf(errDummy().Trace(), "Undo command works only with S3 versioned-enabled buckets.")
	}

	return undoURL(ctx, targetAliasedURL, last, recursive, dryRun)
}
