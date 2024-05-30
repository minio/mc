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
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var licenseUpdateCmd = cli.Command{
	Name:         "update",
	Usage:        "update the license",
	OnUsageError: onUsageError,
	Action:       mainLicenseUpdate,
	Before:       setGlobalsFromContext,
	Flags:        supportGlobalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS LICENSE-FILE-PATH

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Update license for cluster with alias 'play' from the file license.key
     {{.Prompt}} {{.HelpName}} play license.key
  2. Update (renew) license for already registered cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play
`,
}

const licUpdateMsgTag = "licenseUpdateMessage"

type licUpdateMessage struct {
	Status string `json:"status"`
	Alias  string `json:"-"`
}

// String colorized license update message
func (li licUpdateMessage) String() string {
	return console.Colorize(licUpdateMsgTag, "License updated successfully for "+li.Alias)
}

// JSON jsonified license update message
func (li licUpdateMessage) JSON() string {
	jsonBytes, e := json.MarshalIndent(li, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

func mainLicenseUpdate(ctx *cli.Context) error {
	args := ctx.Args()
	argsLen := len(args)
	if argsLen > 2 || argsLen < 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	console.SetColor(licUpdateMsgTag, color.New(color.FgGreen, color.Bold))

	aliasedURL := args.Get(0)
	alias, _ := url2Alias(aliasedURL)

	if argsLen == 2 {
		licFile := args.Get(1)
		printMsg(performLicenseUpdate(licFile, alias))
		return nil
	}

	// renew the license
	printMsg(performLicenseRenew(alias))
	return nil
}

func performLicenseRenew(alias string) licUpdateMessage {
	apiKey, _, e := getSubnetCreds(alias)
	fatalIf(probe.NewError(e), "Error getting subnet creds")

	if len(apiKey) == 0 {
		errMsg := fmt.Sprintf("Please register the cluster first by running 'mc license register %s'", alias)
		fatal(errDummy().Trace(), errMsg)
	}

	renewURL := subnetLicenseRenewURL()
	headers := SubnetAPIKeyAuthHeaders(apiKey)
	headers.addDeploymentIDHeader(alias)
	resp, e := SubnetPostReq(renewURL, nil, headers)
	fatalIf(probe.NewError(e), "Error renewing license for %s", alias)

	extractAndSaveSubnetCreds(alias, resp)

	return licUpdateMessage{
		Alias:  alias,
		Status: "success",
	}
}

func performLicenseUpdate(licFile, alias string) licUpdateMessage {
	lum := licUpdateMessage{
		Alias:  alias,
		Status: "success",
	}

	licBytes, e := os.ReadFile(licFile)
	fatalIf(probe.NewError(e), fmt.Sprintf("Unable to read license file %s", licFile))

	lic := string(licBytes)
	validateAndSaveLic(lic, alias, true)

	return lum
}
