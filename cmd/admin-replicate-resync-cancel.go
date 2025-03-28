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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminReplicateResyncCancelCmd = cli.Command{
	Name:         "cancel",
	Usage:        "cancel ongoing resync operation",
	Action:       mainAdminReplicateResyncCancel,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS1 ALIAS2

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Cancel ongoing resync of bucket data from minio1 to minio2
     {{.Prompt}} {{.HelpName}} minio1 minio2
`,
}

type resyncCancelMessage madmin.SRResyncOpStatus

func (m resyncCancelMessage) JSON() string {
	bs, e := json.MarshalIndent(madmin.SRResyncOpStatus(m), "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(bs)
}

func (m resyncCancelMessage) String() string {
	v := madmin.SRResyncOpStatus(m)
	messages := []string{}
	th := "ResyncMessage"
	if v.ErrDetail != "" {
		messages = append(messages, v.ErrDetail)
		th = "ResyncErr"
	} else {
		messages = append(messages, fmt.Sprintf("Site resync with ID %s canceled successfully.", v.ResyncID))
	}
	return console.Colorize(th, strings.Join(messages, "\n"))
}

func mainAdminReplicateResyncCancel(ctx *cli.Context) error {
	// Check argument count
	argsNr := len(ctx.Args())
	if argsNr != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	console.SetColor("ResyncMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")
	info, e := client.SiteReplicationInfo(globalContext)
	fatalIf(probe.NewError(e), "Unable to fetch site replication info.")

	peerClient := getClient(args.Get(1))
	peerAdmInfo, e := peerClient.ServerInfo(globalContext)
	fatalIf(probe.NewError(e), "Unable to fetch server info of the peer.")

	var peer madmin.PeerInfo
	for _, site := range info.Sites {
		if peerAdmInfo.DeploymentID == site.DeploymentID {
			peer = site
		}
	}
	if peer.DeploymentID == "" {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"alias provided is not part of cluster replication.")
	}
	res, e := client.SiteReplicationResyncOp(globalContext, peer, madmin.SiteResyncCancel)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to cancel replication resync")

	printMsg(resyncCancelMessage(res))

	return nil
}
