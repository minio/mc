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
	"strings"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v6"
	"github.com/minio/minio/pkg/console"
)

var tagRemoveCmd = cli.Command{
	Name:   "remove",
	Usage:  "remove tags assigned to the object",
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
   Delete object tags previously assigned (if any) to target.

EXAMPLES:
  1. Delete the tag values to testobject in the bucket testbucket on alias s3.
     {{.Prompt}} {{.HelpName}} s3/testbucket/testobject

`,
}

func checkRemoveTagSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelp(ctx, "remove")
		os.Exit(globalErrorExitStatus)
	}
}

func mainRemoveTag(ctx *cli.Context) error {
	checkRemoveTagSyntax(ctx)
	setTagSetColorScheme()
	var err error
	var pErr *probe.Error
	objectURL := ctx.Args().Get(0)

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

	if err = s3c.api.RemoveObjectTagging(bucket, object); err != nil {
		// S3 returns this error even after Tags are removed successfully
		// Error not seen with minio server
		if !strings.Contains(err.Error(), "204 No Content") {
			console.Errorln(err.Error() + ". Unable to remove tags for " + objectURL)
			return err
		}
	}
	console.Infoln("Tag(s) removed for `" + bucket + slashSeperator + object + "`.")
	return nil
}
