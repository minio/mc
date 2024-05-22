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
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/minio/pkg/v3/console"
)

var replicateUpdateFlags = []cli.Flag{
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
		Usage: `storage class for destination, valid values are ['STANDARD', 'REDUCED_REDUNDANCY']`,
	},
	cli.StringFlag{
		Name:  "state",
		Usage: "change rule status, valid values are ['enable', 'disable']",
	},
	cli.IntFlag{
		Name:  "priority",
		Usage: "priority of the rule, should be unique and is a required field",
	},
	cli.StringFlag{
		Name:  "remote-bucket",
		Usage: "destination bucket, should be a unique value for the configuration",
	},
	cli.StringFlag{
		Name:  "replicate",
		Usage: `comma separated list to enable replication of soft deletes, permanent deletes, existing objects and metadata sync. Valid options are "delete-marker","delete","existing-objects","metadata-sync" and ""'`,
	},
	cli.StringFlag{
		Name:  "sync",
		Usage: "enable synchronous replication for this target, valid values are ['enable', 'disable'].",
		Value: "disable",
	},
	cli.StringFlag{
		Name:  "proxy",
		Usage: "enable proxying in active-active replication, valid values are ['enable', 'disable']",
		Value: "enable",
	},
	cli.StringFlag{
		Name:  "bandwidth",
		Usage: "Set bandwidth limit in bytes per second (K,B,G,T for metric and Ki,Bi,Gi,Ti for IEC units)",
	},
	cli.UintFlag{
		Name:  "healthcheck-seconds",
		Usage: "health check duration in seconds",
		Value: 60,
	},
	cli.StringFlag{
		Name:  "path",
		Value: "auto",
		Usage: "bucket path lookup supported by the server, valid options are ['on', 'off', 'auto']",
	},
}

