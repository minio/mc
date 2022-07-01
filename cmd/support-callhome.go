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
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var supportCallhomeCmd = cli.Command{
	Name:         "callhome",
	Usage:        "configure callhome settings",
	OnUsageError: onUsageError,
	Action:       mainCallhome,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} enable|disable|status ALIAS

OPTIONS:
  enable - Enable pushing callhome info to SUBNET every 24hrs
  disable - Disable pushing callhome info to SUBNET
  status - Display callhome settings

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Enable callhome for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} enable play

  2. Disable callhome for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} disable play

  3. Check callhome status for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} status play
`,
}

type supportCallhomeMessage struct {
	Status   string `json:"status"`
	Callhome string `json:"callhome"`
	MsgPfx   string `json:"-"`
}

// String colorized service status message.
func (s supportCallhomeMessage) String() string {
	return console.Colorize(featureToggleMessageTag, s.MsgPfx+s.Callhome)
}

// JSON jsonified service status message.
func (s supportCallhomeMessage) JSON() string {
	s.Status = "success"
	jsonBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

func isSupportCallhomeEnabled(alias string) bool {
	return isFeatureEnabled(alias, "callhome", "callhome")
}

func mainCallhome(ctx *cli.Context) error {
	setToggleMessageColor()
	alias, arg := checkToggleCmdSyntax(ctx, "callhome")

	if arg == "status" {
		printMsg(supportCallhomeMessage{
			Callhome: featureStatusStr(isSupportCallhomeEnabled(alias)),
		})
		return nil
	}

	setCallhomeConfig(alias, arg == "enable")

	return nil
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
		validateClusterRegistered(alias)
		enableStr = "on"
	}
	configStr := "callhome enable=" + enableStr
	_, e := client.SetConfigKV(globalContext, configStr)
	fatalIf(probe.NewError(e), "Unable to set callhome config on minio")

	printMsg(supportCallhomeMessage{
		Callhome: featureStatusStr(enableCallhome),
		MsgPfx:   "Callhome is now ",
	})
}
