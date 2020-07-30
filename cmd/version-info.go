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
	Usage:  "Show bucket versioning status",
	Action: mainversionInfo,
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
   1. Display bucket versioning status for bucket "mybucket".
      {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkversionInfoSyntax - validate all the passed arguments
func checkversionInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "info", 1) // last argument is exit code
	}
}

type versionInfoMessage struct {
	Op         string
	Status     string `json:"status"`
	URL        string `json:"url"`
	Versioning struct {
		Status    string `json:"status"`
		MFADelete string `json:"MFADelete"`
	} `json:"versioning"`
}

func (v versionInfoMessage) JSON() string {
	v.Status = "success"
	jsonMessageBytes, e := json.MarshalIndent(v, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(jsonMessageBytes)
}

func (v versionInfoMessage) String() string {
	msg := ""
	switch v.Versioning.Status {
	case "":
		msg = fmt.Sprintf("%s is un-versioned", v.URL)
	default:
		msg = fmt.Sprintf("%s versioning status is %s", v.URL, strings.ToLower(v.Versioning.Status))
	}
	return console.Colorize("versionInfoMessage", msg)
}

func mainversionInfo(cliCtx *cli.Context) error {
	ctx, cancelVersionInfo := context.WithCancel(globalContext)
	defer cancelVersionInfo()

	console.SetColor("versionInfoMessage", color.New(color.FgGreen))

	checkversionInfoSyntax(cliCtx)

	// Get the alias parameter from cli
	args := cliCtx.Args()
	aliasedURL := args.Get(0)
	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	vConfig, e := client.GetVersioning(ctx)
	fatalIf(e, "Cannot get version info")
	vMsg := versionInfoMessage{
		Op:     "info",
		Status: "success",
		URL:    aliasedURL,
	}
	vMsg.Versioning.Status = vConfig.Status
	vMsg.Versioning.MFADelete = vConfig.MFADelete
	printMsg(vMsg)
	return nil
}
