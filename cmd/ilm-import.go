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

	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/minio/pkg/console"
)

var ilmImportCmd = cli.Command{
	Name:   "import",
	Usage:  "import lifecycle configuration in JSON format",
	Action: mainILMImport,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Import entire lifecycle configuration from STDIN, input file is expected to be in JSON format.

EXAMPLES:
  1. Set lifecycle configuration for the mybucket on alias 'myminio' to the rules imported from lifecycle.json
     {{.Prompt}} {{.HelpName}} myminio/mybucket < lifecycle.json

  2. Set lifecycle configuration for the mybucket on alias 'myminio'. User is expected to enter the JSON contents on STDIN
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

type ilmImportMessage struct {
	Status string `json:"status"`
	Target string `json:"target"`
}

func (i ilmImportMessage) String() string {
	return console.Colorize(ilmThemeResultSuccess, "Lifecycle configuration imported successfully to `"+i.Target+"`.")
}

func (i ilmImportMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// readILMConfig read from stdin, returns XML.
func readILMConfig() (*lifecycle.Configuration, *probe.Error) {
	// User is expected to enter the lifecycleConfiguration instance contents in JSON format
	var cfg = lifecycle.NewConfiguration()

	// Consume json from STDIN
	dec := json.NewDecoder(os.Stdin)
	if e := dec.Decode(cfg); e != nil {
		return cfg, probe.NewError(e)
	}

	return cfg, nil
}

// checkILMImportSyntax - validate arguments passed by user
func checkILMImportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "import", globalErrorExitStatus)
	}
}

func mainILMImport(cliCtx *cli.Context) error {
	ctx, cancelILMImport := context.WithCancel(globalContext)
	defer cancelILMImport()

	checkILMImportSyntax(cliCtx)
	setILMDisplayColorScheme()

	args := cliCtx.Args()
	urlStr := args.Get(0)

	client, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize client for "+urlStr)

	ilmCfg, err := readILMConfig()
	fatalIf(err.Trace(args...), "Unable to read ILM configuration")

	fatalIf(client.SetLifecycle(ctx, ilmCfg).Trace(urlStr), "Unable to set new lifecycle rules")

	printMsg(ilmImportMessage{
		Status: "success",
		Target: urlStr,
	})
	return nil
}
