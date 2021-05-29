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
	"os"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/pkg/console"
)

var ilmImportCmd = cli.Command{
	Name:         "import",
	Usage:        "import lifecycle configuration in JSON format",
	Action:       mainILMImport,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
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

	if len(ilmCfg.Rules) == 0 {
		// Abort here, otherwise client.SetLifecycle will remove the lifecycle configuration
		// since no rules are provided and we will show a success message.
		fatalIf(errDummy(), "The provided ILM configuration does not contain any rule, aborting.")
	}

	fatalIf(client.SetLifecycle(ctx, ilmCfg).Trace(urlStr), "Unable to set new lifecycle rules")

	printMsg(ilmImportMessage{
		Status: "success",
		Target: urlStr,
	})
	return nil
}
