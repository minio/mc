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
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var supportCallhomeCmd = cli.Command{
	Name:         "callhome",
	Usage:        "configure callhome settings",
	OnUsageError: onUsageError,
	Action:       mainCallhome,
	Before:       setGlobalsFromContext,
	Flags:        supportGlobalFlags,
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
`,
}

type supportCallhomeMessage struct {
	Status string `json:"status"`
}

// String colorized callhome command output message.
func (s supportCallhomeMessage) String() string {
	return console.Colorize(supportSuccessMsgTag, "Callhome is "+s.Status)
}

// JSON jsonified callhome command output message.
func (s supportCallhomeMessage) JSON() string {
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
	validateClusterRegistered(alias, false)

	if arg == "status" {
		printCallhomeStatus(alias)
		return nil
	}

	toggleCallhome(alias, arg)

	return nil
}

func printCallhomeStatus(alias string) {
	resultMsg := supportCallhomeMessage{Status: featureStatusStr(isDiagCallhomeEnabled(alias))}
	printMsg(resultMsg)
}

func toggleCallhome(alias, arg string) {
	enable := arg == "enable"
	setCallhomeConfig(alias, enable)
	resultMsg := supportCallhomeMessage{Status: featureStatusStr(enable)}
	printMsg(resultMsg)
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
