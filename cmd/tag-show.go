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
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	tagging "github.com/minio/minio/pkg/bucket/object/tagging"
	"github.com/minio/minio/pkg/console"
)

var tagShowCmd = cli.Command{
	Name:   "show",
	Usage:  "show tags for objects",
	Action: mainShowTag,
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
   Show tags assigned to an object.

EXAMPLES:
  1. Show the tags assigned to the object named testobject in the bucket testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} s3/testbucket/testobject
  2. Show the output in JSON format, the tags assigned to the object named testobject in the bucket testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} s3/testbucket/testobject

`,
}

type tagMessage struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// tagMessage container for displaying tag
type tagSetMessage struct {
	Tags []tagMessage `json:"tagset"`
}

func (t tagSetMessage) JSON() string {
	var tagJSONbytes []byte
	var err error

	tagJSONbytes, err = json.MarshalIndent(t, "", "  ")
	tagJSONbytes = bytes.Replace(tagJSONbytes, []byte("\\u0026"), []byte("&"), -1)
	fatalIf(probe.NewError(err), "Unable to marshal into JSON.")

	return string(tagJSONbytes)
}

// tagStr is in the format key1=value1&key1=value2
func parseTagMessage(tagsStr string, urlStr string) tagSetMessage {
	var t tagSetMessage
	var tagStr string
	tagStr = strings.Replace(tagsStr, "\\u0026", "&", -1)
	kvPairStr := strings.Split(tagStr, "&")

	for _, kvPair := range kvPairStr {
		kvPairSplit := splitStr(kvPair, "=", 2)
		t.Tags = append(t.Tags, tagMessage{Key: kvPairSplit[0], Value: kvPairSplit[1]})
	}

	return t
}

func showTagObjectName(urlStr string) {
	clnt, pErr := newClient(urlStr)

	if pErr != nil {
		console.Errorln(pErr.ToGoError().Error)
		fatalIf(probe.NewError(errors.New("Unable to show tags")), "Unable to obtain client from provided url "+urlStr)
	}
	s3c, ok := clnt.(*s3Client)
	if !ok {
		fatalIf(probe.NewError(errors.New("Unable to show tags")), "Unable to show tags for object in URL "+urlStr)
	}
	bucket, object := s3c.url2BucketAndObject()
	objectName := fmt.Sprintf("%-10s: %s", "Object", bucket+slashSeperator+object)
	console.Println(console.Colorize(tagMainHeader, objectName))
}

func getTagObj(urlStr string) (tagging.Tagging, error) {
	var err error

	clnt, pErr := newClient(urlStr)
	if pErr != nil {
		console.Errorln(pErr.ToGoError().Error)
		fatalIf(probe.NewError(errors.New("Unable to show tags")), "Unable to obtain client from provided url "+urlStr)
	}
	s3c, ok := clnt.(*s3Client)
	if !ok {
		fatalIf(probe.NewError(errors.New("Unable to show tags")), "Unable to show tags for object in URL "+urlStr)
	}

	bucketName, objectName := s3c.url2BucketAndObject()
	if bucketName == "" && objectName == "" {
		return tagging.Tagging{}, errors.New("Bucket name & Object name cannot be empty")
	}
	tagXML, err := s3c.api.GetObjectTagging(bucketName, objectName)
	if err != nil {
		return tagging.Tagging{}, err
	}
	var tagObj tagging.Tagging
	if err = xml.Unmarshal([]byte(tagXML), &tagObj); err != nil {
		console.Errorln(err.Error() + ", Unable to initialize Object Tags for display.")
		return tagging.Tagging{}, err
	}

	return tagObj, err
}

// Color scheme for tag display
func setTagShowColorScheme() {
	console.SetColor(tagRowTheme, color.New(color.FgWhite))
	console.SetColor(tagMainHeader, color.New(color.FgCyan))
	console.SetColor(tagResultsSuccess, color.New(color.Bold, color.FgGreen))
	console.SetColor(tagResultsFailure, color.New(color.Bold, color.FgRed))
}

func checkShowTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "show")
		os.Exit(globalErrorExitStatus)
	}
}

func showTagInfoFieldMultiple(kvpairs []tagging.Tag) {
	padLen := 0
	for _, kv := range kvpairs {
		if len(kv.Key) > padLen {
			padLen = len(kv.Key)
		}
	}
	padLen = getTagSpacePad(padLen)
	for idx := 0; idx < len(kvpairs); idx++ {
		displayField := fmt.Sprintf("    %-*s:    %s ", padLen, kvpairs[idx].Key, kvpairs[idx].Value)
		console.Println(console.Colorize(tagRowTheme, displayField))
	}
}

func getTagSpacePad(padLen int) int {
	if padLen%4 == 0 {
		return (padLen + 8)
	} else if padLen%4 != 0 && padLen%2 == 0 {
		return (padLen + 6)
	} else if padLen%4 != 0 && padLen%2 != 0 {
		if (padLen+1)%4 == 0 {
			return ((padLen + 1) + 8)
		} else if (padLen+3)%4 == 0 {
			return ((padLen + 3) + 4)
		}
	}
	return padLen
}

func mainShowTag(ctx *cli.Context) error {
	checkShowTagSyntax(ctx)
	setTagShowColorScheme()
	args := ctx.Args()
	objectURL := args.Get(0)
	tagObj, err := getTagObj(objectURL)

	if err != nil {
		console.Errorln(err.Error() + ". Error getting tag for " + objectURL)
		return err
	}

	if globalJSON {
		var tMsg tagSetMessage
		tMsg = parseTagMessage(tagObj.String(), objectURL)
		console.Println("\n" + tMsg.JSON())
		return nil
	}
	switch len(tagObj.TagSet.Tags) {
	case 0:
		console.Infoln("Tags not set or not available for " + objectURL)
	default:
		showTagObjectName(objectURL)
		showTagInfoFieldMultiple(tagObj.TagSet.Tags)
	}

	return nil
}
