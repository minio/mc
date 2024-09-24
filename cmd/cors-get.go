// Copyright (c) 2015-2024 MinIO, Inc.
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
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/pkg/v3/console"
)

var corsGetCmd = cli.Command{
	Name:         "get",
	Usage:        "get a bucket CORS configuration",
	Action:       mainCorsGet,
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
  1. Get the CORS configuration for the bucket 'mybucket':
     {{.Prompt}} {{.HelpName}} myminio/mybucket
 `,
}

// checkCorsGetSyntax - validate all the passed arguments
func checkCorsGetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainCorsGet is the handle for "mc cors get" command.
func mainCorsGet(ctx *cli.Context) error {
	checkCorsGetSyntax(ctx)

	console.SetColor("CorsMessage", color.New(color.FgGreen))
	console.SetColor("CorsNotFound", color.New(color.FgYellow))

	// args[0] is the ALIAS/BUCKET argument.
	args := ctx.Args()
	urlStr := args.Get(0)

	client, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize client for "+urlStr)

	corsCfg, err := client.GetBucketCors(globalContext)
	fatalIf(err.Trace(urlStr), "Unable to get bucket CORS configuration for "+urlStr)

	status := "success"
	if corsCfg == nil {
		status = "not found"
	}

	printMsg(corsMessage{
		op:      "get",
		Status:  status,
		CorsCfg: corsCfg,
	})

	return nil
}
