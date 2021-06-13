// Copyright (c) 2015-2021 MinIO, Inc.
//
// This file is part of MinIO Object Storage stack
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"net/url"
	"path"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/pkg/console"
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
	cli.StringFlag{
		Name:  "bandwidth",
		Usage: "Set bandwidth limit in bits per second (K,B,G,T for metric and Ki,Bi,Gi,Ti for IEC units)",
	},
	cli.BoolFlag{
		Name:  "sync",
		Usage: "enable synchronous replication for this target. Default is async",
	},
	cli.UintFlag{
		Name:  "healthcheck-seconds",
		Usage: "health check duration in seconds",
		Value: 60,
	},
	cli.BoolFlag{
		Name:  "disable-proxy",
		Usage: "disable proxying in active-active replication. If unset, default behavior is to proxy",
	},
}
var adminBucketRemoteAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a new remote target",
	Action:       mainAdminBucketRemoteAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminBucketRemoteAddFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET http(s)://ACCESSKEY:SECRETKEY@DEST_URL/DEST_BUCKET [--path | --region | --bandwidth] --service

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
  1. Set a new remote replication target "targetbucket" in region "us-west-1" on https://minio.siteb.example.com for bucket 'sourcebucket'.
     {{.Prompt}} {{.HelpName}} sitea/sourcebucket https://foobar:foo12345@minio.siteb.example.com/targetbucket \
         --service "replication" --region "us-west-1"

  2. Set a new remote replication target 'targetbucket' in region "us-west-1" on https://minio.siteb.example.com for
	 bucket 'sourcebucket' with bandwidth set to 2 gigabits per second. Enable synchronous replication to the target
	 and perform health check of target every 100 seconds
     {{.Prompt}} {{.HelpName}} sitea/sourcebucket https://foobar:foo12345@minio.siteb.example.com/targetbucket \
         --service "replication" --region "us-west-1 --bandwidth "2G" --sync
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
	op                  string
	Status              string        `json:"status"`
	AccessKey           string        `json:"accessKey,omitempty"`
	SecretKey           string        `json:"secretKey,omitempty"`
	SourceBucket        string        `json:"sourceBucket"`
	TargetURL           string        `json:"TargetURL,omitempty"`
	TargetBucket        string        `json:"TargetBucket,omitempty"`
	RemoteARN           string        `json:"RemoteARN,omitempty"`
	Path                string        `json:"path,omitempty"`
	Region              string        `json:"region,omitempty"`
	ServiceType         string        `json:"service"`
	Bandwidth           int64         `json:"bandwidth"`
	ReplicationSync     bool          `json:"replicationSync"`
	Proxy               bool          `json:"proxy"`
	HealthCheckDuration time.Duration `json:"healthcheckDuration"`
	ResetID             string        `json:"resetID"`
	ResetBefore         time.Time     `json:"resetBeforeDate"`
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
		syncStr := "    "
		if r.ReplicationSync && r.ServiceType == string(madmin.ReplicationService) {
			syncStr = "sync"
		}
		message += " " + console.Colorize("SyncLabel", syncStr)
		proxyStr := "     "
		if r.Proxy && r.ServiceType == string(madmin.ReplicationService) {
			proxyStr = "proxy"
		}
		message += " "
		message += console.Colorize("ProxyLabel", proxyStr)
		return message
	case "rm":
		return console.Colorize("RemoteMessage", "Removed remote target for `"+r.SourceBucket+"` bucket successfully.")
	case "add":
		return console.Colorize("RemoteMessage", "Remote ARN = `"+r.RemoteARN+"`.")
	case "edit":
		return console.Colorize("RemoteMessage", "Remote target updated successfully for target with ARN:`"+r.RemoteARN+"`.")
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

