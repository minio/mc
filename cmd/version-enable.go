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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var versionEnableCmd = cli.Command{
	Name:         "enable",
	Usage:        "enable bucket versioning",
	Action:       mainVersionEnable,
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
  1. Enable versioning on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkVersionEnableSyntax - validate all the passed arguments
func checkVersionEnableSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "enable", 1) // last argument is exit code
	}
}

type versionEnableMessage struct {
	Op         string
	Status     string `json:"status"`
	URL        string `json:"url"`
	Versioning struct {
		Status    string `json:"status"`
		MFADelete string `json:"MFADelete"`
	} `json:"versioning"`
}

func (v versionEnableMessage) JSON() string {
	v.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(v, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (v versionEnableMessage) String() string {
	return console.Colorize("versionEnableMessage", fmt.Sprintf("%s versioning is enabled", v.URL))
}

func mainVersionEnable(cliCtx *cli.Context) error {
	ctx, cancelVersionEnable := context.WithCancel(globalContext)
	defer cancelVersionEnable()

	console.SetColor("versionEnableMessage", color.New(color.FgGreen))

	checkVersionEnableSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	fatalIf(client.SetVersion(ctx, "enable"), "Unable to enable versioning")
	printMsg(versionEnableMessage{
		Op:     "enable",
		Status: "success",
		URL:    aliasedURL,
	})
	return nil
}
