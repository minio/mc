// Copyright (c) 2015-2023 MinIO, Inc.
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
)

var idpOpenidAccesskeyInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "info about given access key pairs for OpenID",
	Action:       mainIDPOpenIDAccesskeyInfo,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	OnUsageError: onUsageError,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET ACCESSKEY [ACCESSKEY...]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Get info for the access key "testkey"
	 {{.Prompt}} {{.HelpName}} local/ testkey
  2. Get info for the access keys "testkey" and "testkey2"
	 {{.Prompt}} {{.HelpName}} local/ testkey testkey2
	`,
}

type openIDAccessKeyInfo struct {
	ConfigName       string `json:"configName"`
	UserID           string `json:"userID"`
	UserIDClaim      string `json:"userIDClaim"`
	DisplayName      string `json:"displayName,omitempty"`
	DisplayNameClaim string `json:"displayNameClaim,omitempty"`
}

func (l openIDAccessKeyInfo) String() string {
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")) // green
	o := strings.Builder{}
	if l.ConfigName == "_" {
		l.ConfigName = "_ (default)"
	}
	o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render("Config:"), l.ConfigName))
	if l.DisplayNameClaim != "" {
		dispNameLabel := fmt.Sprintf("Display Name (%s):", l.DisplayNameClaim)
		o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render(dispNameLabel), l.DisplayName))
	}
	uidLabel := fmt.Sprintf("User ID (%s):", l.UserIDClaim)
	o.WriteString(iFmt(2, "%s %s\n", labelStyle.Render(uidLabel), l.UserID))
	return o.String()
}

func mainIDPOpenIDAccesskeyInfo(ctx *cli.Context) error {
	return commonAccesskeyInfo(ctx)
}
