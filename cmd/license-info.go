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
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var licenseInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display license information",
	OnUsageError: onUsageError,
	Action:       mainLicenseInfo,
	Before:       setGlobalsFromContext,
	Flags:        append(supportGlobalFlags, subnetCommonFlags...),
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

type liTableModel struct {
	table table.Model
}

func (m liTableModel) Init() tea.Cmd { return nil }

func (m liTableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m liTableModel) View() string {
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).Render(m.table.View())
}

type licInfoMessage struct {
	Status string  `json:"status"`
	Info   licInfo `json:"info,omitempty"`
	Error  string  `json:"error,omitempty"`
}

type licInfo struct {
	Organization string     `json:"org,omitempty"`           // Subnet organization name
	Plan         string     `json:"plan,omitempty"`          // Subnet plan
	IssuedAt     *time.Time `json:"issued_at,omitempty"`     // Time of license issue
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`    // Time of license expiry
	DeploymentID string     `json:"deployment_id,omitempty"` // Cluster deployment ID
	Message      string     `json:"message,omitempty"`       // Message to be displayed
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
		{Title: "Field", Width: 20},
		{Title: "Value", Width: 45},
	}

	rows := []table.Row{
		{licInfoField("Organization"), licInfoVal(li.Organization)},
		{licInfoField("Deployment ID"), licInfoVal(li.DeploymentID)},
		{licInfoField("Plan"), licInfoVal(li.Plan)},
		{licInfoField("Issued at"), licInfoVal(li.IssuedAt.String())},
		{licInfoField("Expires at"), licInfoVal(li.ExpiresAt.String())},
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
	p := tea.NewProgram(liTableModel{t})
	go p.Start()
	time.Sleep(time.Second) // allow time for the table to be rendered
	p.Quit()

	return ""
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
		showCommandHelpAndExit(ctx, "info", 1) // last argument is exit code
	}

	initLicInfoColors()

	aliasedURL := ctx.Args().Get(0)
	alias, _ := initSubnetConnectivity(ctx, aliasedURL, false)

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
			Organization: li.Organization,
			Plan:         li.Plan,
			IssuedAt:     &li.IssuedAt,
			ExpiresAt:    &li.ExpiresAt,
			DeploymentID: li.DeploymentID,
		},
	}
}

func licErrMsg(e error) licInfoMessage {
	return licInfoMessage{
		Status: "error",
		Error:  e.Error(),
	}
}
