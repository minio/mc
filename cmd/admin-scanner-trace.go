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
	"github.com/minio/pkg/v3/console"
)

var adminScannerTraceFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "print verbose trace",
	},
	cli.StringSliceFlag{
		Name:  "funcname",
		Usage: "trace only matching func name (eg 'scanner.ScanObject')",
	},
	cli.StringSliceFlag{
		Name:  "node",
		Usage: "trace only matching servers",
	},
	cli.StringSliceFlag{
		Name:  "path",
		Usage: "trace only matching path",
	},
	cli.BoolFlag{
		Name:  "filter-request",
		Usage: "trace calls only with request bytes greater than this threshold, use with filter-size",
	},
	cli.BoolFlag{
		Name:  "filter-response",
		Usage: "trace calls only with response bytes greater than this threshold, use with filter-size",
	},
	cli.BoolFlag{
		Name:  "response-duration",
		Usage: "trace calls only with response duration greater than this threshold (e.g. `5ms`)",
	},
	cli.StringFlag{
		Name:  "filter-size",
		Usage: "filter size, use with filter (see UNITS)",
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

UNITS
  --filter-size flags use with --filter-response or --filter-request accept human-readable case-insensitive number
  suffixes such as "k", "m", "g" and "t" referring to the metric units KB,
  MB, GB and TB respectively. Adding an "i" to these prefixes, uses the IEC
  units, so that "gi" refers to "gibibyte" or "GiB". A "b" at the end is
  also accepted. Without suffixes the unit is bytes.

EXAMPLES:
  1. Show scanner trace for MinIO server
     {{.Prompt}} {{.HelpName}} myminio

  2. Show scanner trace for a specific path
    {{.Prompt}} {{.HelpName}} --path my-bucket/my-prefix/* myminio

  3. Show trace for only ScanObject operations
    {{.Prompt}} {{.HelpName}} --funcname=scanner.ScanObject myminio

  4. Avoid printing replication related S3 requests
    {{.Prompt}} {{.HelpName}} --request-header '!X-Minio-Source' myminio

  5. Show trace only for ScanObject operations request bytes greater than 1MB
    {{.Prompt}} {{.HelpName}} --filter-request --filter-size 1MB myminio

  6. Show trace only for ScanObject operations response bytes greater than 1MB
    {{.Prompt}} {{.HelpName}} --filter-response --filter-size 1MB myminio
  
  7. Show trace only for requests operations duration greater than 5ms
    {{.Prompt}} {{.HelpName}} --response-duration 5ms myminio
`,
}

func checkAdminScannerTraceSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
	filterFlag := ctx.Bool("filter-request") || ctx.Bool("filter-response")
	if filterFlag && ctx.String("filter-size") == "" {
		// filter must use with filter-size flags
		showCommandHelpAndExit(ctx, 1)
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

	mopts := matchingOpts(ctx)

	// Start listening on all trace activity.
	traceCh := client.ServiceTrace(ctxt, opts)
	for traceInfo := range traceCh {
		if traceInfo.Err != nil {
			fatalIf(probe.NewError(traceInfo.Err), "Unable to listen to http trace")
		}
		if mopts.matches(traceInfo) {
			printTrace(verbose, traceInfo)
		}
	}
	return nil
}