func extractCredentialURL(argURL string) (accessKey, secretKey string, u *url.URL) {
	var parsedURL string
	if hostKeyTokens.MatchString(argURL) {
		fatalIf(errInvalidArgument().Trace(argURL), "temporary tokens are not allowed for remote targets")
	}
	if hostKeys.MatchString(argURL) {
		parts := hostKeys.FindStringSubmatch(argURL)
		if len(parts) != 5 {
			fatalIf(errInvalidArgument().Trace(argURL), "Unsupported remote target format, please check --help")
		}
		accessKey = parts[2]
		secretKey = parts[3]
		parsedURL = fmt.Sprintf("%s%s", parts[1], parts[4])
	}
	var e error
	if parsedURL == "" {
		fatalIf(errInvalidArgument().Trace(argURL), "No valid credentials were detected")
	}
	u, e = url.Parse(parsedURL)
	if e != nil {
		fatalIf(errInvalidArgument().Trace(parsedURL), "Unsupported URL format %v", e)
	}

	return accessKey, secretKey, u
}

// fetchRemoteTarget - returns the dest bucket, dest endpoint, access and secret key
func fetchRemoteTarget(cli *cli.Context) (sourceBucket string, bktTarget *madmin.BucketTarget) {
	args := cli.Args()
	argCount := len(args)
	if argCount < 2 {
		fatalIf(probe.NewError(fmt.Errorf("Missing Remote target configuration")), "Unable to parse remote target")
	}
	_, sourceBucket = url2Alias(args[0])
	p := cli.String("path")
	if !isValidPath(p) {
		fatalIf(errInvalidArgument().Trace(p),
			"Unrecognized bucket path style. Valid options are `[on,off, auto]`.")
	}

	tgtURL := args[1]
	accessKey, secretKey, u := extractCredentialURL(tgtURL)
	var tgtBucket string
	if u.Path != "" {
		tgtBucket = path.Clean(u.Path[1:])
	}
	if e := s3utils.CheckValidBucketName(tgtBucket); e != nil {
		fatalIf(probe.NewError(e).Trace(tgtURL), "Invalid target bucket specified")
	}

	serviceType := cli.String("service")
	if !madmin.ServiceType(serviceType).IsValid() {
		fatalIf(errInvalidArgument().Trace(serviceType), "Invalid service type. Valid option is `[replication]`.")
	}
	if cli.IsSet("sync") && serviceType != string(madmin.ReplicationService) {
		fatalIf(errInvalidArgument(), "Invalid usage. --sync flag applies only to replication service")
	}
	bandwidthStr := cli.String("bandwidth")
	bandwidth, err := getBandwidthInBytes(bandwidthStr)
	if err != nil {
		fatalIf(errInvalidArgument().Trace(bandwidthStr), "Invalid bandwidth number")
	}
	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	creds := &madmin.Credentials{AccessKey: accessKey, SecretKey: secretKey}
	proxy := cli.Bool("proxy")
	bktTarget = &madmin.BucketTarget{
		TargetBucket:        tgtBucket,
		Secure:              u.Scheme == "https",
		Credentials:         creds,
		Endpoint:            u.Host,
		Path:                p,
		API:                 "s3v4",
		Type:                madmin.ServiceType(serviceType),
		Region:              cli.String("region"),
		BandwidthLimit:      int64(bandwidth),
		ReplicationSync:     cli.Bool("sync"),
		DisableProxy:        !proxy,
		HealthCheckDuration: time.Duration(cli.Uint("healthcheck-seconds")) * time.Second,
	}
	return sourceBucket, bktTarget
}

func getBandwidthInBytes(bandwidthStr string) (bandwidth uint64, err error) {
	if bandwidthStr != "" {
		bandwidth, err = humanize.ParseBytes(bandwidthStr)
		if err != nil {
			return
		}
	}
	bandwidth = bandwidth / 8
	return
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
		op:              ctx.Command.Name,
		TargetURL:       bktTarget.URL().String(),
		TargetBucket:    bktTarget.TargetBucket,
		AccessKey:       bktTarget.Credentials.AccessKey,
		SourceBucket:    sourceBucket,
		RemoteARN:       arn,
		ReplicationSync: bktTarget.ReplicationSync,
		Proxy:           !bktTarget.DisableProxy,
	})

	return nil
}
