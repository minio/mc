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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio/pkg/console"
)

var replicateExportCmd = cli.Command{
	Name:   "export",
	Usage:  "export server side replication configuration",
	Action: mainReplicateExport,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
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
