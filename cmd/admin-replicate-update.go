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
	"net/url"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminReplicateUpdateFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "deployment-id",
		Usage: "deployment id of the site, should be a unique value",
	},
	cli.StringFlag{
		Name:  "endpoint",
		Usage: "endpoint for the site",
	},
	cli.StringFlag{
		Name:  "sync",
		Usage: "enable synchronous replication for this target, valid values are ['enable', 'disable'].",
		Value: "disable",
	},
}

var adminReplicateUpdateCmd = cli.Command{
	Name:          "update",
	Aliases:       []string{"edit"},
	HiddenAliases: true,
	Usage:         "modify endpoint of site participating in site replication",
	Action:        mainAdminReplicateUpdate,
	OnUsageError:  onUsageError,
	Before:        setGlobalsFromContext,
	Flags:         append(globalFlags, adminReplicateUpdateFlags...),
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

type updateSuccessMessage madmin.ReplicateEditStatus

func (m updateSuccessMessage) JSON() string {
	bs, e := json.MarshalIndent(madmin.ReplicateEditStatus(m), "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(bs)
}

func (m updateSuccessMessage) String() string {
	v := madmin.ReplicateEditStatus(m)
	messages := []string{v.Status}

	if v.ErrDetail != "" {
		messages = append(messages, v.ErrDetail)
	}
	return console.Colorize("UserMessage", strings.Join(messages, "\n"))
}

func checkAdminReplicateUpdateSyntax(ctx *cli.Context) {
	// Check argument count
	argsNr := len(ctx.Args())
	if argsNr < 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
	if argsNr != 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Invalid arguments specified for edit command.")
	}
}

func mainAdminReplicateUpdate(ctx *cli.Context) error {
	checkAdminReplicateUpdateSyntax(ctx)
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
	if !ctx.IsSet("endpoint") && !ctx.IsSet("sync") {
		fatalIf(errInvalidArgument(), "--endpoint or --sync is a required flag")
	}
	var syncState string
	if ctx.IsSet("sync") {
		syncState = strings.ToLower(ctx.String("sync"))
		switch syncState {
		case "enable", "disable":
		default:
			fatalIf(errInvalidArgument().Trace(args...), "--sync can be either [enable|disable]")
		}
	}
	var ep string
	if ctx.IsSet("endpoint") {
		parsedURL := ctx.String("endpoint")
		u, e := url.Parse(parsedURL)
		if e != nil {
			fatalIf(errInvalidArgument().Trace(parsedURL), "Unsupported URL format %v", e)
		}
		ep = u.String()
	}
	res, e := client.SiteReplicationEdit(globalContext, madmin.PeerInfo{
		DeploymentID: ctx.String("deployment-id"),
		Endpoint:     ep,
		SyncState:    madmin.SyncStatus(syncState),
	})
	fatalIf(probe.NewError(e).Trace(args...), "Unable to edit cluster replication site endpoint")

	printMsg(updateSuccessMessage(res))

	return nil
}
