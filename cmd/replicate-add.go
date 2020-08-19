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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio/pkg/console"
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
}

var replicateAddCmd = cli.Command{
	Name:   "add",
	Usage:  "add a server side replication configuration rule",
	Action: mainReplicateAdd,
	Before: setGlobalsFromContext,
	Flags:  append(globalFlags, replicateAddFlags...),
	CustomHelpTemplate: `NAME:
	{{.HelpName}} - {{.Usage}}
	  
USAGE:
	{{.HelpName}} TARGET
	  
FLAGS:
	{{range .VisibleFlags}}{{.}}
	{{end}}
EXAMPLES:
	1. Add replication configuration rule on bucket "mybucket" for alias "myminio".
	   {{.Prompt}} {{.HelpName}} myminio/mybucket/prefix --tags "key1=value1&key2=value2" \ 
														 --storage-class "STANDARD" \
														 --arn 'arn:minio:replication::c5be6b16-769d-432a-9ef1-4567081f3566:destbucket' \
														 --priority 1 \
														 --remote-bucket "destbucket"

	2. Add replication configuration rule with Disabled status on bucket "mybucket" for alias "myminio".
      {{.Prompt}} {{.HelpName}} myminio/mybucket/prefix --tags "key1=value1&key2=value2" \ 
														--storage-class "STANDARD" --disable \
														--arn 'arn:minio:replica::c5be6b16-769d-432a-9ef1-4567081f3566:destbucket' \
														--priority 1 \
														--remote-bucket "destbucket"
`,
}

// checkReplicateAddSyntax - validate all the passed arguments
func checkReplicateAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "add", 1) // last argument is exit code
	}
}

type replicateAddMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	URL    string `json:"url"`
	ID     string `json:"id"`
}

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
	ruleStatus := "enable"
	if cliCtx.Bool("disable") {
		ruleStatus = "disable"
	}
	opts := replication.Options{
		TagString:    cliCtx.String("tags"),
		RoleArn:      cliCtx.String("arn"),
		StorageClass: cliCtx.String("storage-class"),
		Priority:     strconv.Itoa(cliCtx.Int("priority")),
		RuleStatus:   ruleStatus,
		ID:           cliCtx.String("id"),
		DestBucket:   cliCtx.String("remote-bucket"),
		Op:           replication.AddOption,
	}
	fatalIf(client.SetReplication(ctx, &rcfg, opts), "Could not add replication rule")
	printMsg(replicateAddMessage{
		Op:  "add",
		URL: aliasedURL,
		ID:  opts.ID,
	})
	return nil
}
