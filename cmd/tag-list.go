// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/pkg/console"
)

var tagListFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "version-id, vid",
		Usage: "list tags of particular object version",
	},
	cli.StringFlag{
		Name:  "rewind",
		Usage: "list tags of particular object version at specified time",
	},
	cli.BoolFlag{
		Name:  "versions",
		Usage: "list tags on all versions for an object",
	},
}

var tagListCmd = cli.Command{
	Name:         "list",
	Usage:        "list tags of a bucket or an object",
	Action:       mainListTag,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(tagListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
   List tags assigned to a bucket or an object

EXAMPLES:
  1. List the tags assigned to an object.
     {{.Prompt}} {{.HelpName}} myminio/testbucket/testobject

  2. List the tags assigned to particular version of an object.
     {{.Prompt}} {{.HelpName}} --version-id "ieQq7aXsyhlhDt47YURGlrucYY3GxWHa" myminio/testbucket/testobject

  3. List the tags assigned to an object versions that are older than one week.
     {{.Prompt}} {{.HelpName}} --versions --rewind 7d myminio/testbucket/testobject

  4. List the tags assigned to an object in JSON format.
     {{.Prompt}} {{.HelpName}} --json myminio/testbucket/testobject

  5. List the tags assigned to a bucket.
     {{.Prompt}} {{.HelpName}} myminio/testbucket

  6. List the tags assigned to a bucket in JSON format.
     {{.Prompt}} {{.HelpName}} --json s3/testbucket
`,
}

// tagListMessage structure for displaying tag
type tagListMessage struct {
	Tags      map[string]string `json:"tagset,omitempty"`
	Status    string            `json:"status"`
	URL       string            `json:"url"`
	VersionID string            `json:"versionID"`
}

func (t tagListMessage) JSON() string {
	tagJSONbytes, err := json.MarshalIndent(t, "", "  ")
	fatalIf(probe.NewError(err), "Unable to marshal into JSON for "+t.URL)
	return string(tagJSONbytes)
}

func (t tagListMessage) String() string {
	keys := []string{}
	maxKeyLen := 4 // len("Name")
	for key := range t.Tags {
		keys = append(keys, key)
		if len(key) > maxKeyLen {
			maxKeyLen = len(key)
		}
	}
	sort.Strings(keys)

	maxKeyLen += 2 // add len(" :")
	strs := []string{
		fmt.Sprintf("%v%*v %v", console.Colorize("Name", "Name"), maxKeyLen-4, ":", console.Colorize("Name", t.URL+" ("+t.VersionID+")")),
	}

	for _, key := range keys {
		strs = append(
			strs,
			fmt.Sprintf("%v%*v %v", console.Colorize("Key", key), maxKeyLen-len(key), ":", console.Colorize("Value", t.Tags[key])),
		)
	}

	if len(keys) == 0 {
		strs = append(strs, console.Colorize("NoTags", "No tags found"))
	}

	return strings.Join(strs, "\n")
}

// parseTagListSyntax performs command-line input validation for tag list command.
func parseTagListSyntax(ctx *cli.Context) (targetURL, versionID string, timeRef time.Time, withOlderVersions bool) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "list", globalErrorExitStatus)
	}

	targetURL = ctx.Args().Get(0)
	versionID = ctx.String("version-id")
	withOlderVersions = ctx.Bool("versions")
	rewind := ctx.String("rewind")

	if versionID != "" && rewind != "" {
		fatalIf(errDummy().Trace(), "You cannot specify both --version-id and --rewind flags at the same time")
	}

	timeRef = parseRewindFlag(rewind)
	return
}

// showTags pretty prints tags of a bucket or a specified object/version
func showTags(ctx context.Context, clnt Client, versionID string, verbose bool) {
	targetName := clnt.GetURL().String()
	if versionID != "" {
		targetName += " (" + versionID + ")"
	}

	tagsMap, err := clnt.GetTags(ctx, versionID)
	if err != nil {
		if minio.ToErrorResponse(err.ToGoError()).Code == "NoSuchTagSet" {
			fatalIf(probe.NewError(errors.New("check 'mc tag set --help' on how to set tags")), "No tags found  for "+targetName)
		}
		fatalIf(err, "Unable to fetch tags for "+targetName)
		return
	}

	printMsg(tagListMessage{
		Tags:      tagsMap,
		Status:    "success",
		URL:       clnt.GetURL().String(),
		VersionID: versionID,
	})
}

func mainListTag(cliCtx *cli.Context) error {
	ctx, cancelListTag := context.WithCancel(globalContext)
	defer cancelListTag()

	console.SetColor("Name", color.New(color.Bold, color.FgCyan))
	console.SetColor("Key", color.New(color.FgGreen))
	console.SetColor("Value", color.New(color.FgYellow))
	console.SetColor("NoTags", color.New(color.FgRed))

	targetURL, versionID, timeRef, withVersions := parseTagListSyntax(cliCtx)
	if timeRef.IsZero() && withVersions {
		timeRef = time.Now().UTC()
	}

	clnt, err := newClient(targetURL)
	fatalIf(err, "Unable to initialize target "+targetURL)

	if timeRef.IsZero() && !withVersions {
		showTags(ctx, clnt, versionID, true)
	} else {
		for content := range clnt.List(ctx, ListOptions{TimeRef: timeRef, WithOlderVersions: withVersions}) {
			if content.Err != nil {
				fatalIf(content.Err.Trace(), "Unable to list target "+targetURL)
			}
			showTags(ctx, clnt, content.VersionID, false)
		}
	}

	return nil
}
