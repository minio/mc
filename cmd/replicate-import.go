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
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/replication"
	"github.com/minio/minio/pkg/console"
)

var replicateImportCmd = cli.Command{
	Name:   "import",
	Usage:  "import server side replication configuration in JSON format",
	Action: mainReplicateImport,
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
  1. Set replication configuration from '/data/replication/config' on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket < '/data/replication/config'

  2. Import replication configuration for bucket "mybucket" on alias "myminio" from STDIN.
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkReplicateImportSyntax - validate all the passed arguments
func checkReplicateImportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "import", 1) // last argument is exit code
	}
}

type replicateImportMessage struct {
	Op                string             `json:"op"`
	Status            string             `json:"status"`
	URL               string             `json:"url"`
	ReplicationConfig replication.Config `json:"config"`
}

func (r replicateImportMessage) JSON() string {
	r.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (r replicateImportMessage) String() string {
	return console.Colorize("replicateImportMessage", "Replication configuration successfully set on `"+r.URL+"`.")
}

// readReplicationConfig read from stdin, returns XML.
func readReplicationConfig() (*replication.Config, *probe.Error) {
	// User is expected to enter the replication configuration in JSON format
	var cfg = replication.Config{}

	// Consume json from STDIN
	dec := json.NewDecoder(os.Stdin)
	if e := dec.Decode(&cfg); e != nil {
		return &cfg, probe.NewError(e)
	}

	return &cfg, nil
}

func mainReplicateImport(cliCtx *cli.Context) error {
	ctx, cancelReplicateImport := context.WithCancel(globalContext)
	defer cancelReplicateImport()

	console.SetColor("replicateImportMessage", color.New(color.FgGreen))
	checkReplicateImportSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	rCfg, err := readReplicationConfig()
	fatalIf(err.Trace(args...), "Unable to read replication configuration")

	fatalIf(client.SetReplication(ctx, rCfg, replication.Options{Op: replication.ImportOption}).Trace(aliasedURL), "Unable to set replication configuration")
	printMsg(replicateImportMessage{
		Op:     "import",
		Status: "success",
		URL:    aliasedURL,
	})
	return nil
}
