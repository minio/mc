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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var tagSetCmd = cli.Command{
	Name:   "set",
	Usage:  "set tags for a bucket(s) and object(s)",
	Action: mainSetTag,
	Before: initBeforeRunningCmd,
	Flags:  globalFlags,
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

  2. Assign tags to a bucket.
     {{.Prompt}} {{.HelpName}} myminio/testbucket "key1=value1&key2=value2&key3=value3"
`,
}

// tagSetTagMessage structure will show message depending on the type of console.
type tagSetMessage struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}

// tagSetMessage console colorized output.
func (t tagSetMessage) String() string {
	return console.Colorize("List", "Tags set for "+t.Name+".")
}

// JSON tagSetMessage.
func (t tagSetMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(t, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func checkSetTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 || ctx.Args().Get(1) == "" {
		cli.ShowCommandHelpAndExit(ctx, "set", globalErrorExitStatus)
	}
}

func mainSetTag(cliCtx *cli.Context) error {
	ctx, cancelSetTag := context.WithCancel(globalContext)
	defer cancelSetTag()

	checkSetTagSyntax(cliCtx)
	console.SetColor("List", color.New(color.FgGreen))

	targetURL := cliCtx.Args().Get(0)
	tags := cliCtx.Args().Get(1)

	clnt, err := newClient(targetURL)
	fatalIf(err.Trace(cliCtx.Args()...), "Unable to initialize target "+targetURL)

	fatalIf(clnt.SetTags(ctx, tags).Trace(tags), "Failed to set tags for "+targetURL)

	printMsg(tagSetMessage{
		Status: "success",
		Name:   targetURL,
	})
	return nil
}
