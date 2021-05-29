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
	"context"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/cmd/ilm"
	"github.com/minio/mc/pkg/probe"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/pkg/console"
)

var ilmEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "modify a lifecycle configuration rule with given id",
	Action:       mainILMEdit,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(ilmEditFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  Modify a lifecycle configuration rule with given id.

EXAMPLES:
  1. Modify the expiration date for an existing rule with id "rHTY.a123".
     {{.Prompt}} {{.HelpName}} --id "rHTY.a123" --expiry-date "2020-09-17" s3/mybucket

  2. Modify the expiration and transition days for an existing rule with id "hGHKijqpo123".
     {{.Prompt}} {{.HelpName}} --id "hGHKijqpo123" --expiry-days "300" \
          --transition-days "200" --storage-class "GLACIER" s3/mybucket
`,
}

var ilmEditFlags = append(
	// Start by showing --id in edit command
	[]cli.Flag{
		cli.StringFlag{
			Name:  "id",
			Usage: "id of the rule to be modified",
		},
	},
	ilmAddFlags...,
)

type ilmEditMessage struct {
	Status string `json:"status"`
	Target string `json:"target"`
	ID     string `json:"id"`
}

func (i ilmEditMessage) String() string {
	return console.Colorize(ilmThemeResultSuccess, "Lifecycle configuration rule with ID `"+i.ID+"` modified  to "+i.Target+".")
}

func (i ilmEditMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Validate user given arguments
func checkILMEditSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "edit", globalErrorExitStatus)
	}
	id := ctx.String("id")
	if id == "" {
		fatalIf(errInvalidArgument(), "ID for lifecycle rule cannot be empty, please refer mc "+ctx.Command.FullName()+" --help for more details")
	}
}

// Calls SetBucketLifecycle with the XML representation of lifecycleConfiguration type.
func mainILMEdit(cliCtx *cli.Context) error {
	ctx, cancelILMEdit := context.WithCancel(globalContext)
	defer cancelILMEdit()

	checkILMEditSyntax(cliCtx)
	setILMDisplayColorScheme()
	args := cliCtx.Args()
	urlStr := args.Get(0)

	client, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize client for "+urlStr)

	// Configuration that is already set.
	lfcCfg, err := client.GetLifecycle(ctx)
	if err != nil {
		if e := err.ToGoError(); minio.ToErrorResponse(e).Code == "NoSuchLifecycleConfiguration" {
			lfcCfg = lifecycle.NewConfiguration()
		} else {
			fatalIf(err.Trace(args...), "Unable to fetch lifecycle rules for "+urlStr)
		}
	}

	// Configuration that needs to be set is returned by ilm.GetILMConfigToSet.
	// A new rule is added or the rule (if existing) is replaced
	opts := ilm.GetLifecycleOptions(cliCtx)
	lfcCfg, err = opts.ToConfig(lfcCfg)
	fatalIf(err.Trace(args...), "Unable to generate new lifecycle rules for the input")

	fatalIf(client.SetLifecycle(ctx, lfcCfg).Trace(urlStr), "Unable to set new lifecycle rules")

	printMsg(ilmEditMessage{
		Status: "success",
		Target: urlStr,
		ID:     opts.ID,
	})

	return nil
}
