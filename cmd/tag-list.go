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
	"github.com/minio/minio-go/v6/pkg/tags"
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
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. List the tags assigned to an object.
     {{.Prompt}} {{.HelpName}} myminio/testbucket/testobject

  2. List the tags assigned to an object in JSON format.
     {{.Prompt}} {{.HelpName}} --json myminio/testbucket/testobject
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
	Tags   []tagList  `json:"tagset,omitempty"`
	Status string     `json:"status"`
	URL    string     `json:"url"`
	TagObj *tags.Tags `json:"-"`
}

func (t tagListMessage) JSON() string {
	tagJSONbytes, err := json.MarshalIndent(t, "", "  ")
	tagJSONbytes = bytes.Replace(tagJSONbytes, []byte("\\u0026"), []byte("&"), -1)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON for "+getTagObjectName(t.URL))
	return string(tagJSONbytes)
}

func (t tagListMessage) String() string {
	return getFormattedTagList(getTagObjectName(t.URL), t.TagObj.ToMap())
}

// getTagListMessage parses the tags(string) and initializes the structure tagsetListMessage.
// tags(string) is in the format key1=value1&key1=value2
func getTagListMessage(t *tags.Tags, urlStr string) tagListMessage {
	var tm = tagListMessage{URL: urlStr}
	if t == nil {
		tm.Status = "failure"
		return tm
	}
	tm.TagObj = t
	tm.Status = "success"
	for k, v := range t.ToMap() {
		tm.Tags = append(tm.Tags, tagList{Key: k, Value: v})
	}
	return tm
}

func mustGetObjectTagging(urlStr string) *tags.Tags {
	clnt, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize target")

	t, err := clnt.GetObjectTagging()
	fatalIf(err.Trace(urlStr), "Unable to fetch tags for "+urlStr)

	return t
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

func getFormattedTagList(tagObjName string, kvMap map[string]string) string {
	var tagListInfo string
	padLen := len("Name")
	for k := range kvMap {
		if len(k) > padLen {
			padLen = len(k)
		}
	}
	padLen = listTagPaddingSpace(padLen)
	objectName := fmt.Sprintf("%-*s:    %s\n", padLen, "Name", tagObjName)
	tagListInfo += console.Colorize(tagMainHeader, objectName)
	for k, v := range kvMap {
		displayField := fmt.Sprintf("%-*s:    %s\n", padLen, k, v)
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

	urlStr := ctx.Args().Get(0)
	tagObj := mustGetObjectTagging(urlStr)
	if tagObj == nil {
		return exitStatus(globalErrorExitStatus)
	}
	printMsg(getTagListMessage(tagObj, urlStr))
	return nil
}