var replicateUpdateCmd = cli.Command{
	Name:          "update",
	Aliases:       []string{"edit"},
	HiddenAliases: true,
	Usage:         "modify an existing server side replication configuration rule",
	Action:        mainReplicateUpdate,
	OnUsageError:  onUsageError,
	Before:        setGlobalsFromContext,
	Flags:         append(globalFlags, replicateUpdateFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET --id=RULE-ID [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Change priority of rule with rule ID "bsibgh8t874dnjst8hkg" on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "bsibgh8t874dnjst8hkg"  --priority 3

  2. Disable a replication configuration rule with rule ID "bsibgh8t874dnjst8hkg" on target myminio/bucket
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "bsibgh8t874dnjst8hkg" --state disable

  3. Set tags and storage class on a replication configuration with rule ID "kMYD.491" on target myminio/bucket/prefix.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "kMYD.491" --tags "key1=value1&key2=value2" \
				  --storage-class "STANDARD" --priority 2
  4. Clear tags for replication configuration rule with ID "kMYD.491" on a target myminio/bucket.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "kMYD.491" --tags ""

  5. Enable delete marker replication on a replication configuration rule with ID "kxYD.491" on a target myminio/bucket.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "kxYD.491" --replicate "delete-marker"

  6. Disable delete marker and versioned delete replication on a replication configuration rule with ID "kxYD.491" on a target myminio/bucket.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "kxYD.491" --replicate ""

  7. Enable existing object replication on a configuration rule with ID "kxYD.491" on a target myminio/bucket. Rule previously had enabled delete marker and versioned delete replication.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "kxYD.491" --replicate "existing-objects,delete-marker,delete"

  8. Edit credentials for remote target with replication rule ID kxYD.491
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "kxYD.491" --remote-bucket  https://foobar:newpassword@minio.siteb.example.com/targetbucket
  
  9. Edit credentials with alias "targetminio" for remote target with replication rule ID kxYD.491
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "kxYD.491" --remote-bucket  targetminio/targetbucket

  10. Disable proxying and enable synchronous replication for remote target of bucket mybucket with rule ID kxYD.492
     {{.Prompt}} {{.HelpName}} myminio/mybucket --id "kxYD.492" --remote-bucket https://foobar:newpassword@minio.siteb.example.com/targetbucket \
         --sync "enable" --proxy "disable"
`,
}

// checkReplicateUpdateSyntax - validate all the passed arguments
func checkReplicateUpdateSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// modifyRemoteTarget - modifies the dest credentials or updates sync , disable-proxy settings
func modifyRemoteTarget(cli *cli.Context, targets []madmin.BucketTarget, arnStr string) (*madmin.BucketTarget, []madmin.TargetUpdateType) {
	args := cli.Args()
	foundIdx := -1
	for i, t := range targets {
		if t.Arn == arnStr {
			arn, e := madmin.ParseARN(arnStr)
			if e != nil {
				fatalIf(errInvalidArgument().Trace(args...), "Malformed ARN `"+arnStr+"` in replication config")
			}
			if arn.Bucket != t.TargetBucket {
				fatalIf(errInvalidArgument().Trace(args...), "Expected remote bucket %s, got %s for rule id %s", t.TargetBucket, arn.Bucket, cli.String("id"))
			}
			foundIdx = i
			break
		}
	}
	if foundIdx < 0 {
		fatalIf(errInvalidArgument().Trace(args...), "`"+arnStr+"` not found in replication config")
	}
	var ops []madmin.TargetUpdateType
	bktTarget := targets[foundIdx].Clone()
	if cli.IsSet("sync") {
		syncState := strings.ToLower(cli.String("sync"))
		switch syncState {
		case "enable", "disable":
			bktTarget.ReplicationSync = syncState == "enable"
			ops = append(ops, madmin.SyncUpdateType)
		default:
			fatalIf(errInvalidArgument().Trace(args...), "--sync can be either [enable|disable]")
		}
	}
	if cli.IsSet("proxy") {
		proxyState := strings.ToLower(cli.String("proxy"))
		switch proxyState {
		case "enable", "disable":
			bktTarget.DisableProxy = proxyState == "disable"
			ops = append(ops, madmin.ProxyUpdateType)

		default:
			fatalIf(errInvalidArgument().Trace(args...), "--proxy can be either [enable|disable]")
		}
	}

	if len(args) == 1 {
		_, sourceBucket := url2Alias(args[0])

		tgtURL := cli.String("remote-bucket")
		accessKey, secretKey, u := extractCredentialURL(tgtURL)
		var tgtBucket string
		if u.Path != "" {
			tgtBucket = path.Clean(u.Path[1:])
		}
		fatalIf(probe.NewError(s3utils.CheckValidBucketName(tgtBucket)).Trace(tgtURL), "invalid target bucket")

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
		creds := &madmin.Credentials{AccessKey: accessKey, SecretKey: secretKey}
		if tgtBucket != bktTarget.TargetBucket {
			fatalIf(errInvalidArgument().Trace(args...), "configured remote target bucket `"+tgtBucket+"` does not match "+bktTarget.TargetBucket+"` for this ARN `"+bktTarget.Arn+"`")
		}
		if sourceBucket != bktTarget.SourceBucket {
			fatalIf(errInvalidArgument().Trace(args...), "configured source bucket `"+sourceBucket+"` does not match "+bktTarget.SourceBucket+"` for this ARN `"+bktTarget.Arn+"`")
		}
		bktTarget.TargetBucket = tgtBucket
		bktTarget.Secure = secure
		bktTarget.Credentials = creds
		bktTarget.Endpoint = host
		ops = append(ops, madmin.CredentialsUpdateType)
	}
	if cli.IsSet("bandwidth") {
		bandwidthStr := cli.String("bandwidth")
		bandwidth, e := getBandwidthInBytes(bandwidthStr)
		fatalIf(probe.NewError(e).Trace(bandwidthStr), "invalid bandwidth value")

		bktTarget.BandwidthLimit = int64(bandwidth)
		ops = append(ops, madmin.BandwidthLimitUpdateType)
	}
	if cli.IsSet("healthcheck-seconds") {
		bktTarget.HealthCheckDuration = time.Duration(cli.Uint("healthcheck-seconds")) * time.Second
		ops = append(ops, madmin.HealthCheckDurationUpdateType)
	}
	if cli.IsSet("path") {
		bktTarget.Path = cli.String("path")
		ops = append(ops, madmin.PathUpdateType)
	}
	return &bktTarget, ops
}

type replicateUpdateMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	URL    string `json:"url"`
	ID     string `json:"id"`
}

