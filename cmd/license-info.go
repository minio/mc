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
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var licenseInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display license information",
	OnUsageError: onUsageError,
	Action:       mainLicenseInfo,
	Before:       setGlobalsFromContext,
	Flags:        subnetCommonFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Display license configuration for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play
`,
}

const (
	licInfoMsgTag   = "licenseInfoMessage"
	licInfoErrTag   = "licenseInfoError"
	licInfoFieldTag = "licenseInfoField"
	licInfoValTag   = "licenseValueField"
)

type licInfoMessage struct {
	Status string  `json:"status"`
	Info   licInfo `json:"info,omitempty"`
	Error  string  `json:"error,omitempty"`
}

type licInfo struct {
	LicenseID    string     `json:"license_id,omitempty"`    // Unique ID of the license
	Organization string     `json:"org,omitempty"`           // Subnet organization name
	Plan         string     `json:"plan,omitempty"`          // Subnet plan
	IssuedAt     *time.Time `json:"issued_at,omitempty"`     // Time of license issue
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`    // Time of license expiry
	DeploymentID string     `json:"deployment_id,omitempty"` // Cluster deployment ID
	Message      string     `json:"message,omitempty"`       // Message to be displayed
	APIKey       string     `json:"api_key,omitempty"`       // API Key of the org account
}

func licInfoField(s string) string {
	return console.Colorize(licInfoFieldTag, s)
}

func licInfoVal(s string) string {
	return console.Colorize(licInfoValTag, s)
}

func licInfoMsg(s string) string {
	return console.Colorize(licInfoMsgTag, s)
}

func licInfoErr(s string) string {
	return console.Colorize(licInfoErrTag, s)
}

// String colorized license info
func (li licInfoMessage) String() string {
	if len(li.Error) > 0 {
		return licInfoErr(li.Error)
	}

	if len(li.Info.Message) > 0 {
		return licInfoMsg(li.Info.Message)
	}

	return getLicInfoStr(li.Info)
}

// JSON jsonified license info
func (li licInfoMessage) JSON() string {
	jsonBytes, e := json.MarshalIndent(li, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

func getLicInfoStr(li licInfo) string {
	columns := []table.Column{
		{Title: "License", Width: 20},
		{Title: "", Width: 45},
	}

	rows := []table.Row{
		{licInfoField("Organization"), licInfoVal(li.Organization)},
		{licInfoField("Plan"), licInfoVal(li.Plan)},
		{licInfoField("Issued"), licInfoVal(li.IssuedAt.Format(http.TimeFormat))},
		{licInfoField("Expires"), licInfoVal(li.ExpiresAt.Format(http.TimeFormat))},
	}

	if len(li.LicenseID) > 0 {
		rows = append(rows, table.Row{licInfoField("License ID"), licInfoVal(li.LicenseID)})
	}
	if len(li.DeploymentID) > 0 {
		rows = append(rows, table.Row{licInfoField("Deployment ID"), licInfoVal(li.DeploymentID)})
	}
	if len(li.APIKey) > 0 {
		rows = append(rows, table.Row{licInfoField("API Key"), licInfoVal(li.APIKey)})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.Bold(false)
	t.SetStyles(s)

	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).Render(t.View())
}

func getAGPLMessage() string {
	return `License: GNU AGPL v3 <https://www.gnu.org/licenses/agpl-3.0.txt>
If you are distributing or hosting MinIO along with your proprietary application as combined works, you may require a commercial license included in the Standard and Enterprise subscription plans. (https://min.io/signup?ref=mc)`
}

func initLicInfoColors() {
	console.SetColor(licInfoMsgTag, color.New(color.FgGreen, color.Bold))
	console.SetColor(licInfoErrTag, color.New(color.FgRed, color.Bold))
	console.SetColor(licInfoFieldTag, color.New(color.FgCyan))
	console.SetColor(licInfoValTag, color.New(color.FgWhite))
}

func mainLicenseInfo(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	initLicInfoColors()

	aliasedURL := ctx.Args().Get(0)
	alias, _ := initSubnetConnectivity(ctx, aliasedURL, true)

	apiKey, lic, e := getSubnetCreds(alias)
	fatalIf(probe.NewError(e), "Error in checking cluster registration status")

	var lim licInfoMessage
	if len(lic) > 0 {
		lim = getLicInfoMsg(lic)
	} else if len(apiKey) > 0 {
		lim = licInfoMessage{
			Status: "success",
			Info: licInfo{
				Message: fmt.Sprintf("%s is registered with SUBNET. License info not available.", alias),
			},
		}
	} else {
		// Not registered. Default to AGPLv3
		lim = licInfoMessage{
			Status: "success",
			Info: licInfo{
				Plan:    "AGPLv3",
				Message: getAGPLMessage(),
			},
		}
	}

	printMsg(lim)
	return nil
}

func getLicInfoMsg(lic string) licInfoMessage {
	li, e := parseLicense(lic)
	if e != nil {
		return licErrMsg(e)
	}
	return licInfoMessage{
		Status: "success",
		Info: licInfo{
			LicenseID:    li.LicenseID,
			Organization: li.Organization,
			Plan:         li.Plan,
			IssuedAt:     &li.IssuedAt,
			ExpiresAt:    &li.ExpiresAt,
			DeploymentID: li.DeploymentID,
			APIKey:       li.APIKey,
		},
	}
}

func licErrMsg(e error) licInfoMessage {
	return licInfoMessage{
		Status: "error",
		Error:  e.Error(),
	}
}
