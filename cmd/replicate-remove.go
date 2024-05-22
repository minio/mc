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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/v3/console"
)

var replicateRemoveFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "id",
		Usage: "id for the rule, should be a unique value",
	},
	cli.BoolFlag{
		Name:  "force",
		Usage: "force remove all the replication configuration rules on the bucket",
	},
	cli.BoolFlag{
		Name:  "all",
		Usage: "remove all replication configuration rules of the bucket, force flag enforced",
	},
}

var replicateRemoveCmd = cli.Command{
	Name:         "remove",
	ShortName:    "rm",
	Usage:        "remove a server side replication configuration rule",
	Action:       mainReplicateRemove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateRemoveFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove replication configuration rule on bucket "mybucket" for alias "myminio" with rule id "bsib5mgt874bi56l0fmg".
     {{.Prompt}} {{.HelpName}} --id "bsib5mgt874bi56l0fmg" myminio/mybucket

  2. Remove all the replication configuration rules on bucket "mybucket" for alias "myminio". --force flag is required.
     {{.Prompt}} {{.HelpName}} --all --force myminio/mybucket
`,
}

// checkReplicateRemoveSyntax - validate all the passed arguments
func checkReplicateRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
	rmAll := ctx.Bool("all")
	rmForce := ctx.Bool("force")
	rID := ctx.String("id")

	rmChk := (rmAll && rmForce) || (!rmAll && !rmForce)
	if !rmChk {
		fatalIf(errInvalidArgument(),
			"It is mandatory to specify --all and --force flag together for mc "+ctx.Command.FullName()+".")
	}
	if rmAll && rmForce {
		return
	}

	if rID == "" {
		fatalIf(errInvalidArgument().Trace(rID), "rule ID cannot be empty")
	}
}

type replicateRemoveMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	URL    string `json:"url"`
	ID     string `json:"id"`
}

func (l replicateRemoveMessage) JSON() string {
	l.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(l, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (l replicateRemoveMessage) String() string {
	if l.ID != "" {
		return console.Colorize("replicateRemoveMessage", "Replication configuration rule with ID `"+l.ID+"`removed from "+l.URL+".")
	}
	return console.Colorize("replicateRemoveMessage", "Replication configuration removed from "+l.URL+" successfully.")
}

func mainReplicateRemove(cliCtx *cli.Context) error {
	ctx, cancelReplicateRemove := context.WithCancel(globalContext)
	defer cancelReplicateRemove()

	console.SetColor("replicateRemoveMessage", color.New(color.FgGreen))

	checkReplicateRemoveSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	rcfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication configuration")

	rmAll := cliCtx.Bool("all")
	rmForce := cliCtx.Bool("force")
	ruleID := cliCtx.String("id")

	if rcfg.Empty() && !rmAll {
		printMsg(replicateRemoveMessage{
			Op:     cliCtx.Command.Name,
			Status: "success",
			URL:    aliasedURL,
		})
		return nil
	}
	if rmAll && rmForce {
		fatalIf(client.RemoveReplication(ctx), "Unable to remove replication configuration")
	} else {
		var removeArn string
		for _, rule := range rcfg.Rules {
			if rule.ID == ruleID {
				removeArn = rule.Destination.Bucket
			}
		}
		opts := replication.Options{
			ID: ruleID,
			Op: replication.RemoveOption,
		}
		fatalIf(client.SetReplication(ctx, &rcfg, opts), "Could not remove replication rule")
		admclient, cerr := newAdminClient(aliasedURL)
		fatalIf(cerr.Trace(aliasedURL), "Unable to initialize admin connection.")
		_, sourceBucket := url2Alias(args[0])
		fatalIf(probe.NewError(admclient.RemoveRemoteTarget(globalContext, sourceBucket, removeArn)).Trace(args...), "Unable to remove remote target")

	}
	printMsg(replicateRemoveMessage{
		Op:     cliCtx.Command.Name,
		Status: "success",
		URL:    aliasedURL,
		ID:     ruleID,
	})
	return nil
}
