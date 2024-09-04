// Copyright (c) 2015-2023 MinIO, Inc.
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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

var supportTopAPIFlags = []cli.Flag{
	cli.StringSliceFlag{
		Name:  "name",
		Usage: "summarize current calls for matching API name",
	},
	cli.StringSliceFlag{
		Name:  "path",
		Usage: "summarize current API calls only on matching path",
	},
	cli.StringSliceFlag{
		Name:  "node",
		Usage: "summarize current API calls only on matching servers",
	},
	cli.BoolFlag{
		Name:  "errors, e",
		Usage: "summarize current API calls throwing only errors",
	},
}

var supportTopAPICmd = cli.Command{
	Name:            "api",
	Usage:           "summarize API events on MinIO server in real-time",
	Action:          mainSupportTopAPI,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportTopAPIFlags, supportGlobalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Display current in-progress all S3 API calls.
      {{.Prompt}} {{.HelpName}} myminio/

   2. Display current in-progress all 's3.PutObject' API calls.
      {{.Prompt}} {{.HelpName}} --name s3.PutObject myminio/
`,
}

// checkSupportTopAPISyntax - validate all the passed arguments
func checkSupportTopAPISyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainSupportTopAPI(ctx *cli.Context) error {
	checkSupportTopAPISyntax(ctx)

	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)
	validateClusterRegistered(alias, false)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	opts, e := tracingOpts(ctx, ctx.StringSlice("call"))
	fatalIf(probe.NewError(e), "Unable to start tracing")

	mopts := matchingOpts(ctx)

	// Start listening on all trace activity.
	traceCh := client.ServiceTrace(ctxt, opts)

	filteredTraces := make(chan madmin.ServiceTraceInfo, 1)
	ui := tea.NewProgram(initTraceStatsUI(false, 30, filteredTraces))
	var te error
	go func() {
		for t := range traceCh {
			if t.Err != nil {
				te = t.Err
				ui.Kill()
				return
			}
			if mopts.matches(t) {
				filteredTraces <- t
			}
		}
	}()
	if _, e := ui.Run(); e != nil {
		cancel()
		if te != nil {
			e = te
		}
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch http trace statistics")
	}
	return nil
}
