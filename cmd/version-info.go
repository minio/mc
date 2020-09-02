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
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
)

var versionInfoCmd = cli.Command{
	Name:   "info",
	Usage:  "show bucket versioning status",
	Action: mainVersionInfo,
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