func (l replicateUpdateMessage) JSON() string {
	l.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (l replicateUpdateMessage) String() string {
	if l.ID != "" {
		return console.Colorize("replicateUpdateMessage", "Replication configuration rule with ID `"+l.ID+"` applied to "+l.URL+".")
	}
	return console.Colorize("replicateUpdateMessage", "Replication configuration rule applied to "+l.URL+" successfully.")
}

func mainReplicateUpdate(cliCtx *cli.Context) error {
	ctx, cancelReplicateUpdate := context.WithCancel(globalContext)
	defer cancelReplicateUpdate()

	console.SetColor("replicateUpdateMessage", color.New(color.FgGreen))

	checkReplicateUpdateSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "unable to initialize connection.")
	rcfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "unable to get replication configuration")

	if !cliCtx.IsSet("id") {
		fatalIf(errInvalidArgument(), "--id is a required flag")
	}
	var state string
	if cliCtx.IsSet("state") {
		state = strings.ToLower(cliCtx.String("state"))
		if state != "enable" && state != "disable" {
			fatalIf(err.Trace(args...), "--state can be either `enable` or `disable`")
		}
	}
	var sourceBucket string
	switch c := client.(type) {
	case *S3Client:
		sourceBucket, _ = c.url2BucketAndObject()
	default:
		fatalIf(err.Trace(args...), "replication is not supported for filesystem")
	}
	// Create a new MinIO Admin Client
	admClient, err := newAdminClient(aliasedURL)
	fatalIf(err, "unable to initialize admin connection.")

	targets, e := admClient.ListRemoteTargets(globalContext, sourceBucket, "")
	fatalIf(probe.NewError(e).Trace(args...), "unable to fetch remote target.")

	var arn string
	for _, rule := range rcfg.Rules {
		if rule.ID == cliCtx.String("id") {
			arn = rule.Destination.Bucket
			if rcfg.Role != "" {
				arn = rcfg.Role
			}
			break
		}
	}
	if cliCtx.IsSet("remote-bucket") {
		bktTarget, ops := modifyRemoteTarget(cliCtx, targets, arn)
		_, e = admClient.UpdateRemoteTarget(globalContext, bktTarget, ops...)
		if e != nil {
			fatalIf(probe.NewError(e).Trace(args...), "Unable to update remote target `"+bktTarget.Endpoint+"` from `"+bktTarget.SourceBucket+"` -> `"+bktTarget.TargetBucket+"`")
		}
	} else {
		if cliCtx.IsSet("sync") || cliCtx.IsSet("bandwidth") || cliCtx.IsSet("proxy") || cliCtx.IsSet("healthcheck-seconds") || cliCtx.IsSet("path") {
			fatalIf(errInvalidArgument().Trace(args...), "--remote-bucket is a required flag`")
		}
	}

	var vDeleteReplicate, dmReplicate, replicasync, existingReplState string
	if cliCtx.IsSet("replicate") {
		replSlice := strings.Split(cliCtx.String("replicate"), ",")
		vDeleteReplicate = disableStatus
		dmReplicate = disableStatus
		replicasync = disableStatus
		existingReplState = disableStatus

		for _, opt := range replSlice {
			switch strings.TrimSpace(strings.ToLower(opt)) {
			case "delete-marker":
				dmReplicate = enableStatus
			case "delete":
				vDeleteReplicate = enableStatus
			case "metadata-sync", "replica-metadata-sync":
				replicasync = enableStatus
			case "existing-objects":
				existingReplState = enableStatus
			default:
				if opt != "" {
					fatalIf(probe.NewError(fmt.Errorf("invalid value for --replicate flag %s", cliCtx.String("replicate"))),
						`--replicate flag takes one or more comma separated string with values "delete", "delete-marker", "metadata-sync", "existing-objects" or "" to disable these settings`)
				}
			}
		}
	}

	opts := replication.Options{
		TagString:    cliCtx.String("tags"),
		RoleArn:      cliCtx.String("arn"),
		StorageClass: cliCtx.String("storage-class"),
		RuleStatus:   state,
		ID:           cliCtx.String("id"),
		Op:           replication.SetOption,
		DestBucket:   arn,
		IsSCSet:      cliCtx.IsSet("storage-class"),
		IsTagSet:     cliCtx.IsSet("tags"),
	}

	if cliCtx.IsSet("priority") {
		opts.Priority = strconv.Itoa(cliCtx.Int("priority"))
	}
	if cliCtx.IsSet("replicate") {
		opts.ReplicateDeletes = vDeleteReplicate
		opts.ReplicateDeleteMarkers = dmReplicate
		opts.ReplicaSync = replicasync
		opts.ExistingObjectReplicate = existingReplState
	}

	fatalIf(client.SetReplication(ctx, &rcfg, opts), "unable to modify replication rule")
	printMsg(replicateUpdateMessage{
		Op:  cliCtx.Command.Name,
		URL: aliasedURL,
		ID:  opts.ID,
	})
	return nil
}
