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
	"net/url"
	"os"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v2/console"
)

// profile command flags.
var (
	uploadFlags = append(globalFlags,
		cli.IntFlag{
			Name:  "issue",
			Usage: "SUBNET issue number to which the file is to be uploaded",
		},
		cli.StringFlag{
			Name:  "comment",
			Usage: "comment to be posted on the issue along with the file",
		},
		cli.BoolFlag{
			Name:   "dev",
			Usage:  "Development mode",
			Hidden: true,
		},
	)
)

type supportUploadMessage struct {
	Status   string `json:"status"`
	IssueNum int    `json:"-"`
	IssueURL string `json:"issueUrl"`
}

// String colorized upload message
func (s supportUploadMessage) String() string {
	msg := fmt.Sprintf("File uploaded to SUBNET successfully. Click here to visit the issue: %s", subnetIssueURL(s.IssueNum))
	return console.Colorize(supportSuccessMsgTag, msg)
}

// JSON jsonified upload message
func (s supportUploadMessage) JSON() string {
	return toJSON(s)
}

var supportUploadCmd = cli.Command{
	Name:            "upload",
	Usage:           "upload file to SUBNET issue",
	Action:          mainSupportUpload,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           uploadFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] ALIAS FILE

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Upload file './trace.log' for cluster 'myminio' to SUBNET issue number 10
     {{.Prompt}} {{.HelpName}} --issue 10 myminio ./trace.log

  2. Upload file './trace.log' for cluster 'myminio' to SUBNET issue number 10 with comment 'here is the trace log'
     {{.Prompt}} {{.HelpName}} --issue 10 --comment "here is the trace log" myminio ./trace.log 
`,
}

func checkSupportUploadSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	if ctx.Int("issue") <= 0 {
		fatal(errDummy().Trace(), "Invalid issue number")
	}
}

// mainSupportUpload is the handle for "mc support upload" command.
func mainSupportUpload(ctx *cli.Context) error {
	// Check for command syntax
	checkSupportUploadSyntax(ctx)
	setSuccessMessageColor()

	// Get the alias parameter from cli
	aliasedURL := ctx.Args().Get(0)
	alias, apiKey := initSubnetConnectivity(ctx, aliasedURL, true)
	if len(apiKey) == 0 {
		// api key not passed as flag. Check that the cluster is registered.
		apiKey = validateClusterRegistered(alias, true)
	}

	// Main execution
	execSupportUpload(ctx, alias, apiKey)
	return nil
}

func execSupportUpload(ctx *cli.Context, alias, apiKey string) {
	filePath := ctx.Args().Get(1)
	issueNum := ctx.Int("issue")
	msg := ctx.String("comment")

	f, e := os.Open(filePath)
	fatalIf(probe.NewError(e), "Unable to open file '%s'", filePath)
	defer f.Close()
	fi, e := f.Stat()
	fatalIf(probe.NewError(e), "Unable to get file info for '%s'", filePath)
	filename := fi.Name()

	params := url.Values{}
	params.Add("issueNumber", fmt.Sprintf("%d", issueNum))
	if len(msg) > 0 {
		params.Add("message", msg)
	}

	uploadURL := subnetUploadURL("attachment", filename, params)
	reqURL, headers := prepareSubnetUploadURL(uploadURL, alias, apiKey)
	_, e = uploadFileToSubnet(alias, filePath, reqURL, headers)
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to upload file to SUBNET")
	}
	printMsg(supportUploadMessage{IssueNum: issueNum, Status: "success", IssueURL: subnetIssueURL(issueNum)})
}
