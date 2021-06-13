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
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"maze.io/x/duration"
)

var replicateResetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "older-than",
		Usage: "re-replicate objects older than n days",
	},
}

var replicateResetCmd = cli.Command{
	Name:         "resync",
	Usage:        "re-replicate all previously replicated objects",
	Aliases:      []string{"reset"},
	Action:       mainReplicateReset,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, replicateResetFlags...),
	CustomHelpTemplate: `NAME:
   {{.HelpName}} - {{.Usage}}

USAGE:
   {{.HelpName}} TARGET

FLAGS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
EXAMPLES:
  1. Re-replicate previously replicated objects in bucket "mybucket" for alias "myminio".
   {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. Re-replicate all objects older than 60 days in bucket "mybucket".
   {{.Prompt}} {{.HelpName}} myminio/mybucket --older-than 60d
`,
}

// checkReplicateResetSyntax - validate all the passed arguments
func checkReplicateResetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "reset", 1) // last argument is exit code
	}
}

type replicateResetMessage struct {
	Op      string `json:"op"`
	URL     string `json:"url"`
	ResetID string `json:"resetID"`
	Status  string `json:"status"`
}

func (r replicateResetMessage) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (r replicateResetMessage) String() string {
	return console.Colorize("replicateResetMessage", fmt.Sprintf("Replication reset started for %s with ID %s", r.URL, r.ResetID))
}

func mainReplicateReset(cliCtx *cli.Context) error {
	ctx, cancelReplicateReset := context.WithCancel(globalContext)
	defer cancelReplicateReset()

	console.SetColor("replicateResetMessage", color.New(color.FgGreen))

	checkReplicateResetSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	var olderThanStr string
	var olderThan time.Duration
	if cliCtx.IsSet("older-than") {
		olderThanStr = cliCtx.String("older-than")
		if olderThanStr != "" {
			days, e := duration.ParseDuration(olderThanStr)
			if e != nil || !strings.ContainsAny(olderThanStr, "dwy") {
				fatalIf(probe.NewError(e), "Unable to parse older-than=`"+olderThanStr+"`.")
			}
			if days == 0 {
				fatalIf(probe.NewError(e), "older-than cannot be set to zero")
			}
			olderThan = time.Duration(days.Days())
		}
	}

	replicateReset, err := client.ResetReplication(ctx, olderThan)
	fatalIf(err.Trace(args...), "Unable to reset replication")
	printMsg(replicateResetMessage{
		Op:      "status",
		URL:     aliasedURL,
		ResetID: replicateReset,
	})
	return nil
}
