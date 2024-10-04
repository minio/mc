// Copyright (c) 2015-2022 MinIO, Inc.
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
	"github.com/minio/pkg/v3/console"
)

var versionEnableFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "excluded-prefixes",
		Usage: "exclude versioning on these prefix patterns",
	},
	cli.BoolFlag{
		Name:  "exclude-folders",
		Usage: "exclude versioning on folder objects",
	},
}

var versionEnableCmd = cli.Command{
	Name:         "enable",
	Usage:        "enable bucket versioning",
	Action:       mainVersionEnable,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, versionEnableFlags...),
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

  2. Enable versioning on bucket "mybucket" while excluding versioning on a few select prefixes.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --excluded-prefixes "app1/*/_temporary/,app2/*/_staging/"

  3. Enable versioning on bucket "mybucket" while excluding versioning on a few select prefixes and all folders.
     Note: this is useful on buckets used with Spark/Hadoop workloads.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --excluded-prefixes "app1/*/_temporary/,app2/*/_staging/" --exclude-folders
`,
}

// checkVersionEnableSyntax - validate all the passed arguments
func checkVersionEnableSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

type versionEnableMessage struct {
	Op         string
	Status     string `json:"status"`
	URL        string `json:"url"`
	Versioning struct {
		Status           string   `json:"status"`
		MFADelete        string   `json:"MFADelete"`
		ExcludedPrefixes []string `json:"ExcludedPrefixes,omitempty"`
		ExcludeFolders   bool     `json:"ExcludeFolders,,omitempty"`
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

	var excludedPrefixes []string
	prefixesStr := cliCtx.String("excluded-prefixes")
	if prefixesStr != "" {
		excludedPrefixes = strings.Split(prefixesStr, ",")
	}
	excludeFolders := cliCtx.Bool("exclude-folders")

	// Create a new Client
	client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize connection.")
	fatalIf(client.SetVersion(ctx, "enable", excludedPrefixes, excludeFolders), "Unable to enable versioning")
	printMsg(versionEnableMessage{
		Op:     cliCtx.Command.Name,
		Status: "success",
		URL:    aliasedURL,
	})
	return nil
}
