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
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminReplicateRemoveFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "all",
		Usage: "remove site replication from all participating sites",
	},
	cli.BoolFlag{
		Name:  "force",
		Usage: "force removal of site(s) from site replication configuration",
	},
}

var adminReplicateRemoveCmd = cli.Command{
	Name:          "remove",
	ShortName:     "rm",
	Usage:         "remove one or more sites from site replication",
	Action:        mainAdminReplicationRemoveStatus,
	OnUsageError:  onUsageError,
	HiddenAliases: true,
	Before:        setGlobalsFromContext,
	Flags:         append(globalFlags, adminReplicateRemoveFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Remove site replication for all sites:
     {{.Prompt}} {{.HelpName}} minio2 --all --force

  2. Remove site replication for site with site names alpha, baker from active cluster minio2:
     {{.Prompt}} {{.HelpName}} minio2 alpha baker --force
`,
}

type srRemoveStatus struct {
	madmin.ReplicateRemoveStatus
	sites     []string
	RemoveAll bool
}

func (i srRemoveStatus) JSON() string {
	ds, e := json.MarshalIndent(i.ReplicateRemoveStatus, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(ds)
}

func (i srRemoveStatus) String() string {
	if i.RemoveAll {
		return console.Colorize("UserMessage", "All site(s) were removed successfully")
	}
	if i.Status == madmin.ReplicateRemoveStatusSuccess {
		return console.Colorize("UserMessage", fmt.Sprintf("Following site(s) %s were removed successfully", i.sites))
	}
	if len(i.sites) == 1 {
		return console.Colorize("UserMessage", fmt.Sprintf("Following site %s was removed partially, some operations failed:\nERROR: '%s'", i.sites, i.ErrDetail))
	}
	return console.Colorize("UserMessage", fmt.Sprintf("Following site(s) %s were removed partially, some operations failed: \nERROR: '%s'", i.sites, i.ErrDetail))
}

func checkAdminReplicateRemoveSyntax(ctx *cli.Context) {
	// Check argument count
	argsNr := len(ctx.Args())
	if ctx.Bool("all") && argsNr > 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"")
	}
	if argsNr < 2 && !ctx.Bool("all") {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Need at least two arguments to remove command.")
	}
	if !ctx.Bool("force") {
		fatalIf(errDummy().Trace(),
			"Site removal requires --force flag. This operation is *IRREVERSIBLE*. Please review carefully before performing this *DANGEROUS* operation.")
	}
}

func mainAdminReplicationRemoveStatus(ctx *cli.Context) error {
	checkAdminReplicateRemoveSyntax(ctx)
	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	var rreq madmin.SRRemoveReq
	rreq.SiteNames = append(rreq.SiteNames, args.Tail()...)
	rreq.RemoveAll = ctx.Bool("all")
	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	st, e := client.SiteReplicationRemove(globalContext, rreq)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to remove cluster replication")

	printMsg(srRemoveStatus{
		ReplicateRemoveStatus: st,
		sites:                 args.Tail(),
		RemoveAll:             rreq.RemoveAll,
	})

	return nil
}
