// Copyright (c) 2015-2022 MinIO, Inc.
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
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var tagSetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "version-id, vid",
		Usage: "set tags on a specific object version",
	},
	cli.StringFlag{
		Name:  "rewind",
		Usage: "set tags on a specific object version at specific time",
	},
	cli.BoolFlag{
		Name:  "versions",
		Usage: "set tags on multiple versions for an object",
	},
	cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "recursivley set tags for all objects of subdirs",
	},
	cli.BoolFlag{
		Name:  "exclude-folders",
		Usage: "exclude setting tags on folder objects",
	},
}

var tagSetCmd = cli.Command{
	Name: "set", Usage: "set tags for a bucket and object(s)",
	Action:       mainSetTag,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(tagSetFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET TAGS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
   Assign tags to a bucket or an object.

EXAMPLES:
  1. Assign tags to an object.
     {{.Prompt}} {{.HelpName}} play/testbucket/testobject "key1=value1&key2=value2&key3=value3"

  2. Assign tags to a particuler version of an object.
     {{.Prompt}} {{.HelpName}} --version-id "ieQq7aXsyhlhDt47YURGlrucYY3GxWHa" play/testbucket/testobject "key1=value1&key2=value2&key3=value3"

  3. Assign tags to a object versions older than one week.
     {{.Prompt}} {{.HelpName}} --versions --rewind 7d play/testbucket/testobject "key1=value1&key2=value2&key3=value3"

  4. Assign tags to a bucket.
     {{.Prompt}} {{.HelpName}} myminio/testbucket "key1=value1&key2=value2&key3=value3"

  5. Assign tags recursively to all the objects of subdirs of bucket.
     {{.Prompt}} {{.HelpName}} myminio/testbucket --recursive "key1=value1&key2=value2&key3=value3"

  6. Assign tags recursively to all versions of all objects of subdirs of bucket.
     {{.Prompt}} {{.HelpName}} myminio/testbucket --recursive --versions "key1=value1&key2=value2&key3=value3"

  7. Assign tags to all the objects on a bucket, excluding folders
     {{.Prompt}} {{.HelpName}} myminio/testbucket --exclude-folders --recursive "key1=value1&key2=value2&key3=value3"
`,
}

// tagSetTagMessage structure will show message depending on the type of console.
type tagSetMessage struct {
	Status    string `json:"status"`
	Name      string `json:"name"`
	VersionID string `json:"versionID"`
}

// tagSetMessage console colorized output.
func (t tagSetMessage) String() string {
	var msg string
	msg += "Tags set for " + t.Name
	if t.VersionID != "" {
		msg += " (" + t.VersionID + ")"
	}
	msg += "."
	return console.Colorize("List", msg)
}

// JSON tagSetMessage.
func (t tagSetMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(t, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func parseSetTagSyntax(ctx *cli.Context) (targetURL, versionID string, timeRef time.Time, withVersions bool, tags string, recursive bool, excludeFolders bool) {
	if len(ctx.Args()) != 2 || ctx.Args().Get(1) == "" {
		showCommandHelpAndExit(ctx, globalErrorExitStatus)
	}

	targetURL = ctx.Args().Get(0)
	tags = ctx.Args().Get(1)
	versionID = ctx.String("version-id")
	withVersions = ctx.Bool("versions")
	rewind := ctx.String("rewind")
	recursive = ctx.Bool("recursive")
	excludeFolders = ctx.Bool("exclude-folders")

	if versionID != "" && (rewind != "" || withVersions) {
		fatalIf(errDummy().Trace(), "You cannot specify both --version-id and --rewind or --versions flags at the same time")
	}

	if excludeFolders && !recursive {
		fatalIf(errDummy().Trace(), "'--exclude-folders' must be used with --recursive only")
	}

	timeRef = parseRewindFlag(rewind)
	return
}

// Set tags to a bucket or to a specified object/version
func setTags(ctx context.Context, clnt Client, versionID, tags string) {
	targetName := clnt.GetURL().String()
	if versionID != "" {
		targetName += " (" + versionID + ")"
	}

	err := clnt.SetTags(ctx, versionID, tags)
	if err != nil {
		fatalIf(err.Trace(tags), "Failed to set tags for "+targetName)
		return
	}
	printMsg(tagSetMessage{
		Status:    "success",
		Name:      clnt.GetURL().String(),
		VersionID: versionID,
	})
}

func setTagsSingle(ctx context.Context, alias, url, versionID, tags string) *probe.Error {
	newClnt, err := newClientFromAlias(alias, url)
	if err != nil {
		return err
	}

	setTags(ctx, newClnt, versionID, tags)
	return nil
}

func mainSetTag(cliCtx *cli.Context) error {
	ctx, cancelSetTag := context.WithCancel(globalContext)
	defer cancelSetTag()

	console.SetColor("List", color.New(color.FgGreen))

	targetURL, versionID, timeRef, withVersions, tags, recursive, excludeFolders := parseSetTagSyntax(cliCtx)
	if timeRef.IsZero() && withVersions {
		timeRef = time.Now().UTC()
	}

	clnt, err := newClient(targetURL)
	fatalIf(err.Trace(cliCtx.Args()...), "Unable to initialize target "+targetURL)

	alias, urlStr, _ := mustExpandAlias(targetURL)
	if timeRef.IsZero() && !withVersions && !recursive && !excludeFolders {
		err := setTagsSingle(ctx, alias, urlStr, versionID, tags)
		fatalIf(err.Trace(), "Unable to set tags on `%s`", targetURL)
		return nil
	}
	for content := range clnt.List(ctx, ListOptions{TimeRef: timeRef, WithOlderVersions: withVersions, Recursive: recursive}) {
		if content.Err != nil {
			fatalIf(content.Err.Trace(), "Unable to list target "+targetURL)
			continue
		}

		// Dont set tag for the delete marker
		if content.IsDeleteMarker {
			continue
		}

		// if excludeFolders dont set tags for subdirs
		_, objName := url2BucketAndObject(&content.URL)
		if strings.Index(objName, string(content.URL.Separator)) > 0 && excludeFolders {
			continue
		}

		if !recursive && getStandardizedURL(alias+getKey(content)) != getStandardizedURL(targetURL) {
			break
		}

		err := setTagsSingle(ctx, alias, content.URL.String(), content.VersionID, tags)
		if err != nil {
			errorIf(err.Trace(clnt.GetURL().String()), "Invalid URL")
			continue
		}
	}

	return nil
}
