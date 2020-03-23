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
	"errors"
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

const (
	tagMainHeader       string = "Main-Heading"
	tagRowTheme         string = "Row-Header"
	tagPrintMsgTheme    string = "Tag-PrintMsg"
	tagPrintErrMsgTheme string = "Tag-PrintMsgErr"
)

type tagList struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// tagListMessage structure for displaying tag
type tagListMessage struct {
	Tags   []tagList       `json:"tagset,omitempty"`
	Status string          `json:"status"`
	URL    string          `json:"url"`
	TagObj tagging.Tagging `json:"-"`
}

func (t tagListMessage) JSON() string {
	tagJSONbytes, err := json.MarshalIndent(t, "", "  ")
	tagJSONbytes = bytes.Replace(tagJSONbytes, []byte("\\u0026"), []byte("&"), -1)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON for "+getTagObjectName(t.URL))
	return string(tagJSONbytes)
}

func (t tagListMessage) String() string {
	return getFormattedTagList(getTagObjectName(t.URL), t.TagObj.TagSet.Tags)
}

// getTagListMessage parses the tags(string) and initializes the structure tagsetListMessage.
// tags(string) is in the format key1=value1&key1=value2
func getTagListMessage(tags tagging.Tagging, urlStr string) tagListMessage {
	var t tagListMessage
	var tagStr string
	var kvPairStr []string
	tagStr = strings.Replace(tags.String(), "\\u0026", "&", -1)
	t.URL = urlStr
	t.TagObj = tags
	if tagStr != "" {
		kvPairStr = strings.SplitN(tagStr, "&", -1)
		t.Status = "success"
	}
	for _, kvPair := range kvPairStr {
		kvPairSplit := splitStr(kvPair, "=", 2)
		t.Tags = append(t.Tags, tagList{Key: kvPairSplit[0], Value: kvPairSplit[1]})
	}

	return t
}

func getObjTagging(urlStr string) tagging.Tagging {
	clnt, pErr := newClient(urlStr)
	if pErr != nil {
		fatalIf(pErr.Trace(urlStr), "Unable to initialize target "+urlStr+". "+pErr.ToGoError().Error())
	}
	tagObj, pErr := clnt.GetObjectTagging()
	fatalIf(pErr, "Failed to get tags for "+urlStr)

	return tagObj
}

// Color scheme for tag display
func setTagListColorScheme() {
	console.SetColor(tagRowTheme, color.New(color.FgWhite))
	console.SetColor(tagMainHeader, color.New(color.Bold, color.FgCyan))
	console.SetColor(tagPrintMsgTheme, color.New(color.FgGreen))
	console.SetColor(tagPrintErrMsgTheme, color.New(color.FgRed))
}

func checkListTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "list")
		os.Exit(globalErrorExitStatus)
	}
}

func getTagObjectName(urlStr string) string {
	if !strings.Contains(urlStr, "/") {
		urlStr = filepath.ToSlash(urlStr)
	}
	splits := splitStr(urlStr, "/", 3)
	object := splits[2]

	return object
}

func getFormattedTagList(tagObjName string, kvpairs []tagging.Tag) string {
	var tagListInfo string
	padLen := len("Name")
	for _, kv := range kvpairs {
		if len(kv.Key) > padLen {
			padLen = len(kv.Key)
		}
	}
	padLen = listTagPaddingSpace(padLen)
	objectName := fmt.Sprintf("%-*s:    %s\n", padLen, "Name", tagObjName)
	tagListInfo += console.Colorize(tagMainHeader, objectName)
	for idx := 0; idx < len(kvpairs); idx++ {
		displayField := fmt.Sprintf("%-*s:    %s\n", padLen, kvpairs[idx].Key, kvpairs[idx].Value)
		tagListInfo += console.Colorize(tagRowTheme, displayField)
	}
	return tagListInfo
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
	tagObj := getObjTagging(objectURL)
	if len(tagObj.TagSet.Tags) == 0 {
		errorIf(probe.NewError(errors.New("Tag(s) not set for "+objectURL)), "Failed to get tags.")
		return exitStatus(globalErrorExitStatus)
	}
	var msg tagListMessage
	msg = getTagListMessage(tagObj, objectURL)
	printMsg(msg)
	return nil
}
