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
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/console"
)

var replicateEditFlags = []cli.Flag{
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
		Usage: "storage class for destination (STANDARD_IA,REDUCED_REDUNDANCY etc)",
	},
	cli.StringFlag{
		Name:  "state",
		Usage: "change rule status. Valid values are [enable|disable]",
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
		Usage: "comma separated list to enable replication of delete markers, deletion of versioned objects and syncing replica metadata modifications.Valid options are \"delete-marker\", \"delete\",\"replica-metadata-sync\", \"existing-objects\" and \"\"",
	},
}

var replicateEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "modify an existing server side replication configuration rule",
	Action:       mainReplicateEdit,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateEditFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
   
USAGE:
  {{.HelpName}} TARGET
	   
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
`,
}

// checkReplicateEditSyntax - validate all the passed arguments
func checkReplicateEditSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "edit", 1) // last argument is exit code
	}
}

type replicateEditMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	URL    string `json:"url"`
	ID     string `json:"id"`
}

func (l replicateEditMessage) JSON() string {
	l.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (l replicateEditMessage) String() string {
	if l.ID != "" {
		return console.Colorize("replicateEditMessage", "Replication configuration rule with ID `"+l.ID+"` applied to "+l.URL+".")
	}
	return console.Colorize("replicateEditMessage", "Replication configuration rule applied to "+l.URL+" successfully.")
}

func mainReplicateEdit(cliCtx *cli.Context) error {
	ctx, cancelReplicateEdit := context.WithCancel(globalContext)
	defer cancelReplicateEdit()

	console.SetColor("replicateEditMessage", color.New(color.FgGreen))

	checkReplicateEditSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	rcfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication configuration")

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
			case "replica-metadata-sync":
				replicasync = enableStatus
			case "existing-objects":
				existingReplState = enableStatus
			default:
				if opt != "" {
					fatalIf(probe.NewError(fmt.Errorf("invalid value for --replicate flag %s", cliCtx.String("replicate"))), "--replicate flag takes one or more comma separated string with values \"delete, delete-marker, replica-metadata-sync\" or \"\"")
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
		DestBucket:   cliCtx.String("remote-bucket"),
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

	fatalIf(client.SetReplication(ctx, &rcfg, opts), "Could not modify replication rule")
	printMsg(replicateEditMessage{
		Op:  "set",
		URL: aliasedURL,
		ID:  opts.ID,
	})
	return nil
}
