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
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var adminTierRmCmd = cli.Command{
	Name:         "rm",
	Usage:        "removes an empty remote tier",
	Action:       mainAdminTierRm,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS NAME

NAME:
  Name of remote tier target. e.g WARM-TIER

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Remove an empty tier by name 'WARM-TIER':
     {{.Prompt}} {{.HelpName}} myminio WARM-TIER
`,
}

func mainAdminTierRm(ctx *cli.Context) error {
	args := ctx.Args()
	nArgs := len(args)
	if nArgs < 2 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1)
	}
	if nArgs != 2 {
		fatalIf(errInvalidArgument().Trace(args.Tail()...),
			"Incorrect number of arguments for tier remove command.")
	}

	aliasedURL := args.Get(0)
	tierName := args.Get(1)
	if tierName == "" {
		fatalIf(errInvalidArgument(), "Tier name can't be empty")
	}

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	if err := client.RemoveTier(globalContext, tierName); err != nil {
		fatalIf(probe.NewError(err).Trace(args...), "Unable to remove remote tier target")
	}

	printMsg(&tierMessage{
		op:       "rm",
		Status:   "success",
		TierName: tierName,
	})
	return nil
}
