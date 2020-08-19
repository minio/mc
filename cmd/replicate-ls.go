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
	"errors"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio/pkg/console"
)

var replicateListFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "status",
		Usage: "show rules by status. Valid options are [enabled,disabled]",
	},
}

var replicateListCmd = cli.Command{
	Name:   "ls",
	Usage:  "list server side replication configuration rules",
	Action: mainReplicateList,
	Before: setGlobalsFromContext,
	Flags:  append(globalFlags, replicateListFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
	 
USAGE:
  {{.HelpName}} TARGET
	 
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List server side replication configuration rules on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkReplicateListSyntax - validate all the passed arguments
func checkReplicateListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1) // last argument is exit code
	}
}
func printReplicateListHeader() {
	if globalJSON {
		return
	}
	idFieldMaxLen := 20
	priorityFieldMaxLen := 8
	statusFieldMaxLen := 8
	prefixFieldMaxLen := 25
	tagsFieldMaxLen := 25
	scFieldMaxLen := 15
	destBucketFieldMaxLen := 20
	console.Println(console.Colorize("Headers", newPrettyTable(" | ",
		Field{"ID", idFieldMaxLen},
		Field{"Priority", priorityFieldMaxLen},
		Field{"Status", statusFieldMaxLen},
		Field{"Prefix", prefixFieldMaxLen},
		Field{"Tags", tagsFieldMaxLen},
		Field{"DestBucket", destBucketFieldMaxLen},
		Field{"StorageClass", scFieldMaxLen},
	).buildRow("ID", "Priority", "Status", "Prefix", "Tags", "DestBucket", "StorageClass")))
}

type replicateListMessage struct {
	Op     string           `json:"op"`
	Status string           `json:"status"`
	URL    string           `json:"url"`
	Rule   replication.Rule `json:"rule"`
}

func (l replicateListMessage) JSON() string {
	l.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (l replicateListMessage) String() string {
	idFieldMaxLen := 20
	priorityFieldMaxLen := 8
	statusFieldMaxLen := 8
	prefixFieldMaxLen := 25
	tagsFieldMaxLen := 25
	scFieldMaxLen := 15
	destBucketFieldMaxLen := 20
	r := l.Rule
	return console.Colorize("replicateListMessage", newPrettyTable(" | ",
		Field{"ID", idFieldMaxLen},
		Field{"Priority", priorityFieldMaxLen},
		Field{"Status", statusFieldMaxLen},
		Field{"Prefix", prefixFieldMaxLen},
		Field{"Tags", tagsFieldMaxLen},
		Field{"DestBucket", destBucketFieldMaxLen},
		Field{"StorageClass", scFieldMaxLen},
	).buildRow(r.ID, strconv.Itoa(r.Priority), string(r.Status), r.Filter.And.Prefix, r.Tags(), r.Destination.Bucket, r.Destination.StorageClass))
}

func mainReplicateList(cliCtx *cli.Context) error {
	ctx, cancelReplicateList := context.WithCancel(globalContext)
	defer cancelReplicateList()

	console.SetColor("Headers", color.New(color.Bold, color.FgHiGreen))

	checkReplicateListSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	rCfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication configuration")

	if rCfg.Empty() {
		fatalIf(probe.NewError(errors.New("replication configuration not set")).Trace(aliasedURL),
			"Unable to list replication configuration")
	}
	printReplicateListHeader()
	statusFlag := cliCtx.String("status")
	for _, rule := range rCfg.Rules {
		if statusFlag == "" || strings.EqualFold(statusFlag, string(rule.Status)) {
			printMsg(replicateListMessage{
				Rule: rule,
			})
		}
	}

	return nil
}
