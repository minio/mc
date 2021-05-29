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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/pkg/console"
)

var replicateExportCmd = cli.Command{
	Name:         "export",
	Usage:        "export server side replication configuration",
	Action:       mainReplicateExport,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}
	 
USAGE:
  {{.HelpName}} TARGET
	 
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Print replication configuration on bucket "mybucket" for alias "myminio" to STDOUT.
     {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. Export replication configuration on bucket "mybucket" for alias "myminio" to '/data/replicate/config'.
     {{.Prompt}} {{.HelpName}} myminio/mybucket > /data/replicate/config
`,
}

// checkReplicateExportSyntax - validate all the passed arguments
func checkReplicateExportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "export", 1) // last argument is exit code
	}
}

type replicateExportMessage struct {
	Op                string             `json:"op"`
	Status            string             `json:"status"`
	URL               string             `json:"url"`
	ReplicationConfig replication.Config `json:"config"`
}

func (r replicateExportMessage) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (r replicateExportMessage) String() string {
	if r.ReplicationConfig.Empty() {
		return console.Colorize("ReplicateNMessage", "No replication configuration found for "+r.URL+".")
	}
	msgBytes, e := json.MarshalIndent(r.ReplicationConfig, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal replication configuration")
	return string(msgBytes)
}

func mainReplicateExport(cliCtx *cli.Context) error {
	ctx, cancelReplicateExport := context.WithCancel(globalContext)
	defer cancelReplicateExport()

	console.SetColor("replicateExportMessage", color.New(color.FgGreen))
	console.SetColor("replicateExportFailure", color.New(color.FgRed))

	checkReplicateExportSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	rCfg, err := client.GetReplication(ctx)
	fatalIf(err.Trace(args...), "Unable to get replication configuration")
	printMsg(replicateExportMessage{
		Op:                "export",
		Status:            "success",
		URL:               aliasedURL,
		ReplicationConfig: rCfg,
	})
	return nil
}
