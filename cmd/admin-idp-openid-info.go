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

	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPOpenidInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "get OpenID IDP server configuration info",
	Action:       mainAdminIDPOpenIDInfo,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [CFG_NAME]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get configuration info on the default OpenID IDP configuration (CFG_NAME is omitted).
     {{.Prompt}} {{.HelpName}} play/
  2. Get configuration info on OpenID IDP configuration named "dex_test".
     {{.Prompt}} {{.HelpName}} play/ dex_test
`,
}

func mainAdminIDPOpenIDInfo(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 || len(ctx.Args()) > 2 {
		showCommandHelpAndExit(ctx, 1)
	}

	args := ctx.Args()
	var cfgName string
	if len(args) == 2 {
		cfgName = args.Get(1)
	}

	return adminIDPInfo(ctx, true, cfgName)
}

func adminIDPInfo(ctx *cli.Context, isOpenID bool, cfgName string) error {
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	idpType := madmin.LDAPIDPCfg
	if isOpenID {
		idpType = madmin.OpenidIDPCfg
	}

	result, e := client.GetIDPConfig(globalContext, idpType, cfgName)
	fatalIf(probe.NewError(e), "Unable to get %s IDP config from server", idpType)

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
