// Copyright (c) 2022 MinIO, Inc.
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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminServiceUnfreezeCmd = cli.Command{
	Name:         "unfreeze",
	Usage:        "unfreeze S3 API calls on MinIO cluster",
	Action:       mainAdminServiceUnfreeze,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Unfreeze all S3 API calls on MinIO server at 'myminio/'.
     {{.Prompt}} {{.HelpName}} myminio/
`,
}

// serviceUnfreezeCommand is container for service unfreeze command success and failure messages.
type serviceUnfreezeCommand struct {
	Status    string `json:"status"`
	ServerURL string `json:"serverURL"`
}

// String colorized service unfreeze command message.
func (s serviceUnfreezeCommand) String() string {
	return console.Colorize("ServiceUnfreeze", "Unfreeze command successfully sent to `"+s.ServerURL+"`.")
}

// JSON jsonified service unfreeze command message.
func (s serviceUnfreezeCommand) JSON() string {
	serviceUnfreezeJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(serviceUnfreezeJSONBytes)
}

// checkAdminServiceUnfreezeSyntax - validate all the passed arguments
func checkAdminServiceUnfreezeSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainAdminServiceUnfreeze(ctx *cli.Context) error {
	// Validate service unfreeze syntax.
	checkAdminServiceUnfreezeSyntax(ctx)

	// Set color.
	console.SetColor("ServiceUnfreeze", color.New(color.FgGreen, color.Bold))
	console.SetColor("FailedServiceUnfreeze", color.New(color.FgRed, color.Bold))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	// Unfreeze the specified MinIO server
	e := client.ServiceUnfreezeV2(ctxt)
	if e != nil {
		// Attempt an older API server might be old
		// nolint:staticcheck
		// we need this fallback
		e = client.ServiceUnfreeze(ctxt)
	}
	// Unfreeze the specified MinIO server
	fatalIf(probe.NewError(e), "Unable to unfreeze the server.")

	// Success..
	printMsg(serviceUnfreezeCommand{Status: "success", ServerURL: aliasedURL})

	return nil
}
