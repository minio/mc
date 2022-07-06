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
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "Show IDP server config info",
	Before:       setGlobalsFromContext,
	Action:       mainAdminIDPGet,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ID_TYPE [CFG_NAME]

FLAGS:
   {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show configuration info for default (un-named) openid configuration.
     {{.Prompt}} {{.HelpName}} play/ openid
  2. Show configuration info for openid configuration named "dex_test".
     {{.Prompt}} {{.HelpName}} play/ openid dex_test
`,
}

func mainAdminIDPGet(ctx *cli.Context) error {
	if len(ctx.Args()) < 2 || len(ctx.Args()) > 3 {
		cli.ShowCommandHelpAndExit(ctx, "get", 1)
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	idpType := args.Get(1)

	if idpType != "openid" {
		fatalIf(probe.NewError(errors.New("not implemented")), "This feature is not yet available")
	}

	var cfgName string
	if len(args) == 3 {
		cfgName = args.Get(2)
	}

	result, e := client.GetIDPConfig(globalContext, idpType, cfgName)
	fatalIf(probe.NewError(e), "Unable to get IDP config for '%s' to server", idpType)

	// Print set config result
	printMsg(idpConfig(result))

	return nil
}

type idpConfig madmin.IDPConfig

func (i idpConfig) JSON() string {
	bs, e := json.MarshalIndent(i, "", "  ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(bs)
}

func (i idpConfig) String() string {
	// Determine required width for key column.
	fieldColWidth := 0
	for _, kv := range i.Info {
		if fieldColWidth < len(kv.Key) {
			fieldColWidth = len(kv.Key)
		}
	}
	// Add 1 for the colon-suffix in each entry.
	fieldColWidth++

	fieldColStyle := lipgloss.NewStyle().
		Width(fieldColWidth).
		Foreground(lipgloss.Color("#04B575")). // green
		Bold(true).
		Align(lipgloss.Right)
	valueColStyle := lipgloss.NewStyle().
		PaddingLeft(1).
		Align(lipgloss.Left)
	envMarkStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("201")). // pinkish-red
		PaddingLeft(1)

	var lines []string
	for _, kv := range i.Info {
		envStr := ""
		if kv.IsCfg && kv.IsEnv {
			envStr = " (environment)"
		}
		lines = append(lines, fmt.Sprintf("%s%s%s",
			fieldColStyle.Render(kv.Key+":"),
			valueColStyle.Render(kv.Value),
			envMarkStyle.Render(envStr),
		))
	}

	boxContent := strings.Join(lines, "\n")

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder())

	return boxStyle.Render(boxContent)
}
