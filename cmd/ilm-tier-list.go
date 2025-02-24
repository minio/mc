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
	"cmp"
	"fmt"
	"slices"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	madmin "github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminTierListCmd = cli.Command{
	Name:         "list",
	ShortName:    "ls",
	Usage:        "list configured remote tier targets",
	Action:       mainAdminTierList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. List remote tier targets configured on 'myminio':
     {{.Prompt}} {{.HelpName}} myminio
`,
}

// checkAdminTierListSyntax - validate all the passed arguments
func checkAdminTierListSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
	if argsNr > 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier-ls subcommand.")
	}
}

func storageClass(t *madmin.TierConfig) string {
	switch t.Type {
	case madmin.S3:
		return t.S3.StorageClass
	case madmin.Azure:
		return t.Azure.StorageClass
	case madmin.GCS:
		return t.GCS.StorageClass
	default:
		return ""
	}
}

type tierListMessage struct {
	Status  string               `json:"status"`
	Context *cli.Context         `json:"-"`
	Tiers   []*madmin.TierConfig `json:"tiers"`
}

// String method returns a tabular listing of remote tier configurations.
func (msg *tierListMessage) String() string {
	return "" // Not used in rendering; only to satisfy msg interface
}

// JSON method returns JSON encoding of msg.
func (msg *tierListMessage) JSON() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

func mainAdminTierList(ctx *cli.Context) error {
	checkAdminTierListSyntax(ctx)

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	tiers, e := client.ListTiers(globalContext)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to list configured remote tier targets")

	if len(tiers) == 0 {
		console.Infoln("No remote tier targets found for alias '" + aliasedURL + "'. Use `mc ilm tier add` to configure one.")
		return nil
	}

	if globalJSON {
		printMsg(&tierListMessage{
			Status:  "success",
			Context: ctx,
			Tiers:   tiers,
		})
		return nil
	}

	tableData := tierTable(tiers)
	slices.SortFunc(tableData, func(a, b *madmin.TierConfig) int {
		return cmp.Compare(a.Name, b.Name)
	})
	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		Headers(tableData.Headers()...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			switch {
			case row == 0:
				return lipgloss.NewStyle().Bold(true).Align(lipgloss.Center)
			case row%2 == 0:
				return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Align(lipgloss.Center)
			default:
				return lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Align(lipgloss.Center)
			}
		}).
		Data(tableData)
	fmt.Println(tbl)
	return nil
}

type tierTable []*madmin.TierConfig

var _ table.Data = tierTable(nil)

func (tt tierTable) Headers() []string {
	return []string{
		"Name",
		"Type",
		"Endpoint",
		"Bucket",
		"Prefix",
		"Region",
		"Storage-Class",
	}
}

func (tt tierTable) At(row, col int) string {
	tc := []*madmin.TierConfig(tt)
	cell := ""
	switch col {
	case 0:
		cell = tc[row].Name
	case 1:
		cell = tc[row].Type.String()
	case 2:
		cell = tc[row].Endpoint()
	case 3:
		cell = tc[row].Bucket()
	case 4:
		cell = tc[row].Prefix()
	case 5:
		cell = tc[row].Region()
	case 6:
		cell = storageClass(tc[row])
	}
	if cell == "" {
		return "-"
	}
	return cell
}

func (tt tierTable) Rows() int {
	return len(tt)
}

func (tt tierTable) Columns() int {
	return len(tt.Headers())
}
