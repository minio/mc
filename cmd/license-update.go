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
  {{.HelpName}} ALIAS LICENSE-FILE-PATH

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Update license for cluster with alias 'play' from the file license.key
     {{.Prompt}} {{.HelpName}} play license.key
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
	if len(ctx.Args()) != 2 {
		cli.ShowCommandHelpAndExit(ctx, "update", 1) // last argument is exit code
	}

	console.SetColor(licUpdateMsgTag, color.New(color.FgGreen, color.Bold))

	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)

	licFile := ctx.Args().Get(1)

	// If set, the subnet public key will not be downloaded from subnet
	// and the offline key embedded in mc will be used.
	airgap := ctx.Bool("airgap")

	printMsg(performLicenseUpdate(licFile, alias, airgap))
	return nil
}

func performLicenseUpdate(licFile string, alias string, airgap bool) licUpdateMessage {
	lum := licUpdateMessage{
		Alias:  alias,
		Status: "success",
	}

	licBytes, e := os.ReadFile(licFile)
	fatalIf(probe.NewError(e), fmt.Sprintf("Unable to read license file %s", licFile))

	lic := string(licBytes)
	li, e := parseLicense(lic, airgap)
	fatalIf(probe.NewError(e), fmt.Sprintf("Error parsing license from %s", licFile))

	if li.ExpiresAt.Before(time.Now()) {
		fatalIf(errDummy().Trace(), fmt.Sprintf("License has expired on %s", li.ExpiresAt))
	}

	setSubnetCreds(alias, "", lic)
	return lum
}
