/*
 * MinIO Client (C) 2020 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var encryptClearCmd = cli.Command{
	Name:   "clear",
	Usage:  "clear encryption config",
	Action: mainEncryptClear,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
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
