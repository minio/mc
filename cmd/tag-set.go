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
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
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

func parseSetTagSyntax(ctx *cli.Context) (targetURL, versionID string, timeRef time.Time, withVersions bool, tags string) {
	if len(ctx.Args()) != 2 || ctx.Args().Get(1) == "" {
		cli.ShowCommandHelpAndExit(ctx, "set", globalErrorExitStatus)
	}

	targetURL = ctx.Args().Get(0)
	tags = ctx.Args().Get(1)
	versionID = ctx.String("version-id")
	withVersions = ctx.Bool("versions")
	rewind := ctx.String("rewind")

	if versionID != "" && (rewind != "" || withVersions) {
		fatalIf(errDummy().Trace(), "You cannot specify both --version-id and --rewind or --versions flags at the same time")
	}

	timeRef = parseRewindFlag(rewind)
	return
}

// Set tags to a bucket or to a specified object/version
func setTags(ctx context.Context, clnt Client, versionID, tags string, verbose bool) {
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

func mainSetTag(cliCtx *cli.Context) error {
	ctx, cancelSetTag := context.WithCancel(globalContext)
	defer cancelSetTag()

	console.SetColor("List", color.New(color.FgGreen))

	targetURL, versionID, timeRef, withVersions, tags := parseSetTagSyntax(cliCtx)
	if timeRef.IsZero() && withVersions {
		timeRef = time.Now().UTC()
	}

	clnt, err := newClient(targetURL)
	fatalIf(err.Trace(cliCtx.Args()...), "Unable to initialize target "+targetURL)

	if timeRef.IsZero() && !withVersions {
		setTags(ctx, clnt, versionID, tags, true)
	} else {
		for content := range clnt.List(ctx, ListOptions{TimeRef: timeRef, WithOlderVersions: withVersions}) {
			if content.Err != nil {
				fatalIf(content.Err.Trace(), "Unable to list target "+targetURL)
			}
			setTags(ctx, clnt, content.VersionID, tags, false)
		}
	}

	return nil
}
