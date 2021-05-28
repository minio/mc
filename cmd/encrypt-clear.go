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

var encryptClearCmd = cli.Command{
	Name:         "clear",
	Usage:        "clear encryption config",
	Action:       mainEncryptClear,
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
  1. Remove auto encryption config on bucket "mybucket" for alias "myminio".
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkEncryptClearSyntax - validate all the passed arguments
func checkEncryptClearSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "clear", 1) // last argument is exit code
	}
}

type encryptClearMessage struct {
	Op     string `json:"op"`
	Status string `json:"status"`
	URL    string `json:"url"`
}

func (v encryptClearMessage) JSON() string {
	v.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(v, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (v encryptClearMessage) String() string {
	return console.Colorize("encryptClearMessage", fmt.Sprintf("Auto encryption configuration has been cleared successfully for %s", v.URL))
}

func mainEncryptClear(cliCtx *cli.Context) error {
	ctx, cancelencryptClear := context.WithCancel(globalContext)
	defer cancelencryptClear()

	console.SetColor("encryptClearMessage", color.New(color.FgGreen))

	checkEncryptClearSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	fatalIf(client.DeleteEncryption(ctx), "Unable to clear auto encryption configuration")
	printMsg(encryptClearMessage{
		Op:     "clear",
		Status: "success",
		URL:    aliasedURL,
	})
	return nil
}
