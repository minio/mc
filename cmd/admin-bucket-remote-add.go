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
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/auth"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminBucketRemoteAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "path",
		Value: "auto",
		Usage: "bucket path lookup supported by the server. Valid options are '[on,off,auto]'",
	},
	cli.StringFlag{
		Name:  "service",
		Usage: "type of service. Valid options are '[replication]'",
	},
	cli.StringFlag{
		Name:  "region",
		Usage: "region of the destination bucket (optional)",
	},
}
var adminBucketRemoteAddCmd = cli.Command{
	Name:   "add",
	Usage:  "add a new remote target",
	Action: mainAdminBucketRemoteAdd,
	Before: setGlobalsFromContext,
	Flags:  append(globalFlags, adminBucketRemoteAddFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET http(s)://ACCESSKEY:SECRETKEY@DEST_URL/DEST_BUCKET [--path | --region ] --service

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
  1. Set a new remote replication target 'replicabucket' in region "us-west-1" on https://minio2:9000 for bucket 'srcbucket' on MinIO server.
     {{.DisableHistory}}
     {{.Prompt}} {{.HelpName}} myminio/srcbucket \
                 https://foobar:foo12345@minio2:9000/replicabucket \
                 --service "replication" --region "us-west-1"
     {{.EnableHistory}}
`,
}

// checkAdminBucketRemoteAddSyntax - validate all the passed arguments
func checkAdminBucketRemoteAddSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 2 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
	if argsNr > 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for remote add command.")
	}
}

// RemoteMessage container for content message structure
type RemoteMessage struct {
	op           string
	Status       string `json:"status"`
	AccessKey    string `json:"accessKey,omitempty"`
	SecretKey    string `json:"secretKey,omitempty"`
	SourceBucket string `json:"sourceBucket"`
	TargetURL    string `json:"TargetURL,omitempty"`
	TargetBucket string `json:"TargetBucket,omitempty"`
	RemoteARN    string `json:"RemoteARN,omitempty"`
	Path         string `json:"path,omitempty"`
	Region       string `json:"region,omitempty"`
	ServiceType  string `json:"service"`
}

func (r RemoteMessage) String() string {
	switch r.op {
	case "ls":
		message := console.Colorize("TargetURL", fmt.Sprintf("%s ", r.TargetURL))
		message += console.Colorize("SourceBucket", r.SourceBucket)
		message += console.Colorize("Arrow", "->")
		message += console.Colorize("TargetBucket", r.TargetBucket)
		message += " "
		message += console.Colorize("ARN", r.RemoteARN)
		return message
	case "rm":
		return console.Colorize("RemoteMessage", "Removed remote target for `"+r.SourceBucket+"` bucket successfully.")
	case "add":
		return console.Colorize("RemoteMessage", "Remote ARN = `"+r.RemoteARN+"`.")
	}
	return ""
}

// JSON returns jsonified message
func (r RemoteMessage) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

var targetKeys = regexp.MustCompile("^(https?://)(.*?):(.*?)@(.*?)/(.*?)$")

// fetchRemoteTarget - returns the dest bucket, dest endpoint, access and secret key
func fetchRemoteTarget(cli *cli.Context) (sourceBucket string, bktTarget *madmin.BucketTarget) {
	args := cli.Args()
	argCount := len(args)
	if argCount < 2 {
		fatalIf(probe.NewError(fmt.Errorf("Missing Remote target configuration")), "Unable to parse remote target")
	}
	_, sourceBucket = url2Alias(args[0])
	TargetURL := args[1]
	path := cli.String("path")
	if !isValidPath(path) {
		fatalIf(errInvalidArgument().Trace(path),
			"Unrecognized bucket path style. Valid options are `[on,off, auto]`.")
	}
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
	serviceType := cli.String("service")
	if !madmin.ServiceType(serviceType).IsValid() {
		fatalIf(errInvalidArgument().Trace(serviceType), "Invalid service type. Valid option is `[replication]`.")
	}

	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	creds := &auth.Credentials{AccessKey: accessKey, SecretKey: secretKey}
	bktTarget = &madmin.BucketTarget{
		TargetBucket: TargetBucket,
		Secure:       secure,
		Credentials:  creds,
		Endpoint:     host,
		Path:         path,
		API:          "s3v4",
		Type:         madmin.ServiceType(serviceType),
		Region:       cli.String("region"),
	}
	return sourceBucket, bktTarget
}

// mainAdminBucketRemoteAdd is the handle for "mc admin bucket remote set" command.
func mainAdminBucketRemoteAdd(ctx *cli.Context) error {
	checkAdminBucketRemoteAddSyntax(ctx)

	console.SetColor("RemoteMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	sourceBucket, bktTarget := fetchRemoteTarget(ctx)
	arn, e := client.SetRemoteTarget(globalContext, sourceBucket, bktTarget)
	if e != nil {
		fatalIf(probe.NewError(e).Trace(args...), "Unable to configure remote target")
	}

	printMsg(RemoteMessage{
		op:           ctx.Command.Name,
		TargetURL:    bktTarget.URL(),
		TargetBucket: bktTarget.TargetBucket,
		AccessKey:    bktTarget.Credentials.AccessKey,
		SourceBucket: sourceBucket,
		RemoteARN:    arn,
	})

	return nil
}
