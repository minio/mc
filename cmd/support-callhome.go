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

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var supportCallhomeFlags = append([]cli.Flag{
	cli.BoolFlag{
		Name:  "logs",
		Usage: "push logs to SUBNET in real-time",
	},
	cli.BoolFlag{
		Name:  "diag",
		Usage: "push diagnostics info to SUBNET every 24hrs",
	},
}, supportGlobalFlags...)

var supportCallhomeCmd = cli.Command{
	Name:         "callhome",
	Usage:        "configure callhome settings",
	OnUsageError: onUsageError,
	Action:       mainCallhome,
	Before:       setGlobalsFromContext,
	Flags:        supportCallhomeFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} enable|disable|status ALIAS

OPTIONS:
  enable - Enable callhome
  disable - Disable callhome
  status - Display callhome settings

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Enable callhome for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} enable myminio

  2. Disable callhome for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} disable myminio

  3. Check callhome status for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} status myminio

  4. Enable diagnostics callhome for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} enable myminio --diag

  5. Disable logs callhome for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} disable myminio --logs

  6. Check logs callhome status for cluster with alias 'myminio'
     {{.Prompt}} {{.HelpName}} status myminio --logs
`,
}

type supportCallhomeMessage struct {
	Status  string `json:"status"`
	Diag    string `json:"diag,omitempty"`
	Logs    string `json:"logs,omitempty"`
	Feature string `json:"-"`
	Action  string `json:"-"`
}

// String colorized callhome command output message.
func (s supportCallhomeMessage) String() string {
	if s.Action == "status" {
		columns := []table.Column{
			{Title: "Features", Width: 20},
			{Title: "", Width: 15},
		}

		rows := []table.Row{}

		if len(s.Diag) > 0 {
			rows = append(rows, table.Row{licInfoField("Diagnostics"), licInfoVal(s.Diag)})
		}
		if len(s.Logs) > 0 {
			rows = append(rows, table.Row{licInfoField("Logs"), licInfoVal(s.Logs)})
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

	return console.Colorize(supportSuccessMsgTag, s.Feature+" is now "+s.Action)
}

// JSON jsonified callhome command output message.
func (s supportCallhomeMessage) JSON() string {
	s.Status = "success"
	jsonBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

func isDiagCallhomeEnabled(alias string) bool {
	return isFeatureEnabled(alias, "callhome", madmin.Default)
}

func mainCallhome(ctx *cli.Context) error {
	initLicInfoColors()

	setSuccessMessageColor()
	alias, arg := checkToggleCmdSyntax(ctx)
	apiKey := validateClusterRegistered(alias, true)

	diag, logs := parseCallhomeFlags(ctx)

	if arg == "status" {
		printCallhomeStatus(alias, diag, logs)
		return nil
	}

	toggleCallhome(alias, apiKey, arg == "enable", diag, logs)

	return nil
}

func parseCallhomeFlags(ctx *cli.Context) (diag, logs bool) {
	diag = ctx.Bool("diag")
	logs = ctx.Bool("logs")

	if !diag && !logs {
		// When both flags are not passed, apply the action to both
		diag = true
		logs = true
	}

	return diag, logs
}

func printCallhomeStatus(alias string, diag, logs bool) {
	resultMsg := supportCallhomeMessage{Action: "status"}
	if diag {
		resultMsg.Diag = featureStatusStr(isDiagCallhomeEnabled(alias))
	}

	if logs {
		resultMsg.Logs = featureStatusStr(isLogsCallhomeEnabled(alias))
	}
	printMsg(resultMsg)
}

func toggleCallhome(alias, apiKey string, enable, diag, logs bool) {
	newStatus := featureStatusStr(enable)
	resultMsg := supportCallhomeMessage{
		Action:  newStatus,
		Feature: getFeature(diag, logs),
	}

	if diag {
		setCallhomeConfig(alias, enable)
		resultMsg.Diag = newStatus
	}

	if logs {
		configureSubnetWebhook(alias, apiKey, enable)
		resultMsg.Logs = newStatus
	}

	printMsg(resultMsg)
}

func getFeature(diag, logs bool) string {
	if diag && logs {
		return "Diagnostics and logs callhome"
	}

	if diag {
		return "Diagnostics"
	}

	return "Logs"
}

func setCallhomeConfig(alias string, enableCallhome bool) {
	// Create a new MinIO Admin Client
	client, err := newAdminClient(alias)
	fatalIf(err, "Unable to initialize admin connection.")

	if !minioConfigSupportsSubSys(client, "callhome") {
		fatal(errDummy().Trace(), "Your version of MinIO doesn't support this configuration")
	}

	enableStr := "off"
	if enableCallhome {
		enableStr = "on"
	}
	configStr := "callhome enable=" + enableStr
	_, e := client.SetConfigKV(globalContext, configStr)
	fatalIf(probe.NewError(e), "Unable to set callhome config on minio")
}

func configureSubnetWebhook(alias, apiKey string, enable bool) {
	// Create a new MinIO Admin Client
	client, err := newAdminClient(alias)
	fatalIf(err, "Unable to initialize admin connection.")

	var input string
	if enable {
		input = fmt.Sprintf("logger_webhook:subnet endpoint=%s auth_token=%s enable=on",
			subnetLogWebhookURL(), apiKey)
	} else {
		input = "logger_webhook:subnet enable=off"
	}

	// Call set config API
	_, e := client.SetConfigKV(globalContext, input)
	fatalIf(probe.NewError(e), "Unable to set '%s' to server", input)
}

func isLogsCallhomeEnabled(alias string) bool {
	return isFeatureEnabled(alias, "logger_webhook", "subnet")
}
