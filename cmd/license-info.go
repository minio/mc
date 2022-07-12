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
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"github.com/olekukonko/tablewriter"
)

var licenseInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display license information",
	OnUsageError: onUsageError,
	Action:       mainLicenseInfo,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, subnetCommonFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Display license configuration for cluster with alias 'play'
     {{.Prompt}} {{.HelpName}} play
`,
}

const (
	licInfoMsgTag   = "licenseInfoMessage"
	licInfoErrTag   = "licenseInfoError"
	licInfoFieldTag = "licenseInfoField"
	licInfoValTag   = "licenseValueField"
)

type licInfoMessage struct {
	Status string  `json:"status"`
	Info   licInfo `json:"info,omitempty"`
	Error  string  `json:"error,omitempty"`
}

type licInfo struct {
	Organization string     `json:"org,omitempty"`           // Subnet organization name
	Plan         string     `json:"plan,omitempty"`          // Subnet plan
	IssuedAt     *time.Time `json:"issued_at,omitempty"`     // Time of license issue
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`    // Time of license expiry
	DeploymentID string     `json:"deployment_id,omitempty"` // Cluster deployment ID
	Message      string     `json:"message,omitempty"`       // Message to be displayed
}

func licInfoField(s string) string {
	return console.Colorize(licInfoFieldTag, s)
}

func licInfoVal(s string) string {
	return console.Colorize(licInfoValTag, s)
}

func licInfoMsg(s string) string {
	return console.Colorize(licInfoMsgTag, s)
}

func licInfoErr(s string) string {
	return console.Colorize(licInfoErrTag, s)
}

// String colorized license info
func (li licInfoMessage) String() string {
	if len(li.Error) > 0 {
		return licInfoErr(li.Error)
	}

	if len(li.Info.Message) > 0 {
		return licInfoMsg(li.Info.Message)
	}

	return licInfoMsg(getLicInfoStr(li.Info))
}

// JSON jsonified license info
func (li licInfoMessage) JSON() string {
	jsonBytes, e := json.MarshalIndent(li, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonBytes)
}

func getLicInfoStr(li licInfo) string {
	var s strings.Builder

	s.WriteString(color.WhiteString(""))
	table := tablewriter.NewWriter(&s)
	table.SetAutoWrapText(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(true)
	table.SetRowLine(false)

	data := [][]string{
		{licInfoField("Organization"), licInfoVal(li.Organization)},
		{licInfoField("Deployment ID"), licInfoVal(li.DeploymentID)},
		{licInfoField("Plan"), licInfoVal(li.Plan)},
		{licInfoField("Issued at"), licInfoVal(li.IssuedAt.String())},
		{licInfoField("Expires at"), licInfoVal(li.ExpiresAt.String())},
	}
	table.AppendBulk(data)
	table.Render()

	return s.String()
}

func getAGPLMessage() string {
	return `You are using GNU AFFERO GENERAL PUBLIC LICENSE Verson 3 (https://www.gnu.org/licenses/agpl-3.0.txt)

If you are building proprietary applications, you may want to choose the commercial license
included as part of the Standard and Enterprise subscription plans. (https://min.io/signup?ref=mc)

Applications must otherwise comply with all the GNU AGPLv3 License & Trademark obligations.`
}

func initLicInfoColors() {
	console.SetColor(licInfoMsgTag, color.New(color.FgGreen, color.Bold))
	console.SetColor(licInfoErrTag, color.New(color.FgRed, color.Bold))
	console.SetColor(licInfoFieldTag, color.New(color.FgCyan))
	console.SetColor(licInfoValTag, color.New(color.FgWhite))
}

func mainLicenseInfo(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "info", 1) // last argument is exit code
	}

	initLicInfoColors()

	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)

	apiKey, lic, e := getSubnetCreds(alias)
	fatalIf(probe.NewError(e), "Error in checking cluster registration status")

	if len(apiKey) == 0 && len(lic) == 0 {
		// Not registered. Default to AGPLv3
		printMsg(licInfoMessage{
			Status: "success",
			Info: licInfo{
				Plan:    "AGPLv3",
				Message: getAGPLMessage(),
			},
		})
		return nil
	}

	var ssm licInfoMessage
	if len(lic) > 0 {
		// If set, the subnet public key will not be downloaded from subnet
		// and the offline key embedded in mc will be used.
		airgap := ctx.Bool("airgap")

		li, e := parseLicense(lic, airgap)
		if e != nil {
			ssm = licInfoMessage{
				Status: "error",
				Error:  e.Error(),
			}
		} else {
			ssm = licInfoMessage{
				Status: "success",
				Info: licInfo{
					Organization: li.Organization,
					Plan:         li.Plan,
					IssuedAt:     &li.IssuedAt,
					ExpiresAt:    &li.ExpiresAt,
					DeploymentID: li.DeploymentID,
				},
			}
		}
	} else {
		// Only api key is available, no license info
		ssm = licInfoMessage{
			Status: "success",
			Info: licInfo{
				Message: fmt.Sprintf("%s is registered with SUBNET. License info not available.", alias),
			},
		}
	}

	printMsg(ssm)
	return nil
}
