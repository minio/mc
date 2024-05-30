// Copyright (c) 2015-2022 MinIO, Inc.
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
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/pkg/v3/console"
)

var replicateAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "arn",
		Usage:  "unique role ARN",
		Hidden: true,
	},
	cli.StringFlag{
		Name:  "id",
		Usage: "id for the rule, should be a unique value",
	},
	cli.StringFlag{
		Name:  "tags",
		Usage: "format '<key1>=<value1>&<key2>=<value2>&<key3>=<value3>', multiple values allowed for multiple key/value pairs",
	},
	cli.StringFlag{
		Name:  "storage-class",
		Usage: `storage class for destination, valid values are either "STANDARD" or "REDUCED_REDUNDANCY"`,
	},
	cli.BoolFlag{
		Name:  "disable",
		Usage: "disable the rule",
	},
	cli.IntFlag{
		Name:  "priority",
		Usage: "priority of the rule, should be unique and is a required field",
	},
	cli.StringFlag{
		Name:  "remote-bucket",
		Usage: "remote bucket, should be a unique value for the configuration",
	},
	cli.StringFlag{
		Name:  "replicate",
		Value: `delete-marker,delete,existing-objects,metadata-sync`,
		Usage: `comma separated list to enable replication of soft deletes, permanent deletes, existing objects and metadata sync`,
	},
	cli.StringFlag{
		Name:  "path",
		Value: "auto",
		Usage: "bucket path lookup supported by the server. Valid options are ['auto', 'on', 'off']'",
	},
	cli.StringFlag{
		Name:  "region",
		Usage: "region of the destination bucket (optional)",
	},
	cli.StringFlag{
		Name:  "bandwidth",
		Usage: "set bandwidth limit in bytes per second (K,B,G,T for metric and Ki,Bi,Gi,Ti for IEC units)",
	},
	cli.BoolFlag{
		Name:  "sync",
		Usage: "enable synchronous replication for this target. default is async",
	},
	cli.UintFlag{
		Name:  "healthcheck-seconds",
		Usage: "health check interval in seconds",
		Value: 60,
	},
	cli.BoolFlag{
		Name:  "disable-proxy",
		Usage: "disable proxying in active-active replication. If unset, default behavior is to proxy",
	},
}

var replicateAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a server side replication configuration rule",
	Action:       mainReplicateAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateAddFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add replication configuration rule on bucket "sourcebucket" for alias "sourceminio" with alias "targetminio" to replicate all operations in an active-active replication setup.
     {{.Prompt}} {{.HelpName}} sourceminio/sourcebucket --remote-bucket targetminio/targetbucket \
         --priority 1 

  2. Add replication configuration rule on bucket "mybucket" for alias "myminio" to replicate all operations in an active-active replication setup.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --remote-bucket https://foobar:foo12345@minio.siteb.example.com/targetbucket \
         --priority 1 

  3. Add replication configuration rule on bucket "mybucket" for alias "myminio" to replicate all objects with tags
     "key1=value1, key2=value2" to targetbucket synchronously with bandwidth set to 2 gigabits per second. 
     {{.Prompt}} {{.HelpName}} myminio/mybucket --remote-bucket https://foobar:foo12345@minio.siteb.example.com/targetbucket  \
         --tags "key1=value1&key2=value2" --bandwidth "2G" --sync \
         --priority 1

  4. Disable a replication configuration rule on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket --remote-bucket https://foobar:foo12345@minio.siteb.example.com/targetbucket  \
         --tags "key1=value1&key2=value2" \
         --priority 1 --disable

  5. Add replication configuration rule with existing object replication, delete marker replication and versioned deletes
     enabled on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket --remote-bucket https://foobar:foo12345@minio.siteb.example.com/targetbucket  \
         --replicate "existing-objects,delete,delete-marker" \
         --priority 1
