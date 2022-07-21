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

	"encoding/json"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminRebalanceStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "Fetch status of an ongoing rebalance operation",
	Action:       mainAdminRebalanceStatus,
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
  1. Fetch status of an ongoing rebalance on a MinIO deployment with alias myminio
     {{.Prompt}} {{.HelpName}} myminio
`,
}

type rebalanceStatusMsg struct {
	Status string    `json:"status"`
	URL    string    `json:"url"`
	ARN    uuid.UUID `json:"arn"`
}

func mainAdminRebalanceStatus(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "rebalance", 1)
	}

	console.SetColor("rebalanceStatusMsg", color.New(color.FgGreen))

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	var pErr *probe.Error
	client, pErr := newAdminClient(aliasedURL)
	if pErr != nil {
		fatalIf(pErr.Trace(aliasedURL), "Unable to initialize admin client")
		return pErr.ToGoError()
	}

	rInfo, err := client.RebalanceStatus(globalContext)
	if err != nil {
		fatalIf(probe.NewError(err), "Failed to get rebalance status")
	}
	b, err := json.Marshal(rInfo)
	fatalIf(probe.NewError(err), "Failed to marshal json")
	fmt.Println(string(b))
	return nil
}
