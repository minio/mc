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
	"github.com/minio/mc/pkg/probe"
)

var tagRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove tags assigned to an object",
	Action: mainRemoveTag,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
   Remove object tags assigned to an object .

EXAMPLES:
  1. Remove the tags added to an existing object.
     {{.Prompt}} {{.HelpName}} s3/testbucket/testobject

`,
}

func checkRemoveTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "remove")
		os.Exit(globalErrorExitStatus)
	}
}

func parseTagRemoveMessage(tags string, urlStr string, err error) tagsetListMessage {
	var t tagsetListMessage
	if err != nil {
		t.Status = "Remove tags to target " + urlStr + ". Error " + err.Error()
	} else {
		t.Status = "Tags removed for " + urlStr + "."
	}
	return t
}

func mainRemoveTag(ctx *cli.Context) error {
	checkRemoveTagSyntax(ctx)
	setTagListColorScheme()
	var pErr *probe.Error
	objectURL := ctx.Args().Get(0)
	clnt, pErr := newClient(objectURL)
	fatalIf(pErr.Trace(objectURL), "Unable to initialize target "+objectURL+".")
	pErr = clnt.DeleteObjectTagging()
	fatalIf(pErr, "Failed to remove tags")
	tagObj, err := getObjTagging(objectURL)
	var tMsg tagsetListMessage
	tMsg = parseTagRemoveMessage(tagObj.String(), objectURL, err)
	printMsg(tMsg)

	return nil
}
