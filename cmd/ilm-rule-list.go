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
	"context"
	"errors"
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/cmd/ilm"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

var ilmListFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "expiry",
		Usage: "display only expiration fields",
	},
	cli.BoolFlag{
		Name:  "transition",
		Usage: "display only transition fields",
	},
}

var ilmLsCmd = cli.Command{
	Name:         "list",
	ShortName:    "ls",
	Usage:        "lists lifecycle configuration rules set on a bucket",
	Action:       mainILMList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(ilmListFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  List lifecycle configuration rules set on a bucket.

EXAMPLES:
  1. List the lifecycle management rules (all fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. List the lifecycle management rules (expration date/days fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --expiry myminio/mybucket

  3. List the lifecycle management rules (transition date/days, storage class fields) for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --transition myminio/mybucket

  4. List the lifecycle management rules in JSON format for mybucket on alias 'myminio'.
     {{.Prompt}} {{.HelpName}} --json myminio/mybucket
`,
}

type ilmListMessage struct {
	Status    string                   `json:"status"`
	Target    string                   `json:"target"`
	Context   *cli.Context             `json:"-"`
	Config    *lifecycle.Configuration `json:"config"`
	UpdatedAt time.Time                `json:"updatedAt,omitempty"`
}

func (i ilmListMessage) String() string {
	// We don't use this method to display ilm-ls output. This is here to
	// implement the interface required by printMsg
	return ""
}

func (i ilmListMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// validateILMListFlagSet - validates ilm list flags
func validateILMListFlagSet(ctx *cli.Context) bool {
	expiryOnly := ctx.Bool("expiry")
	transitionOnly := ctx.Bool("transition")
	// Only one of expiry or transition rules can be filtered
	if expiryOnly && transitionOnly {
		return false
	}
	return true
}

// checkILMListSyntax - validate arguments passed by a user
func checkILMListSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, globalErrorExitStatus)
	}

	if !validateILMListFlagSet(ctx) {
		fatalIf(errInvalidArgument(), "only one display field flag is allowed per ls command. Refer mc "+ctx.Command.FullName()+" --help.")
	}
}

func mainILMList(cliCtx *cli.Context) error {
	ctx, cancelILMList := context.WithCancel(globalContext)
	defer cancelILMList()

	checkILMListSyntax(cliCtx)
	setILMDisplayColorScheme()

	args := cliCtx.Args()
	urlStr := args.Get(0)

	// Note: validateILMListFlagsSet ensures we deal with only valid
	// combinations here.
	var filter ilm.LsFilter
	if v := cliCtx.Bool("expiry"); v {
		filter = ilm.ExpiryOnly
	}
	if v := cliCtx.Bool("transition"); v {
		filter = ilm.TransitionOnly
	}
	client, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize client for "+urlStr)

	ilmCfg, updatedAt, err := client.GetLifecycle(ctx)
	fatalIf(err.Trace(args...), "Unable to get lifecycle")

	if len(ilmCfg.Rules) == 0 {
		fatalIf(probe.NewError(errors.New("lifecycle configuration not set")).Trace(urlStr),
			"Unable to ls lifecycle configuration")
	}

	// applies listing filter on ILM rules
	ilmCfg.Rules = filter.Apply(ilmCfg.Rules)

	if globalJSON {
		printMsg(ilmListMessage{
			Status:    "success",
			Target:    urlStr,
			Context:   cliCtx,
			Config:    ilmCfg,
			UpdatedAt: updatedAt,
		})
		return nil
	}

	for _, tbl := range ilm.ToTables(ilmCfg) {
		rows := tbl.Rows()
		if len(rows) == 0 {
			continue
		}
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		var colCfgs []table.ColumnConfig
		for i := 0; i < len(rows[0]); i++ {
			colCfgs = append(colCfgs, table.ColumnConfig{
				Align: text.AlignCenter,
			})
		}
		t.SetColumnConfigs(colCfgs)
		t.SetTitle(tbl.Title())
		t.AppendHeader(tbl.ColumnHeaders())
		t.AppendRows(rows)
		t.SetStyle(table.StyleLight)
		t.Render()
	}

	return nil
}
