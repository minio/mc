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
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminReplicateEditFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "deployment-id",
		Usage: "deployment id of the site, should be a unique value",
	},
	cli.StringFlag{
		Name:  "endpoint",
		Usage: "endpoint for the site",
	},
}

var adminReplicateEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "edit endpoint of site participating in cluster replication",
	Action:       mainAdminReplicateEdit,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminReplicateEditFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS --deployment-id [DEPLOYMENT-ID] --endpoint [NEW-ENDPOINT]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Edit a site endpoint participating in cluster-level replication:
     {{.Prompt}} {{.HelpName}} myminio --deployment-id c1758167-4426-454f-9aae-5c3dfdf6df64 --endpoint https://minio2:9000
`,
}

type editSuccessMessage madmin.ReplicateEditStatus

func (m editSuccessMessage) JSON() string {
	bs, e := json.MarshalIndent(madmin.ReplicateEditStatus(m), "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(bs)
}

func (m editSuccessMessage) String() string {
	v := madmin.ReplicateEditStatus(m)
	messages := []string{v.Status}

	if v.ErrDetail != "" {
		messages = append(messages, v.ErrDetail)
	}
	return console.Colorize("UserMessage", strings.Join(messages, "\n"))
}

func checkAdminReplicateEditSyntax(ctx *cli.Context) {
	// Check argument count
	argsNr := len(ctx.Args())
	if argsNr < 1 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
	if argsNr != 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Invalid arguments specified for edit command.")
	}
}

func mainAdminReplicateEdit(ctx *cli.Context) error {
	checkAdminReplicateEditSyntax(ctx)
	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	if !ctx.IsSet("deployment-id") {
		fatalIf(errInvalidArgument(), "--deployment-id is a required flag")
	}
	if !ctx.IsSet("endpoint") {
		fatalIf(errInvalidArgument(), "--endpoint is a required flag")
	}
	parsedURL := ctx.String("endpoint")
	u, e := url.Parse(parsedURL)
	if e != nil {
		fatalIf(errInvalidArgument().Trace(parsedURL), "Unsupported URL format %v", e)
	}
	res, e := client.SiteReplicationEdit(globalContext, madmin.PeerInfo{
		DeploymentID: ctx.String("deployment-id"),
		Endpoint:     u.String(),
	})
	fatalIf(probe.NewError(e).Trace(args...), "Unable to edit cluster replication site endpoint")

	printMsg(editSuccessMessage(res))

	return nil
}
