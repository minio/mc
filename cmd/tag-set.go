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

var tagSetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "vid",
		Usage: "Set tags of particular object version",
	},
	cli.StringFlag{
		Name:  "rewind",
		Usage: "Go back in time",
	},
	cli.BoolFlag{
		Name:  "versions",
		Usage: "Select multiple versions for an object",
	},
}

var tagSetCmd = cli.Command{
	Name: "set", Usage: "set tags for a bucket and object(s)", Action: mainSetTag,
	Before: setGlobalsFromContext,
	Flags:  append(tagSetFlags, globalFlags...),
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
     {{.Prompt}} {{.HelpName}} --vid "ieQq7aXsyhlhDt47YURGlrucYY3GxWHa" play/testbucket/testobject "key1=value1&key2=value2&key3=value3"

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
	versionID = ctx.String("vid")
	withVersions = ctx.Bool("versions")
	rewind := ctx.String("rewind")

	if versionID != "" && (rewind != "" || withVersions) {
		fatalIf(errDummy().Trace(), "You cannot specify both --vid and --rewind or --versions flags at the same time")
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
		for content := range clnt.List(ctx, ListOptions{timeRef: timeRef, withOlderVersions: withVersions}) {
			if content.Err != nil {
				fatalIf(content.Err.Trace(), "Unable to list target "+targetURL)
			}
			setTags(ctx, clnt, content.VersionID, tags, false)
		}
	}

	return nil
}
