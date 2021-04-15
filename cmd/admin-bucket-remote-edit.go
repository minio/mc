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
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/auth"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminBucketRemoteEditFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "arn",
		Usage: "ARN of target",
	}}
var adminBucketRemoteEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "edit credentials for existing remote target",
	Action:       mainAdminBucketRemoteEdit,
	Before:       setGlobalsFromContext,
	OnUsageError: onUsageError,
	Flags:        append(globalFlags, adminBucketRemoteEditFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET http(s)://ACCESSKEY:SECRETKEY@DEST_URL/DEST_BUCKET --arn arn

TARGET:
  Also called as alias/sourcebucketname

DEST_BUCKET:
  Also called as remote target bucket.

DEST_URL:
  Also called as remote endpoint.

ACCESSKEY:
  Also called as username.

SECRETKEY:
  Also called as password.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Edit credentials for existing remote target with arn where a remote target has been configured between sourcebucket on sitea to targetbucket on siteb.
  	{{.DisableHistory}}
  	{{.Prompt}} {{.HelpName}} sitea/sourcebucket \
                 https://foobar:newpassword@minio.siteb.example.com/targetbucket \
                 --arn "arn:minio:replication:us-west-1:993bc6b6-accd-45e3-884f-5f3e652aed2a:dest1"
	{{.EnableHistory}}
`,
}

// checkAdminBucketRemoteEditSyntax - validate all the passed arguments
func checkAdminBucketRemoteEditSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr != 2 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
}

// fetchRemoteEditTarget - returns the dest bucket, dest endpoint, access and secret key
func fetchRemoteEditTarget(cli *cli.Context) (bktTarget *madmin.BucketTarget) {
	args := cli.Args()
	_, sourceBucket := url2Alias(args[0])
	TargetURL := args[1]
	parts := targetKeys.FindStringSubmatch(TargetURL)
	if len(parts) != 6 {
		fatalIf(probe.NewError(fmt.Errorf("invalid url format")), "Malformed Remote target URL")
	}
	accessKey := parts[2]
	secretKey := parts[3]
	parsedURL := fmt.Sprintf("%s%s", parts[1], parts[4])
	TargetBucket := strings.TrimSuffix(parts[5], slashSeperator)
	TargetBucket = strings.TrimPrefix(TargetBucket, slashSeperator)
	u, cerr := url.Parse(parsedURL)
	if cerr != nil {
		fatalIf(probe.NewError(cerr), "Malformed Remote target URL")
	}
	secure := u.Scheme == "https"
	host := u.Host
	if u.Port() == "" {
		port := 80
		if secure {
			port = 443
		}
		host = host + ":" + strconv.Itoa(port)
	}
	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	creds := &auth.Credentials{AccessKey: accessKey, SecretKey: secretKey}
	bktTarget = &madmin.BucketTarget{
		SourceBucket: sourceBucket,
		TargetBucket: TargetBucket,
		Secure:       secure,
		Credentials:  creds,
		Endpoint:     host,
		API:          "s3v4",
		Region:       cli.String("region"),
		Arn:          cli.String("arn"),
	}
	return bktTarget
}

// mainAdminBucketRemoteEdit is the handle for "mc admin bucket remote edit" command.
func mainAdminBucketRemoteEdit(ctx *cli.Context) error {
	checkAdminBucketRemoteEditSyntax(ctx)

	console.SetColor("RemoteMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")
	bktTarget := fetchRemoteEditTarget(ctx)

	targets, e := client.ListRemoteTargets(globalContext, bktTarget.SourceBucket, "")
	fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch remote target.")

	var found bool
	for _, t := range targets {
		if t.Arn == bktTarget.Arn {
			if t.Endpoint != bktTarget.Endpoint {
				fatalIf(errInvalidArgument().Trace(args...), "configured Endpoint `"+t.Endpoint+"` does not match "+bktTarget.Endpoint+"` for this ARN `"+bktTarget.Arn+"`")
			}
			if t.TargetBucket != bktTarget.TargetBucket {
				fatalIf(errInvalidArgument().Trace(args...), "configured remote target bucket `"+t.TargetBucket+"` does not match "+bktTarget.TargetBucket+"` for this ARN `"+bktTarget.Arn+"`")
			}
			if t.SourceBucket != bktTarget.SourceBucket {
				fatalIf(errInvalidArgument().Trace(args...), "configured source bucket `"+t.SourceBucket+"` does not match "+bktTarget.SourceBucket+"` for this ARN `"+bktTarget.Arn+"`")
			}
			found = true
			break
		}
	}
	if !found {
		fatalIf(errInvalidArgument().Trace(args...), "Unable to edit remote target - `"+bktTarget.Arn+"` is not a valid Arn")
	}
	arn, e := client.UpdateRemoteTarget(globalContext, bktTarget)
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to update credentials for remote target `"+bktTarget.Endpoint+"` from `"+bktTarget.SourceBucket+"` -> `"+bktTarget.TargetBucket+"`")
	}

	printMsg(RemoteMessage{
		op:           ctx.Command.Name,
		TargetURL:    bktTarget.URL().String(),
		TargetBucket: bktTarget.TargetBucket,
		AccessKey:    bktTarget.Credentials.AccessKey,
		SourceBucket: bktTarget.SourceBucket,
		RemoteARN:    arn,
	})

	return nil
}
