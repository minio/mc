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
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminReplicateInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "get site replication information",
	Action:       mainAdminReplicationInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS1

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Get Site Replication information:
     {{.Prompt}} {{.HelpName}} minio1
`,
}

type srInfo madmin.SiteReplicationInfo

func (i srInfo) JSON() string {
	bs, e := json.MarshalIndent(madmin.SiteReplicationInfo(i), "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(bs)
}

func (i srInfo) String() string {
	var messages []string
	info := madmin.SiteReplicationInfo(i)
	if info.Enabled {
		messages = []string{
			"SiteReplication: on",
			fmt.Sprintf("ServiceAccountAccessKey: %s", info.ServiceAccountAccessKey),
			"SiteReplicationMembers:",
		}
		for _, peer := range info.Sites {
			messages = append(messages, fmt.Sprintf("  Name: %s, Endpoint: %s, DeploymentID: %s", peer.Name, peer.Endpoint, peer.DeploymentID))
		}
	} else {
		messages = []string{"SiteReplication: off"}
	}

	return console.Colorize("UserMessage", strings.Join(messages, "\n"))
}

func mainAdminReplicationInfo(ctx *cli.Context) error {
	{
		// Check argument count
		argsNr := len(ctx.Args())
		if argsNr != 1 {
			fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
				"Need exactly one alias argument.")
		}
	}

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	info, e := client.SiteReplicationInfo(globalContext)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get cluster replication information")

	printMsg(srInfo(info))

	return nil
}
