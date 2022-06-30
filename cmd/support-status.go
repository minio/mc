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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var supportStatusCmd = cli.Command{
	Name:         "status",
	Usage:        "Display support configuration",
	OnUsageError: onUsageError,
	Action:       mainSupportStatus,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

EXAMPLES:
  1. Display support configuration for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play
`,
}

type supportStatusMessage struct {
	Status         string `json:"status"`
	Registered     bool   `json:"registered"`
	LogsStatus     string `json:"logs"`
	CallhomeStatus string `json:"callhome"`
}

// String colorized status message.
func (s supportStatusMessage) String() string {
	str := "Cluster is "
	if !s.Registered {
		str += "not "
	}
	str += fmt.Sprintln("registered with SUBNET")
	str += fmt.Sprintln("Logs:", s.LogsStatus)
	str += fmt.Sprint("Callhome: ", s.CallhomeStatus)

	return console.Colorize(featureStatusMessageTag, str)
}

// JSON jsonified status message.
func (s supportStatusMessage) JSON() string {
	s.Status = "success"
	jsonBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

func mainSupportStatus(ctx *cli.Context) error {
	console.SetColor(featureStatusMessageTag, color.New(color.FgGreen, color.Bold))

	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "status", 1) // last argument is exit code
	}

	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)

	apiKey, lic, e := getSubnetCreds(alias)
	fatalIf(probe.NewError(e), "Error in checking cluster registration status")

	ssm := supportStatusMessage{
		Registered:     len(apiKey) > 0 || len(lic) > 0,
		LogsStatus:     featureStatusStr(isSupportLogsEnabled(alias)),
		CallhomeStatus: featureStatusStr(isSupportCallhomeEnabled(alias)),
	}

	printMsg(ssm)
	return nil
}
