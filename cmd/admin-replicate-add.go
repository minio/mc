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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminReplicateAddFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "replicate-ilm-expiry",
		Usage: "replicate ILM expiry rules",
	},
}

var adminReplicateAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add one or more sites for replication",
	Action:       mainAdminReplicateAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminReplicateAddFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS1 ALIAS2 [ALIAS3...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Add a site for cluster-level replication:
     {{.Prompt}} {{.HelpName}} minio1 minio2

  2. Add a site for cluster-level replication with replication of ILM expiry rules:
     {{.Prompt}} {{.HelpName}} minio1 minio2 --replicate-ilm-expiry
`,
}

type successMessage madmin.ReplicateAddStatus

func (m successMessage) JSON() string {
	bs, e := json.MarshalIndent(madmin.ReplicateAddStatus(m), "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(bs)
}

func (m successMessage) String() string {
	v := madmin.ReplicateAddStatus(m)
	messages := []string{v.Status}

	if v.ErrDetail != "" {
		messages = append(messages, v.ErrDetail)
	}
	if v.InitialSyncErrorMessage != "" {
		messages = append(messages, v.InitialSyncErrorMessage)
	}
	return console.Colorize("UserMessage", strings.Join(messages, "\n"))
}

func mainAdminReplicateAdd(ctx *cli.Context) error {
	{
		// Check argument count
		argsNr := len(ctx.Args())
		if argsNr < 2 {
			fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
				"Need at least two arguments to add command.")
		}
	}

	console.SetColor("UserMessage", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	ps := make([]madmin.PeerSite, 0, len(ctx.Args()))
	for _, clusterName := range ctx.Args() {
		admClient, err := newAdminClient(clusterName)
		fatalIf(err, "unable to initialize admin connection")

		ak, sk := admClient.GetAccessAndSecretKey()
		ps = append(ps, madmin.PeerSite{
			Name:      clusterName,
			Endpoint:  admClient.GetEndpointURL().String(),
			AccessKey: ak,
			SecretKey: sk,
		})
	}

	var opts madmin.SRAddOptions
	opts.ReplicateILMExpiry = ctx.Bool("replicate-ilm-expiry")
	res, e := client.SiteReplicationAdd(globalContext, ps, opts)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to add sites for replication")

	printMsg(successMessage(res))

	return nil
}
