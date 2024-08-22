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
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/cors"
	"github.com/minio/pkg/v3/console"
)

var corsSetCmd = cli.Command{
	Name:         "set",
	Usage:        "set a bucket CORS configuration",
	Action:       mainCorsSet,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS/BUCKET CORSFILE

CORSFILE:
  Path to the XML file containing the CORS configuration.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set the CORS configuration for the bucket 'mybucket':
     {{.Prompt}} {{.HelpName}} myminio/mybucket /path/to/cors.xml

  2. Set the CORS configuration for the bucket 'mybucket' using stdin:
     {{.Prompt}} {{.HelpName}} myminio/mybucket -
 `,
}

// corsMessage container for output message.
type corsMessage struct {
	op      string
	Status  string       `json:"status"`
	Message string       `json:"message,omitempty"`
	CorsCfg *cors.Config `json:"cors,omitempty"`
}

func (c corsMessage) String() string {
	switch c.op {
	case "get":
		if c.CorsCfg == nil {
			return console.Colorize("CorsNotFound", "No bucket CORS configuration found.")
		}
		corsXML, e := c.CorsCfg.ToXML()
		fatalIf(probe.NewError(e), "Unable to marshal to XML.")
		return string(corsXML)
	case "set":
		return console.Colorize("CorsMessage", "Set bucket CORS config successfully.")
	case "remove":
		return console.Colorize("CorsMessage", "Removed bucket CORS config successfully.")
	}
	return ""
}

func (c corsMessage) JSON() string {
	jsonBytes, e := json.MarshalIndent(&c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal to JSON.")

	return string(jsonBytes)
}

// checkCorsSetSyntax - validate all the passed arguments
func checkCorsSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainCorsSet is the handle for "mc cors set" command.
func mainCorsSet(ctx *cli.Context) error {
	checkCorsSetSyntax(ctx)

	console.SetColor("CorsMessage", color.New(color.FgGreen))

	// args[0] is the ALIAS/BUCKET argument.
	args := ctx.Args()
	urlStr := args.Get(0)

	// args[1] is the CORSFILE which is a local file, or in the case of "-", stdin.
	var e error
	in := os.Stdin
	if f := args.Get(1); f != "-" {
		in, e = os.Open(f)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to open bucket CORS configuration file.")
		defer in.Close()
	}
	corsXML, e := io.ReadAll(in)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to read bucket CORS configuration file.")

	client, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize client for "+urlStr)

	fatalIf(client.SetBucketCors(globalContext, corsXML), "Unable to set bucket CORS configuration for "+urlStr)

	printMsg(corsMessage{
		op:     "set",
		Status: "success",
	})

	return nil
}
