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
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/v3/console"
)

var replicateListFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "status",
		Usage: "show rules by status. Valid options are [enabled,disabled]",
	},
}

var replicateListCmd = cli.Command{
	Name:         "list",
	ShortName:    "ls",
	Usage:        "list server side replication configuration rules",
	Action:       mainReplicateList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateListFlags...),
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
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func printReplicateListHeader() {
	if globalJSON {
		return
	}
	console.Println(console.Colorize("Headers", "Rules:"))
}

type replicateListMessage struct {
	Op      string           `json:"op"`
	Status  string           `json:"status"`
	URL     string           `json:"url"`
	Rule    replication.Rule `json:"rule"`
	targets []madmin.BucketTarget
}

func (l replicateListMessage) JSON() string {
	l.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (l replicateListMessage) String() string {
	r := l.Rule
	destBucket := r.Destination.Bucket
	if arn, err := madmin.ParseARN(r.Destination.Bucket); err == nil {
		destBucket = arn.Bucket
	}
	endpoint := r.Destination.Bucket
	for _, t := range l.targets {
		if t.Arn == r.Destination.Bucket {
			endpoint = t.Endpoint
			break
		}
	}
	var sb strings.Builder
	sb.WriteString(console.Colorize("Key", "Remote Bucket: "))

	sb.WriteString(console.Colorize("EpVal", fmt.Sprintf("%s/%s\n", endpoint, destBucket)))

	sb.WriteString(fmt.Sprintf("  Rule ID: %s\n", console.Colorize("Val", r.ID)))
	sb.WriteString(fmt.Sprintf("  Priority: %s\n", console.Colorize("Val", r.Priority)))
	sb.WriteString(fmt.Sprintf("  ARN: %s\n", console.Colorize("Val", r.Destination.Bucket)))
	if r.Filter.And.Prefix != "" {
		sb.WriteString(fmt.Sprintf("  Prefix: %s\n", console.Colorize("Val", r.Filter.And.Prefix)))
	}
	if r.Tags() != "" {
		sb.WriteString(fmt.Sprintf("  Tags: %s\n", console.Colorize("Val", r.Tags())))
	}
	if r.Destination.StorageClass != "" && r.Destination.StorageClass != "STANDARD" {
		sb.WriteString(fmt.Sprintf("  StorageClass: %s\n", console.Colorize("Val", r.Destination.StorageClass)))
	}
	return sb.String() + "\n"
}

func mainReplicateList(cliCtx *cli.Context) error {
	ctx, cancelReplicateList := context.WithCancel(globalContext)
	defer cancelReplicateList()

	console.SetColor("Headers", color.New(color.Bold, color.FgHiGreen))
	console.SetColor("Key", color.New(color.Bold, color.FgWhite))

	console.SetColor("Val", color.New(color.Bold, color.FgCyan))
	console.SetColor("EpVal", color.New(color.Bold, color.FgYellow))

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
	// Create a new MinIO Admin Client
	admClient, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")
	_, sourceBucket := url2Alias(args[0])
	targets, e := admClient.ListRemoteTargets(globalContext, sourceBucket, "")
	fatalIf(probe.NewError(e).Trace(args...), "Unable to fetch remote target.")

	statusFlag := cliCtx.String("status")
	for _, rule := range rCfg.Rules {
		if statusFlag == "" || strings.EqualFold(statusFlag, string(rule.Status)) {
			printMsg(replicateListMessage{
				Rule:    rule,
				targets: targets,
			})
		}
	}

	return nil
}
