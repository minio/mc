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
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminScannerTraceFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "response-threshold",
		Usage: "trace calls only with response duration greater than this threshold (e.g. `5ms`)",
	},
	cli.StringSliceFlag{
		Name:  "funcname",
		Usage: "trace only matching func name (eg 'scanner.ScanObject')",
	},
	cli.StringSliceFlag{
		Name:  "node",
		Usage: "trace only matching servers",
	},
}

var adminScannerTraceCmd = cli.Command{
	Name:            "trace",
	Usage:           "show trace for MinIO scanner operations",
	Action:          mainAdminScannerTrace,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminScannerTraceFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show scanner trace for MinIO server
     {{.Prompt}} {{.HelpName}} myminio

  2. Show scanner trace for a specific path
    {{.Prompt}} {{.HelpName}} --path my-bucket/my-prefix/* myminio

  3. Show trace for only ScanObject operations
    {{.Prompt}} {{.HelpName}} --funcname=scanner.ScanObject myminio
`,
}

func checkAdminScannerTraceSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "trace", 1) // last argument is exit code
	}
}

// mainAdminScannerTrace - the entry function of trace command
func mainAdminScannerTrace(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminScannerTraceSyntax(ctx)

	verbose := ctx.Bool("verbose")
	aliasedURL := ctx.Args().Get(0)

	console.SetColor("Stat", color.New(color.FgYellow))

	console.SetColor("Request", color.New(color.FgCyan))
	console.SetColor("Method", color.New(color.Bold, color.FgWhite))
	console.SetColor("Host", color.New(color.Bold, color.FgGreen))
	console.SetColor("FuncName", color.New(color.Bold, color.FgGreen))

	console.SetColor("ReqHeaderKey", color.New(color.Bold, color.FgWhite))
	console.SetColor("RespHeaderKey", color.New(color.Bold, color.FgCyan))
	console.SetColor("HeaderValue", color.New(color.FgWhite))
	console.SetColor("RespStatus", color.New(color.Bold, color.FgYellow))
	console.SetColor("ErrStatus", color.New(color.Bold, color.FgRed))

	console.SetColor("Response", color.New(color.FgGreen))
	console.SetColor("Body", color.New(color.FgYellow))
	for _, c := range colors {
		console.SetColor(fmt.Sprintf("Node%d", c), color.New(c))
	}
	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	opts, e := tracingOpts(ctx, []string{"scanner"})
	fatalIf(probe.NewError(e), "Unable to start tracing")

	mopts := matchOpts{
		funcNames: ctx.StringSlice("funcname"),
		apiPaths:  ctx.StringSlice("path"),
		nodes:     ctx.StringSlice("node"),
	}

	// Start listening on all trace activity.
	traceCh := client.ServiceTrace(ctxt, opts)
	for traceInfo := range traceCh {
		if traceInfo.Err != nil {
			fatalIf(probe.NewError(traceInfo.Err), "Unable to listen to http trace")
		}
		if matchTrace(mopts, traceInfo) {
			printTrace(verbose, traceInfo)
		}
	}
	return nil
}
