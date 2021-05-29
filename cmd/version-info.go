// Copyright (c) 2015-2021 MinIO, Inc.
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
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var versionInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "show bucket versioning status",
	Action:       mainVersionInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS/BUCKET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Display bucket versioning status for bucket "mybucket".
      {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkVersionInfoSyntax - validate all the passed arguments
func checkVersionInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "info", 1) // last argument is exit code
	}
}

type versioningInfoMessage struct {
	Op         string
	Status     string `json:"status"`
	URL        string `json:"url"`
	Versioning struct {
		Status    string `json:"status"`
		MFADelete string `json:"MFADelete"`
	} `json:"versioning"`
}

func (v versioningInfoMessage) JSON() string {
	v.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(v, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (v versioningInfoMessage) String() string {
	msg := ""
	switch v.Versioning.Status {
	case "":
		msg = fmt.Sprintf("%s is un-versioned", v.URL)
	default:
		msg = fmt.Sprintf("%s versioning is %s", v.URL, strings.ToLower(v.Versioning.Status))
	}
	return console.Colorize("versioningInfoMessage", msg)
}

func mainVersionInfo(cliCtx *cli.Context) error {
	ctx, cancelVersioningInfo := context.WithCancel(globalContext)
	defer cancelVersioningInfo()

	console.SetColor("versioningInfoMessage", color.New(color.FgGreen))

	checkVersionInfoSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	vConfig, e := client.GetVersion(ctx)
	fatalIf(e, "Unable to get versioning info")
	vMsg := versioningInfoMessage{
		Op:     "info",
		Status: "success",
		URL:    aliasedURL,
	}
	vMsg.Versioning.Status = vConfig.Status
	vMsg.Versioning.MFADelete = vConfig.MFADelete
	printMsg(vMsg)
	return nil
}
