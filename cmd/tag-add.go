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
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var tagAddCmd = cli.Command{
	Name:   "add",
	Usage:  "add tags for an object",
	Action: mainAddTag,
	Before: setGlobalsFromContext,
	Flags:  append(tagAddFlags, globalFlags...),
	CustomHelpTemplate: `Name:
	{{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
   Assign object tags (key,value) to target.

EXAMPLES:
  1. Assign the tags to an existing object.
     {{.Prompt}} {{.HelpName}} s3/testbucket/testobject --tags "key1=value1&key2=value2&key3=value3"

`,
}

var tagAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "tags",
		Usage: "format '<key1>=<value1>&<key2>=<value2>'; <key1>=<value1> is a key value pair, different key value pairs are separated by '&'",
	},
}

func checkAddTagSyntax(ctx *cli.Context) {
	tagValues := ctx.String("tags")
	if len(ctx.Args()) != 1 || len(tagValues) == 0 {
		cli.ShowCommandHelp(ctx, "add")
		os.Exit(globalErrorExitStatus)
	}
}

func getTaggingMap(ctx *cli.Context) (map[string]string, error) {
	tagKVMap := make(map[string]string)
	tagValues := strings.Split(ctx.String("tags"), "&")
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

func parseTagAddMessage(tags string, urlStr string, err error) tagsetListMessage {
	var t tagsetListMessage
	if err != nil {
		t.Status = "Failed to add tags to target " + urlStr + ". Error: " + err.Error()
	} else {
		t.Status = "Tags added for " + urlStr + "."
	}

	return t
}

func mainAddTag(ctx *cli.Context) error {
	checkAddTagSyntax(ctx)
	setTagListColorScheme()
	objectURL := ctx.Args().Get(0)
	var err error
	var pErr *probe.Error
	var objTagMap map[string]string
	if objTagMap, err = getTaggingMap(ctx); err != nil {
		console.Errorln(err.Error() + ". Key value parsing failed from arguments provided. Please refer to mc " + ctx.Command.FullName() + " --help for details.")
		return err
	}
	clnt, pErr := newClient(objectURL)
	fatalIf(pErr.Trace(objectURL), "Unable to initialize target "+objectURL+".")
	pErr = clnt.SetObjectTagging(objTagMap)
	fatalIf(pErr, "Failed to add tags")
	tagObj, err := getObjTagging(objectURL)
	var tMsg tagsetListMessage
	tMsg = parseTagAddMessage(tagObj.String(), objectURL, err)
	printMsg(tMsg)

	return nil
}
