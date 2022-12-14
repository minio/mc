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

var supportTopUnLocksFlag = []cli.Flag{
	cli.StringSliceFlag{
		Name:  "locks, l",
		Usage: "unlock locks",
		Value: nil,
	},
}

var supportTopUnLocksCmd = cli.Command{
	Name:         "unlocks",
	Usage:        "unlocks locks on a MinIO cluster.",
	Before:       setGlobalsFromContext,
	Action:       mainSupportTopUnLocks,
	OnUsageError: onUsageError,
	Flags:        append(supportTopUnLocksFlag, supportGlobalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [FLAGS]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. unlocks locks on a MinIO cluster.
     {{.Prompt}} {{.HelpName}} myminio --locks lock1 --locks lock2
`,
}

// checkSupportTopUnLocksSyntax - validate all the passed arguments
func checkSupportTopUnLocksSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.StringSlice("locks")) < 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainSupportTopUnLocks(ctx *cli.Context) error {
	checkSupportTopUnLocksSyntax(ctx)
	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	alias, _ := url2Alias(aliasedURL)
	validateClusterRegistered(alias, false)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	// Call unlocks API
	locks := ctx.StringSlice("locks")

	e := client.ForceUnlock(globalContext, locks...)
	fatalIf(probe.NewError(e), "Unable to unlock server locks list.")

	return nil
}
