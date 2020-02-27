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

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio/pkg/console"
)

var tagSetCmd = cli.Command{
	Name:   "set",
	Usage:  "set/configure tags for objects",
	Action: mainSetTag,
	Before: setGlobalsFromContext,
	Flags:  append(tagSetFlags, globalFlags...),
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
  1. Assign the tag values to testobject in the bucket testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} s3/testbucket/testobject --tags "key1:value1" --tags "key2:value2" --tags "key3:value3"

`,
}

var tagSetFlags = []cli.Flag{
	cli.StringSliceFlag{
		Name:  "tags",
		Usage: "format '<key>:<value>'; multiple --tags flag allowed for multiple key/value pairs",
	},
}

// Color scheme for set tag results
func setTagSetColorScheme() {
	console.SetColor(tagResultsSuccess, color.New(color.Bold, color.FgGreen))
	console.SetColor(tagResultsFailure, color.New(color.Bold, color.FgRed))
}

func checkSetTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "set")
		os.Exit(globalErrorExitStatus)
	}
}

func getTagMap(ctx *cli.Context) (map[string]string, error) {
	ilmTagKVMap := make(map[string]string)
	tagValues := ctx.StringSlice(strings.ToLower("tags"))
	for tagIdx, tag := range tagValues {
		if !strings.Contains(tag, ":") {
			return nil, errors.New("Tag argument(#" + strconv.Itoa(tagIdx+1) + ") `" + tag + "` not in `key:value` format")
		}
		key := splitStr(tag, ":", 2)[0]
		val := splitStr(tag, ":", 2)[1]
		if key != "" && val != "" || key != "" && val == "" {
			ilmTagKVMap[key] = val
		} else {
			return nil, errors.New("error extracting tag argument(#" + strconv.Itoa(tagIdx+1) + ") " + tag)
		}
	}
	return ilmTagKVMap, nil
}

func mainSetTag(ctx *cli.Context) error {
	checkSetTagSyntax(ctx)
	setTagSetColorScheme()

	objectURL := ctx.Args().Get(0)
	var err error
	var pErr *probe.Error
	var objTagMap map[string]string
	if objTagMap, err = getTagMap(ctx); err != nil {
		console.Errorln(err.Error() + ". Key value parsing failed from arguments provided. Please refer to mc " + ctx.Command.FullName() + " --help for details.")
		return err
	}
	alias, _ := url2Alias(objectURL)
	if alias == "" {
		fatalIf(errInvalidAliasedURL(objectURL), "Unable to set tags to target "+objectURL)
	}
	clnt, pErr := newClient(objectURL)
	if pErr != nil {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "Cannot parse the provided url "+objectURL)
	}
	s3c, ok := clnt.(*s3Client)
	if !ok {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), "For "+objectURL+" unable to obtain client reference.")
	}
	bucket, object := s3c.url2BucketAndObject()
	opts := minio.StatObjectOptions{}
	_, pErr = s3c.getObjectStat(bucket, object, opts)
	if pErr != nil {
		fatalIf(probe.NewError(errors.New("Unable to set tags")), objectURL+" unable to access object. Error: "+pErr.ToGoError().Error())
	}
	if err = s3c.api.PutObjectTagging(bucket, object, objTagMap); err != nil {
		console.Errorln(err.Error() + ". Unable to set tags for " + objectURL)
		return err
	}
	console.Infoln("Tag set for `" + bucket + slashSeperator + object + "`.")
	return nil
}
