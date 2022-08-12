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
	"github.com/google/uuid"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminRebalanceStartCmd = cli.Command{
	Name:         "start",
	Usage:        "Start rebalance operation",
	Action:       mainAdminRebalanceStart,
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
  1. Start rebalance on a MinIO deployment with alias myminio
     {{.Prompt}} {{.HelpName}} myminio
`,
}

type rebalanceStartMsg struct {
	Status string    `json:"status"`
	URL    string    `json:"url"`
	ID     uuid.UUID `json:"id"`
}

func (r rebalanceStartMsg) JSON() string {
	r.Status = "success"
	b, err := json.MarshalIndent(r, "", " ")
	fatalIf(probe.NewError(err), "Unable to marshal to JSON")
	return string(b)
}

func (r rebalanceStartMsg) String() string {
	return console.Colorize("rebalanceStartMsg", fmt.Sprintf("Rebalance started for %s", r.URL))
}

func mainAdminRebalanceStart(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "rebalance", 1)
	}

	console.SetColor("rebalanceStartMsg", color.New(color.FgGreen))

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	var pErr *probe.Error
	client, pErr := newAdminClient(aliasedURL)
	if pErr != nil {
		fatalIf(pErr.Trace(aliasedURL), "Unable to initialize admin client")
		return pErr.ToGoError()
	}

	var id uuid.UUID
	var err error
	id, err = client.RebalanceStart(globalContext)
	if err != nil {
		fatalIf(probe.NewError(err), "Failed to start rebalance")
		return err
	}
	printMsg(rebalanceStartMsg{
		URL: aliasedURL,
		ID:  id,
	})
	return nil
}
