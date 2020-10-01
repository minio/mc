/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var tagRemoveFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "version-id, vid",
		Usage: "remove tags on a specific object version",
	},
	cli.StringFlag{
		Name:  "rewind",
		Usage: "remove tags on an object version at specified time",
	},
	cli.BoolFlag{
		Name:  "versions",
		Usage: "remote tags on multiple versions of an object",
	},
}

var tagRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove tags assigned to a bucket or an object",
	Action: mainRemoveTag,
	Before: setGlobalsFromContext,
	Flags:  append(tagRemoveFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  Remove tags assigned to a bucket or an object.

EXAMPLES:
  1. Remove the tags assigned to an object.
     {{.Prompt}} {{.HelpName}} myminio/testbucket/testobject

  2. Remove the tags assigned to a particular version of an object.
     {{.Prompt}} {{.HelpName}} --version-id "ieQq7aXsyhlhDt47YURGlrucYY3GxWHa" myminio/testbucket/testobject

  3. Remove the tags assigned to an object versions that are older than one week
     {{.Prompt}} {{.HelpName}} --versions --rewind 7d myminio/testbucket/testobject

  4. Remove the tags assigned to a bucket.
     {{.Prompt}} {{.HelpName}} play/testbucket
`,
}

// tagSetTagMessage structure will show message depending on the type of console.
type tagRemoveMessage struct {
	Status    string `json:"status"`
	Name      string `json:"name"`
	VersionID string `json:"versionID"`
}

// tagRemoveMessage console colorized output.
func (t tagRemoveMessage) String() string {
	var msg string
	msg += "Tags removed for " + t.Name
	if t.VersionID != "" {
		msg += " (" + t.VersionID + ")"
	}
	msg += "."
	return console.Colorize("Remove", msg)
}

// JSON tagRemoveMessage.
func (t tagRemoveMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(t, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func parseRemoveTagSyntax(ctx *cli.Context) (targetURL, versionID string, timeRef time.Time, withVersions bool) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "remove", globalErrorExitStatus)
	}

	targetURL = ctx.Args().Get(0)
	versionID = ctx.String("version-id")
	withVersions = ctx.Bool("versions")
	rewind := ctx.String("rewind")

	if versionID != "" && (rewind != "" || withVersions) {
		fatalIf(errDummy().Trace(), "You cannot specify both --version-id and --rewind or --versions flags at the same time")
	}

	timeRef = parseRewindFlag(rewind)
	return
}

// Delete tags of a bucket or a specified object/version
func deleteTags(ctx context.Context, clnt Client, versionID string, verbose bool) {
	targetName := clnt.GetURL().String()
	if versionID != "" {
		targetName += " (" + versionID + ")"
	}

	err := clnt.DeleteTags(ctx, versionID)
	if err != nil {
		fatalIf(err, "Unable to remove tags for "+targetName)
		return
	}

	printMsg(tagRemoveMessage{
		Status:    "success",
		Name:      clnt.GetURL().String(),
		VersionID: versionID,
	})
}

func mainRemoveTag(cliCtx *cli.Context) error {
	ctx, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	console.SetColor("Remove", color.New(color.FgGreen))

	targetURL, versionID, timeRef, withVersions := parseRemoveTagSyntax(cliCtx)
	if timeRef.IsZero() && withVersions {
		timeRef = time.Now().UTC()
	}

	clnt, pErr := newClient(targetURL)
	fatalIf(pErr, "Unable to initialize target "+targetURL)

	if timeRef.IsZero() && !withVersions {
		deleteTags(ctx, clnt, versionID, true)
	} else {
		for content := range clnt.List(ctx, ListOptions{TimeRef: timeRef, WithOlderVersions: withVersions}) {
			if content.Err != nil {
				fatalIf(content.Err.Trace(), "Unable to list target "+targetURL)
			}
			deleteTags(ctx, clnt, content.VersionID, false)
		}
	}
	return nil
}