`,
}

// checkReplicateAddSyntax - validate all the passed arguments
func checkReplicateAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
	if ctx.String("remote-bucket") == "" {
		fatal(errDummy().Trace(), "--remote-bucket flag needs to be specified.")
	}
}

type replicateAddMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	URL    string `json:"url"`
	ID     string `json:"id"`
}

const (
	enableStatus  = "enable"
	disableStatus = "disable"
)

func (l replicateAddMessage) JSON() string {
	l.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (l replicateAddMessage) String() string {
	if l.ID != "" {
		return console.Colorize("replicateAddMessage", "Replication configuration rule with ID `"+l.ID+"` applied to "+l.URL+".")
	}
	return console.Colorize("replicateAddMessage", "Replication configuration rule applied to "+l.URL+" successfully.")
}

func extractCredentialURL(argURL string) (accessKey, secretKey string, u *url.URL) {
	var parsedURL string
	if strings.HasPrefix(argURL, "http://") || strings.HasPrefix(argURL, "https://") {
		if hostKeyTokens.MatchString(argURL) {
			fatalIf(errInvalidArgument().Trace(argURL), "temporary tokens are not allowed for remote targets")
		}
		if hostKeys.MatchString(argURL) {
			parts := hostKeys.FindStringSubmatch(argURL)
			if len(parts) != 5 {
				fatalIf(errInvalidArgument().Trace(argURL), "unsupported remote target format, please check --help")
			}
			accessKey = parts[2]
			secretKey = parts[3]
			parsedURL = fmt.Sprintf("%s%s", parts[1], parts[4])
		}
	} else {
		var alias string
		var aliasCfg *aliasConfigV10
		// get alias config by alias url
		alias, parsedURL, aliasCfg = mustExpandAlias(argURL)
		if aliasCfg == nil {
			fatalIf(errInvalidAliasedURL(alias).Trace(argURL), "No such alias `"+alias+"` found.")
			return
		}
		accessKey, secretKey = aliasCfg.AccessKey, aliasCfg.SecretKey
	}
	var e error
	if parsedURL == "" {
		fatalIf(errInvalidArgument().Trace(argURL), "no valid credentials were detected")
	}
	u, e = url.Parse(parsedURL)
	if e != nil {
		fatalIf(errInvalidArgument().Trace(parsedURL), "unsupported URL format %v", e)
	}

	return accessKey, secretKey, u
}

// fetchRemoteTarget - returns the dest bucket, dest endpoint, access and secret key
func fetchRemoteTarget(cli *cli.Context) (bktTarget *madmin.BucketTarget) {
	if !cli.IsSet("remote-bucket") {
		fatalIf(probe.NewError(fmt.Errorf("missing Remote target configuration")), "unable to parse remote target")
	}
	p := cli.String("path")
	if !isValidPath(p) {
		fatalIf(errInvalidArgument().Trace(p),
			"unrecognized bucket path style. Valid options are `[on, off, auto]`.")
	}

	tgtURL := cli.String("remote-bucket")
	accessKey, secretKey, u := extractCredentialURL(tgtURL)
	var tgtBucket string
	if u.Path != "" {
		tgtBucket = path.Clean(u.Path[1:])
	}
	fatalIf(probe.NewError(s3utils.CheckValidBucketName(tgtBucket)).Trace(tgtURL), "invalid target bucket")

	bandwidthStr := cli.String("bandwidth")
	bandwidth, e := getBandwidthInBytes(bandwidthStr)
	fatalIf(probe.NewError(e).Trace(bandwidthStr), "invalid bandwidth value")

	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	creds := &madmin.Credentials{AccessKey: accessKey, SecretKey: secretKey}
	disableproxy := cli.Bool("disable-proxy")
	bktTarget = &madmin.BucketTarget{
		TargetBucket:        tgtBucket,
		Secure:              u.Scheme == "https",
		Credentials:         creds,
		Endpoint:            u.Host,
		Path:                p,
		API:                 "s3v4",
		Type:                madmin.ServiceType("replication"),
		Region:              cli.String("region"),
		BandwidthLimit:      int64(bandwidth),
		ReplicationSync:     cli.Bool("sync"),
		DisableProxy:        disableproxy,
		HealthCheckDuration: time.Duration(cli.Uint("healthcheck-seconds")) * time.Second,
	}
	return bktTarget
}

func getBandwidthInBytes(bandwidthStr string) (bandwidth uint64, err error) {
	if bandwidthStr != "" {
		bandwidth, err = humanize.ParseBytes(bandwidthStr)
		if err != nil {
			return
		}
	}
	return
}

func mainReplicateAdd(cliCtx *cli.Context) error {
	ctx, cancelReplicateAdd := context.WithCancel(globalContext)
	defer cancelReplicateAdd()

	console.SetColor("replicateAddMessage", color.New(color.FgGreen))

	checkReplicateAddSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)

	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "unable to initialize connection.")

	var sourceBucket string
	switch c := client.(type) {
	case *S3Client:
		sourceBucket, _ = c.url2BucketAndObject()
	default:
		fatalIf(err.Trace(args...), "replication is not supported for filesystem")
	}
	// Create a new MinIO Admin Client
	admclient, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "unable to initialize admin connection.")

	bktTarget := fetchRemoteTarget(cliCtx)
	arn, e := admclient.SetRemoteTarget(globalContext, sourceBucket, bktTarget)
	fatalIf(probe.NewError(e).Trace(args...), "unable to configure remote target")

	rcfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "unable to fetch replication configuration")

	ruleStatus := enableStatus
	if cliCtx.Bool(disableStatus) {
		ruleStatus = disableStatus
	}
	dmReplicateStatus := disableStatus
	deleteReplicationStatus := disableStatus
	replicaSync := enableStatus
	existingReplicationStatus := disableStatus
	replSlice := strings.Split(cliCtx.String("replicate"), ",")
	for _, opt := range replSlice {
		switch strings.TrimSpace(strings.ToLower(opt)) {
		case "delete-marker":
			dmReplicateStatus = enableStatus
		case "delete":
			deleteReplicationStatus = enableStatus
		case "metadata-sync", "replica-metadata-sync":
			replicaSync = enableStatus
		case "existing-objects":
			existingReplicationStatus = enableStatus
		default:
			fatalIf(probe.NewError(fmt.Errorf("invalid value for --replicate flag %s", cliCtx.String("replicate"))),
				`--replicate flag takes one or more comma separated string with values "delete", "delete-marker", "metadata-sync", "existing-objects" or "" to disable these settings`)
		}
	}

	opts := replication.Options{
		TagString:               cliCtx.String("tags"),
		StorageClass:            cliCtx.String("storage-class"),
		Priority:                strconv.Itoa(cliCtx.Int("priority")),
		RuleStatus:              ruleStatus,
		ID:                      cliCtx.String("id"),
		DestBucket:              arn,
		Op:                      replication.AddOption,
		ReplicateDeleteMarkers:  dmReplicateStatus,
		ReplicateDeletes:        deleteReplicationStatus,
		ReplicaSync:             replicaSync,
		ExistingObjectReplicate: existingReplicationStatus,
	}
	fatalIf(client.SetReplication(ctx, &rcfg, opts), "unable to add replication rule")

	printMsg(replicateAddMessage{
		Op:  cliCtx.Command.Name,
		URL: aliasedURL,
		ID:  opts.ID,
	})
	return nil
}
