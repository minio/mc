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
	"github.com/minio/pkg/console"
)

var ilmRemoveFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "id",
		Usage: "id of the lifecycle rule",
	},
	cli.BoolFlag{
		Name:  "force",
		Usage: "force flag is to be used when deleting all lifecycle configuration rules for the bucket",
	},
	cli.BoolFlag{
		Name:  "all",
		Usage: "delete all lifecycle configuration rules of the bucket, force flag enforced",
	},
}

var ilmRmCmd = cli.Command{
	Name:         "rm",
	Usage:        "remove (if any) existing lifecycle configuration rule",
	Action:       mainILMRemove,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(ilmRemoveFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  Remove a lifecycle configuration rule for the bucket by ID, optionally you can remove
  all the lifecycle rules on a bucket with '--all --force' option.

EXAMPLES:
  1. Remove the lifecycle management configuration rule given by ID "bgrt1ghju" for mybucket on alias 'myminio'. ID is case sensitive.
     {{.Prompt}} {{.HelpName}} --id "bgrt1ghju" myminio/mybucket

  2. Remove ALL the lifecycle management configuration rules for mybucket on alias 'myminio'.
     Because the result is complete removal, the use of --force flag is enforced.
     {{.Prompt}} {{.HelpName}} --all --force myminio/mybucket
`,
}

type ilmRmMessage struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Target string `json:"target"`
	All    bool   `json:"all"`
}

func (i ilmRmMessage) String() string {
	msg := "Rule ID `" + i.ID + "` from target " + i.Target + " removed."
	if i.All {
		msg = "Rules for `" + i.Target + "` removed."
	}
	return console.Colorize(ilmThemeResultSuccess, msg)
}

func (i ilmRmMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

func checkILMRemoveSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "rm", globalErrorExitStatus)
	}

	ilmAll := ctx.Bool("all")
	ilmForce := ctx.Bool("force")
	forceChk := (ilmAll && ilmForce) || (!ilmAll && !ilmForce)
	if !forceChk {
		fatalIf(errInvalidArgument(),
			"It is mandatory to specify --all and --force flag together for mc "+ctx.Command.FullName()+".")
	}
	if ilmAll && ilmForce {
		return
	}

	ilmID := ctx.String("id")
	if ilmID == "" {
		fatalIf(errInvalidArgument().Trace(ilmID), "ilm ID cannot be empty")
	}
}

func mainILMRemove(cliCtx *cli.Context) error {
	ctx, cancelILMImport := context.WithCancel(globalContext)
	defer cancelILMImport()

	checkILMRemoveSyntax(cliCtx)
	setILMDisplayColorScheme()
	args := cliCtx.Args()
	urlStr := args.Get(0)

	client, err := newClient(urlStr)
	fatalIf(err.Trace(args...), "Unable to initialize client for "+urlStr+".")

	ilmCfg, err := client.GetLifecycle(ctx)
	fatalIf(err.Trace(urlStr), "Unable to fetch lifecycle rules")

	ilmAll := cliCtx.Bool("all")
	ilmForce := cliCtx.Bool("force")

	if ilmAll && ilmForce {
		ilmCfg.Rules = nil // Remove all rules
	} else {
		ilmCfg, err = ilm.RemoveILMRule(ilmCfg, cliCtx.String("id"))
		fatalIf(err.Trace(urlStr, cliCtx.String("id")), "Unable to remove rule by id")
	}

	fatalIf(client.SetLifecycle(ctx, ilmCfg).Trace(urlStr), "Unable to set lifecycle rules")

	printMsg(ilmRmMessage{
		Status: "success",
		ID:     cliCtx.String("id"),
		All:    ilmAll,
		Target: urlStr,
	})

	return nil
}
