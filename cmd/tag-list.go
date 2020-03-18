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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/bucket/object/tagging"
	"github.com/minio/minio/pkg/console"
)

var tagListCmd = cli.Command{
	Name:   "list",
	Usage:  "list tags for an object",
	Action: mainListTag,
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
   List tags assigned to an object.

EXAMPLES:
  1. List the tags assigned to an object.
     {{.Prompt}} {{.HelpName}} s3/testbucket/testobject
  2. List the tags assigned to an object in JSON format.
     {{.Prompt}} {{.HelpName}} --json s3/testbucket/testobject

`,
}

type tagList struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// tagsetListMessage container for displaying tag
type tagsetListMessage struct {
	Tags   []tagList `json:"tagset,omitempty"`
	Status string    `json:"status,omitempty"`
}

func (t tagsetListMessage) JSON() string {
	var tagJSONbytes []byte
	var err error

	tagJSONbytes, err = json.MarshalIndent(t, "", "  ")
	tagJSONbytes = bytes.Replace(tagJSONbytes, []byte("\\u0026"), []byte("&"), -1)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	return string(tagJSONbytes)
}

func (t tagsetListMessage) String() string {
	return console.Colorize(tagPrintMsgTheme, t.Status)
}

const (
	tagMainHeader    string = "Main-Heading"
	tagRowTheme      string = "Row-Header"
	tagPrintMsgTheme string = "Tag-PrintMsg"
)

// tagStr is in the format key1=value1&key1=value2
func parseTagListMessage(tags string, urlStr string) tagsetListMessage {
	var t tagsetListMessage
	var tagStr string
	var kvPairStr []string
	tagStr = strings.Replace(tags, "\\u0026", "&", -1)
	if tagStr != "" {
		kvPairStr = strings.SplitN(tagStr, "&", -1)
	}
	if len(kvPairStr) == 0 {
		t.Status = "Tags not added or not available."
	}

	for _, kvPair := range kvPairStr {
		kvPairSplit := splitStr(kvPair, "=", 2)
		t.Tags = append(t.Tags, tagList{Key: kvPairSplit[0], Value: kvPairSplit[1]})
	}

	return t
}

func getObjTagging(urlStr string) (tagging.Tagging, error) {
	clnt, pErr := newClient(urlStr)
	fatalIf(pErr.Trace(urlStr), "Unable to initialize target "+urlStr+".")
	tagObj, pErr := clnt.GetObjectTagging()
	fatalIf(pErr, "Failed to get tags for "+urlStr)

	return tagObj, nil
}

// Color scheme for tag display
func setTagListColorScheme() {
	console.SetColor(tagRowTheme, color.New(color.FgWhite))
	console.SetColor(tagMainHeader, color.New(color.Bold, color.FgCyan))
	console.SetColor(tagPrintMsgTheme, color.New(color.FgGreen))
}

func checkListTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "list")
		os.Exit(globalErrorExitStatus)
	}
}

func listTagInfoFieldMultiple(urlStr string, kvpairs []tagging.Tag) {
	var object string
	var splits []string
	padLen := len("Name")
	for _, kv := range kvpairs {
		if len(kv.Key) > padLen {
			padLen = len(kv.Key)
		}
	}
	padLen = listTagPaddingSpace(padLen)
	if !strings.Contains(urlStr, "/") {
		urlStr = filepath.ToSlash(urlStr)
	}
	splits = splitStr(urlStr, "/", 3)
	object = splits[2]
	objectName := fmt.Sprintf("%-*s:    %s", padLen, "Name", object)
	console.Println(console.Colorize(tagMainHeader, objectName))
	for idx := 0; idx < len(kvpairs); idx++ {
		displayField := fmt.Sprintf("%-*s:    %s", padLen, kvpairs[idx].Key, kvpairs[idx].Value)
		console.Println(console.Colorize(tagRowTheme, displayField))
	}
}

func listTagPaddingSpace(padLen int) int {
	switch padLen % 4 {
	case 0:
		padLen += 4
	case 1:
		padLen += 3
	case 2:
		padLen += 2
	case 3:
		padLen += 5
	}
	return padLen
}

func mainListTag(ctx *cli.Context) error {
	checkListTagSyntax(ctx)
	setTagListColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	tagObj, err := getObjTagging(objectURL)

	if err != nil {
		console.Errorln(err.Error() + ". Error getting tag for " + objectURL)
		return err
	}

	if globalJSON {
		var tMsg tagsetListMessage
		tMsg = parseTagListMessage(tagObj.String(), objectURL)
		printMsg(tMsg)
		return nil
	}
	switch len(tagObj.TagSet.Tags) {
	case 0:
		var tMsg tagsetListMessage
		tMsg = parseTagListMessage(tagObj.String(), objectURL)
		printMsg(tMsg)
	default:
		listTagInfoFieldMultiple(objectURL, tagObj.TagSet.Tags)
	}

	return nil
}
