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

var versionEnableCmd = cli.Command{
	Name:   "enable",
	Usage:  "enable bucket versioning",
	Action: mainVersionEnable,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
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
