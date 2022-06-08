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
	"encoding/json"
	"errors"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
)

var adminIDPLsCmd = cli.Command{
	Name:         "ls",
	Usage:        "List IDP configurations",
	Before:       setGlobalsFromContext,
	Action:       mainAdminIDPList,
	OnUsageError: onUsageError,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ID_TYPE

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. List the available openid configurations.
     {{.Prompt}} {{.HelpName}} play/ openid
`,
}

func mainAdminIDPList(ctx *cli.Context) error {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "ls", 1)
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

	result, e := client.ListIDPConfig(globalContext, idpType)
	fatalIf(probe.NewError(e), "Unable to list IDP config for '%s'", idpType)

	printMsg(idpCfgList(result))

	return nil
}

type idpCfgList []madmin.IDPListItem

func (i idpCfgList) JSON() string {
	bs, e := json.MarshalIndent(i, "", "  ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(bs)
}

func (i idpCfgList) String() string {
	maxNameWidth := len("Name")
	maxRoleARNWidth := len("RoleArn")
	for _, item := range i {
		name := item.Name
		if name == "_" {
			name = "(default)" // for the un-named config, don't show `_`
		}
		if maxNameWidth < len(name) {
			maxNameWidth = len(name)
		}
		if maxRoleARNWidth < len(item.RoleARN) {
			maxRoleARNWidth = len(item.RoleARN)
		}
	}
	enabledWidth := 5
	// Add 2 for padding
	maxNameWidth += 2
	maxRoleARNWidth += 2

	enabledColStyle := lipgloss.NewStyle().
		Align(lipgloss.Center).
		PaddingLeft(1).
		PaddingRight(1).
		Width(enabledWidth)
	nameColStyle := lipgloss.NewStyle().
		Align(lipgloss.Right).
		PaddingLeft(1).
		PaddingRight(1).
		Width(maxNameWidth)
	arnColStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		PaddingLeft(1).
		PaddingRight(1).
		Foreground(lipgloss.Color("#04B575")). // green
		Width(maxRoleARNWidth)

	styles := []lipgloss.Style{enabledColStyle, nameColStyle, arnColStyle}

	headers := []string{"On?", "Name", "RoleARN"}
	headerRow := []string{}

	// Override some style settings for the header
	for ii, hdr := range headers {
		headerRow = append(headerRow,
			styles[ii].Copy().
				Bold(true).
				Foreground(lipgloss.Color("#6495ed")). // green
				Align(lipgloss.Center).
				Render(hdr),
		)
	}

	lines := []string{strings.Join(headerRow, "")}

	enabledOff := "🔴"
	enabledOn := "🟢"

	for _, item := range i {
		enabled := enabledOff
		if item.Enabled {
			enabled = enabledOn
		}

		line := []string{
			styles[0].Render(enabled),
			styles[1].Render(item.Name),
			styles[2].Render(item.RoleARN),
		}
		if item.Name == "_" {
			// For default config, don't display `_` and make it look faint.
			line[1] = styles[1].Copy().
				Faint(true).
				Render("(default)")
		}
		lines = append(lines, strings.Join(line, ""))
	}

	boxContent := strings.Join(lines, "\n")
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder())

	return boxStyle.Render(boxContent)
}
