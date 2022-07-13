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
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var licenseUpdateCmd = cli.Command{
	Name:         "update",
	Usage:        "update the license",
	OnUsageError: onUsageError,
	Action:       mainLicenseUpdate,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, subnetCommonFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS LICENSE-TOKEN

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Update license for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play eyJ0eXAiOiJKV1QiLCJhbGciOiJFUzM4NCJ9.eyJsaWQiOiJjMGRlNDk4OC0wYzA3LTRiYTMtYjY5NS00MDQ2NjllZDQwMWYiLCJzdWIiOiJzaGlyZWVzaCtjMSthZG1pbkBtaW5pby5pbyIsImNhcCI6MjAwMDAsIm9yZyI6IlNoaXJlZXNoLUMxIiwiaXNzIjoic3VibmV0QG1pbi5pbyIsImFpZCI6MSwiaWF0IjoxLjY1NzA4NDEwNDAzNjQxMzQ3MmU5LCJwbGFuIjoiRU5URVJQUklTRSIsImV4cCI6MS42ODg2MjAxMDQwMzY0MTM0NzJlOSwiZGlkIjoiY2Q2NTIwZTctYjg2Ny00YWU2LTkyY2EtNDc5MWI2OWEwY2M3In0.Ya9_HSpog8EhPY1Ckcay5J70_Rms1dNnu4xNlKvwy-8fF6lyF2bqsQMvuDOKaCIYj5w4May8l-1SJ5tC2mQ9Z_ycgCVWpwHGx6h2b7EOAtjGiN6yFEWBLedEScUx34u8
`,
}

const (
	licUpdateMsgTag = "licenseUpdateMessage"
	licUpdateErrTag = "licenseUpdateError"
)

type licUpdateMessage struct {
	Alias  string `json:"-"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func licUpdateMsg(s string) string {
	return console.Colorize(licUpdateMsgTag, s)
}

func licUpdateErr(s string) string {
	return console.Colorize(licUpdateErrTag, s)
}

// String colorized license update message
func (li licUpdateMessage) String() string {
	if len(li.Error) > 0 {
		return licUpdateErr(li.Error)
	}

	msg := fmt.Sprint("License updated successfully for", li.Alias)
	return licUpdateMsg(msg)
}

// JSON jsonified license update message
func (li licUpdateMessage) JSON() string {
	jsonBytes, e := json.MarshalIndent(li, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

func initLicUpdateColors() {
	console.SetColor(licUpdateMsgTag, color.New(color.FgGreen, color.Bold))
	console.SetColor(licUpdateErrTag, color.New(color.FgRed, color.Bold))
}

func mainLicenseUpdate(ctx *cli.Context) error {
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "update", 1) // last argument is exit code
	}

	initLicUpdateColors()

	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)

	lic := ctx.Args().Get(1)

	// If set, the subnet public key will not be downloaded from subnet
	// and the offline key embedded in mc will be used.
	airgap := ctx.Bool("airgap")

	printMsg(performLicenseUpdate(lic, alias, airgap))
	return nil
}

func performLicenseUpdate(lic string, alias string, airgap bool) licUpdateMessage {
	lum := licUpdateMessage{
		Alias:  alias,
		Status: "success",
	}

	li, e := parseLicense(lic, airgap)
	if e != nil {
		lum.Status = "error"
		lum.Error = e.Error()
		return lum
	}

	if li.ExpiresAt.Before(time.Now()) {
		lum.Status = "error"
		lum.Error = "License has expired"
		return lum
	}

	setSubnetCreds(alias, "", lic)
	return lum
}
