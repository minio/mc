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
	"errors"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

var ilmExportCmd = cli.Command{
	Name:         "export",
	Usage:        "export lifecycle configuration in JSON format",
	Action:       mainILMExport,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

DESCRIPTION:
  Exports lifecycle configuration in JSON format to STDOUT.

EXAMPLES:
  1. Export lifecycle configuration for 'mybucket' to 'lifecycle.json' file.
     {{.Prompt}} {{.HelpName}} myminio/mybucket > lifecycle.json

  2. Print lifecycle configuration for 'mybucket' to STDOUT.
     {{.Prompt}} {{.HelpName}} play/mybucket
`,
}

type ilmExportMessage struct {
	Status string                   `json:"status"`
	Target string                   `json:"target"`
	Config *lifecycle.Configuration `json:"config"`
}

func (i ilmExportMessage) String() string {
	msgBytes, e := json.MarshalIndent(i.Config, "", " ")
	fatalIf(probe.NewError(e), "Unable to export ILM configuration")

	return string(msgBytes)
}

func (i ilmExportMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal ILM message")

	return string(msgBytes)
}

// checkILMExportSyntax - validate arguments passed by user
func checkILMExportSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "export", globalErrorExitStatus)
	}
}

func mainILMExport(cliCtx *cli.Context) error {
	ctx, cancelILMExport := context.WithCancel(globalContext)
	defer cancelILMExport()

	checkILMExportSyntax(cliCtx)
	setILMDisplayColorScheme()

	args := cliCtx.Args()
	urlStr := args.Get(0)

	client, err := newClient(urlStr)
	fatalIf(err.Trace(args...), "Unable to initialize client for "+urlStr+".")

	ilmCfg, err := client.GetLifecycle(ctx)
	fatalIf(err.Trace(args...), "Unable to get lifecycle configuration")
	if len(ilmCfg.Rules) == 0 {
		fatalIf(probe.NewError(errors.New("lifecycle configuration not set")).Trace(urlStr),
			"Unable to export lifecycle configuration")
	}

	printMsg(ilmExportMessage{
		Status: "success",
		Target: urlStr,
		Config: ilmCfg,
	})

	return nil
}
