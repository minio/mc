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
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var tagSetCmd = cli.Command{
	Name:   "set",
	Usage:  "set tags for an object",
	Action: mainSetTag,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET [TAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
   Assign object tags (key,value) to target.

EXAMPLES:
  1. Assign the tags to an existing object.
     {{.Prompt}} {{.HelpName}} s3/testbucket/testobject "key1=value1&key2=value2&key3=value3"

`,
}

// tagSetTagMessage structure will show message depending on the type of console.
type tagSetMessage struct {
	Status string `json:"status"`
	Name   string `json:"name"`
	Error  error  `json:"error,omitempty"`
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

func getTagSetMessage(tags string, urlStr string, err error) tagSetMessage {
	var t tagSetMessage
	t.Name = getTagObjectName(urlStr)
	if err != nil {
		t.Status = "error"
		t.Error = err
	} else {
		t.Status = "success"
	}
	return t
}

func checkSetTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 || (len(ctx.Args()) == 2 && len(ctx.Args().Get(1)) == 0) {
		cli.ShowCommandHelp(ctx, "set")
		os.Exit(globalErrorExitStatus)
	}
}

func getTaggingMap(ctx *cli.Context) (map[string]string, error) {
	if len(ctx.Args()) != 2 {
		return nil, errors.New("Tags argument is empty")
	}
	tagKVMap := make(map[string]string)
	tagValues := strings.Split(ctx.Args().Get(1), "&")
	for tagIdx, tag := range tagValues {
		var key, val string
		if !strings.Contains(tag, "=") {
			key = tag
			val = ""
		} else {
			key = splitStr(tag, "=", 2)[0]
			val = splitStr(tag, "=", 2)[1]
		}
		if key != "" {
			tagKVMap[key] = val
		} else {
			return nil, errors.New("error extracting tag argument(#" + strconv.Itoa(tagIdx+1) + ") " + tag)
		}
	}
	return tagKVMap, nil
}

func mainSetTag(ctx *cli.Context) error {
	checkSetTagSyntax(ctx)
	setTagListColorScheme()
	objectURL := ctx.Args().Get(0)
	var err error
	var pErr *probe.Error
	var objTagMap map[string]string
	var msg tagSetMessage

	if objTagMap, err = getTaggingMap(ctx); err != nil {
		fatalIf(probe.NewError(err), ". Key value parsing failed from arguments provided. Please refer to mc "+ctx.Command.FullName()+" --help for details.")
	}
	clnt, pErr := newClient(objectURL)
	fatalIf(pErr.Trace(objectURL), "Unable to initialize target "+objectURL+".")
	pErr = clnt.SetObjectTagging(objTagMap)
	fatalIf(pErr, "Failed to set tags")
	tagObj, err := getObjTagging(objectURL)
	msg = getTagSetMessage(tagObj.String(), objectURL, err)
	printMsg(msg)

	return nil
}
