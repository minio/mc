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
	"os"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var tagRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove tags assigned to an object",
	Action: mainRemoveTag,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Remove the tags assigned to an object.
     {{.Prompt}} {{.HelpName}} myminio/testbucket/testobject
`,
}

// tagSetTagMessage structure will show message depending on the type of console.
type tagRemoveMessage struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}

// tagRemoveMessage console colorized output.
func (t tagRemoveMessage) String() string {
	return console.Colorize(tagPrintMsgTheme, "Tags removed for "+t.Name+".")
}

// JSON tagRemoveMessage.
func (t tagRemoveMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(t, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}
func checkRemoveTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "remove")
		os.Exit(globalErrorExitStatus)
	}
}

func mainRemoveTag(ctx *cli.Context) error {
	checkRemoveTagSyntax(ctx)
	setTagListColorScheme()

	objectURL := ctx.Args().Get(0)

	clnt, err := newClient(objectURL)
	fatalIf(err.Trace(objectURL), "Unable to initialize target")

	fatalIf(clnt.DeleteObjectTagging().Trace(objectURL), "Unable to remove tags")

	printMsg(tagRemoveMessage{
		Status: "success",
		Name:   objectURL,
	})

	return nil
}
