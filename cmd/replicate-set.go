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
	"context"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio/pkg/console"
)

var replicateSetFlags = []cli.Flag{
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
}

var replicateSetCmd = cli.Command{
	Name:   "set",
	Usage:  "modify an existing server side replication configuration rule",
	Action: mainReplicateSet,
	Before: setGlobalsFromContext,
	Flags:  append(globalFlags, replicateSetFlags...),
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
`,
}

// checkReplicateSetSyntax - validate all the passed arguments
func checkReplicateSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "set", 1) // last argument is exit code
	}
}

type replicateSetMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	URL    string `json:"url"`
	ID     string `json:"id"`
}

func (l replicateSetMessage) JSON() string {
	l.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (l replicateSetMessage) String() string {
	if l.ID != "" {
		return console.Colorize("replicateSetMessage", "Replication configuration rule with ID `"+l.ID+"` applied to "+l.URL+".")
	}
	return console.Colorize("replicateSetMessage", "Replication configuration rule applied to "+l.URL+" successfully.")
}

func mainReplicateSet(cliCtx *cli.Context) error {
	ctx, cancelReplicateSet := context.WithCancel(globalContext)
	defer cancelReplicateSet()

	console.SetColor("replicateSetMessage", color.New(color.FgGreen))

	checkReplicateSetSyntax(cliCtx)

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

	fatalIf(client.SetReplication(ctx, &rcfg, opts), "Could not modify replication rule")
	printMsg(replicateSetMessage{
		Op:  "set",
		URL: aliasedURL,
		ID:  opts.ID,
	})
	return nil
}
