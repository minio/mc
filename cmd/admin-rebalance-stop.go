// Copyright (c) 2022 MinIO, Inc.
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
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminRebalanceStopCmd = cli.Command{
	Name:         "stop",
	Usage:        "stop an ongoing rebalance operation",
	Action:       mainAdminRebalanceStop,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Stop an ongoing rebalance on a MinIO deployment with alias myminio
     {{.Prompt}} {{.HelpName}} myminio
`,
}

type rebalanceStopMsg struct {
	Status string `json:"status"`
	Target string `json:"url"`
}

func (r rebalanceStopMsg) JSON() string {
	r.Status = "success"
	b, e := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal to JSON")
	return string(b)
}

func (r rebalanceStopMsg) String() string {
	return console.Colorize("rebalanceStopMsg", fmt.Sprintf("Rebalance stopped for %s", r.Target))
}

func mainAdminRebalanceStop(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1)
	}

	console.SetColor("rebalanceStopMsg", color.New(color.FgGreen))

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client")

	fatalIf(probe.NewError(client.RebalanceStop(globalContext)), "Unable to stop rebalance operation")

	printMsg(rebalanceStopMsg{
		Target: aliasedURL,
	})

	return nil
}
