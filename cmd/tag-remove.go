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

var tagRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove tags assigned to a bucket or an object",
	Action: mainRemoveTag,
	Before: initBeforeRunningCmd,
	Flags:  globalFlags,
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

  2. Remove the tags assigned to a bucket.
     {{.Prompt}} {{.HelpName}} play/testbucket
`,
}

// tagSetTagMessage structure will show message depending on the type of console.
type tagRemoveMessage struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}

// tagRemoveMessage console colorized output.
func (t tagRemoveMessage) String() string {
	return console.Colorize("Remove", "Tags removed for "+t.Name+".")
}

// JSON tagRemoveMessage.
func (t tagRemoveMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(t, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}
func checkRemoveTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "remove", globalErrorExitStatus)
	}
}

func mainRemoveTag(cliCtx *cli.Context) error {
	ctx, cancelList := context.WithCancel(globalContext)
	defer cancelList()

	checkRemoveTagSyntax(cliCtx)

	console.SetColor("Remove", color.New(color.FgGreen))

	targetURL := cliCtx.Args().Get(0)
	clnt, pErr := newClient(targetURL)
	fatalIf(pErr, "Unable to initialize target "+targetURL)
	pErr = clnt.DeleteTags(ctx)
	fatalIf(pErr, "Unable to remove tags for "+targetURL)

	printMsg(tagRemoveMessage{
		Status: "success",
		Name:   targetURL,
	})
	return nil
}
