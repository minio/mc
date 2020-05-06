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
	"github.com/minio/minio-go/v6/pkg/tags"
	"github.com/minio/minio/pkg/console"
)

var tagSetCmd = cli.Command{
	Name:   "set",
	Usage:  "set tags for an object",
	Action: mainSetTag,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET VALUE

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

VALUE:
  Value is of the form "k1=v1&k2=v2"

EXAMPLES:
  1. Set tags to an object.
     {{.Prompt}} {{.HelpName}} myminio/testbucket/testobject "key1=value1&key2=value2&key3=value3"
`,
}

// tagSetTagMessage structure will show message depending on the type of console.
type tagSetMessage struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}

// tagSetMessage console colorized output.
func (t tagSetMessage) String() string {
	return console.Colorize(tagPrintMsgTheme, "Tags set for "+t.Name+".")
}

// JSON tagSetMessage.
func (t tagSetMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(t, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func checkSetTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 || (len(ctx.Args()) == 2 && len(ctx.Args().Get(1)) == 0) {
		cli.ShowCommandHelp(ctx, "set")
		os.Exit(globalErrorExitStatus)
	}
}

func getTaggingMap(ctx *cli.Context) (*tags.Tags, *probe.Error) {
	if len(ctx.Args()) != 2 {
		return nil, errInvalidArgument().Trace(ctx.Args()...)
	}
	t, e := tags.ParseObjectTags(ctx.Args().Get(1))
	if e != nil {
		return nil, probe.NewError(e)
	}
	return t, nil
}

func mainSetTag(ctx *cli.Context) error {
	checkSetTagSyntax(ctx)
	setTagListColorScheme()

	objectURL := ctx.Args().Get(0)

	t, err := getTaggingMap(ctx)

	fatalIf(err.Trace(ctx.Args()...), "Unable to parse input tags, Please refer to mc "+ctx.Command.FullName()+" --help.")

	clnt, err := newClient(objectURL)

	fatalIf(err.Trace(objectURL), "Unable to initialize target "+objectURL)

	fatalIf(clnt.SetObjectTagging(t).Trace(objectURL), "Unable to set tags")

	printMsg(tagSetMessage{
		Status: "success",
		Name:   objectURL,
	})

	return nil
}
