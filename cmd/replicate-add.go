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

var replicateAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "arn",
		Usage: "unique role ARN",
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
		Usage: "storage class for destination (STANDARD_IA,REDUCED_REDUNDANCY etc)",
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
		Usage: "comma separated list to enable replication of delete markers, deletion of versioned objects and replica metadata sync(in the case of active-active replication).Valid options are \"delete-marker\", \"delete\" ,\"replica-metadata-sync\", \"existing-objects\" and \"\"",
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
 1. Add replication configuration rule on bucket "mybucket" for alias "myminio" to replicate all objects with tags
    "key1=value1, key2=value2" to destbucket, including delete markers and versioned deletes.
    {{.Prompt}} {{.HelpName}} myminio/mybucket/prefix --tags "key1=value1&key2=value2" \
         --storage-class "STANDARD" \
         --arn 'arn:minio:replication::c5be6b16-769d-432a-9ef1-4567081f3566:destbucket' \
         --priority 1 \
         --remote-bucket "destbucket"
         --replicate "delete,delete-marker"

 2. Add replication configuration rule with Disabled status on bucket "mybucket" for alias "myminio".
    {{.Prompt}} {{.HelpName}} myminio/mybucket/prefix --tags "key1=value1&key2=value2" \
        --storage-class "STANDARD" --disable \
        --arn 'arn:minio:replica::c5be6b16-769d-432a-9ef1-4567081f3566:destbucket' \
        --priority 1 \
		--remote-bucket "destbucket"

 3. Add replication configuration rule with existing object replication, delete marker replication and versioned deletes 
    enabled on bucket "mybucket" for alias "myminio".
	{{.Prompt}} {{.HelpName}} myminio/mybucket/prefix --tags "key1=value1&key2=value2" \
	 --storage-class "STANDARD" --disable \
	 --arn 'arn:minio:replica::c5be6b16-769d-432a-9ef1-4567081f3566:destbucket' \
	 --priority 1 \
	 --remote-bucket "destbucket" \
	 --replicate "existing-objects,delete,delete-marker"
`,
}

// checkReplicateAddSyntax - validate all the passed arguments
func checkReplicateAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "add", 1) // last argument is exit code
	}
	if ctx.String("arn") == "" {
		fatal(errDummy().Trace(), "--arn flag needs to be specified.")
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
	fatalIf(err, "Unable to initialize connection.")
	rcfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication configuration")
	ruleStatus := enableStatus
	if cliCtx.Bool(disableStatus) {
		ruleStatus = disableStatus
	}
	dmReplicateStatus := disableStatus
	deleteReplicationStatus := disableStatus
	replicaSync := enableStatus
	existingReplicationStatus := disableStatus
	if cliCtx.IsSet("replicate") {
		replSlice := strings.Split(cliCtx.String("replicate"), ",")
		for _, opt := range replSlice {
			switch strings.TrimSpace(strings.ToLower(opt)) {
			case "delete-marker":
				dmReplicateStatus = enableStatus
			case "delete":
				deleteReplicationStatus = enableStatus
			case "replica-metadata-sync":
				replicaSync = enableStatus
			case "existing-objects":
				existingReplicationStatus = enableStatus

			default:
				fatalIf(probe.NewError(fmt.Errorf("invalid value for --replicate flag %s", cliCtx.String("replicate"))), "--replicate flag takes one or more comma separated string with values \"delete, delete-marker, replica-metadata-sync\",\"existing-objects\" or \"\" to disable these settings")
			}
		}
	}

	opts := replication.Options{
		TagString:               cliCtx.String("tags"),
		RoleArn:                 cliCtx.String("arn"),
		StorageClass:            cliCtx.String("storage-class"),
		Priority:                strconv.Itoa(cliCtx.Int("priority")),
		RuleStatus:              ruleStatus,
		ID:                      cliCtx.String("id"),
		DestBucket:              cliCtx.String("remote-bucket"),
		Op:                      replication.AddOption,
		ReplicateDeleteMarkers:  dmReplicateStatus,
		ReplicateDeletes:        deleteReplicationStatus,
		ReplicaSync:             replicaSync,
		ExistingObjectReplicate: existingReplicationStatus,
	}
	fatalIf(client.SetReplication(ctx, &rcfg, opts), "Could not add replication rule")
	printMsg(replicateAddMessage{
		Op:  "add",
		URL: aliasedURL,
		ID:  opts.ID,
	})
	return nil
}
