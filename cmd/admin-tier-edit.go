// Copyright (c) 2015-2021 MinIO, Inc.
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
	"io/ioutil"

	"github.com/fatih/color"
	"github.com/minio/cli"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminTierEditFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "access-key",
		Value: "",
		Usage: "AWS S3 or compatible object storage access-key",
	},
	cli.StringFlag{
		Name:  "secret-key",
		Value: "",
		Usage: "AWS S3 or compatible object storage secret-key",
	},
	cli.BoolFlag{
		Name:  "use-aws-role",
		Usage: "use AWS S3 role",
	},
	cli.StringFlag{
		Name:  "account-name",
		Value: "",
		Usage: "Azure Blob Storage account name",
	},
	cli.StringFlag{
		Name:  "account-key",
		Value: "",
		Usage: "Azure Blob Storage account key",
	},
	cli.StringFlag{
		Name:  "credentials-file",
		Value: "",
		Usage: "path to Google Cloud Storage credentials file",
	},
}

var adminTierEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "edits a remote tier",
	Action:       mainAdminTierEdit,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        adminTierEditFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET NAME

NAME:
  Name of remote tier. e.g WARM-TIER

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Update credentials for an existing Azure Blob Storage remote tier.
     {{.Prompt}} {{.HelpName}} myminio AZTIER --account-name foobar-new --account-key foobar-new123

  2. Update credentials for an existing AWS S3 compatible remote tier.
     {{.Prompt}} {{.HelpName}} myminio S3TIER --access-key foobar-new --secret-key foobar-new123

  3. Update credentials for an existing Google Cloud Storage remote tier.
     {{.Prompt}} {{.HelpName}} myminio GCSTIER --credentials-file /path/to/credentials.json
`,
}

// checkAdminTierEditSyntax - validate all the postitional arguments
func checkAdminTierEditSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 2 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
	if argsNr > 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier-edit subcommand.")
	}
}

func mainAdminTierEdit(ctx *cli.Context) error {
	checkAdminTierEditSyntax(ctx)

	console.SetColor("TierMessage", color.New(color.FgGreen))

	args := ctx.Args()
	aliasedURL := args.Get(0)
	tierName := args.Get(1)

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	var creds madmin.TierCreds
	accessKey := ctx.String("access-key")
	secretKey := ctx.String("secret-key")
	accountName := ctx.String("account-name")
	accountKey := ctx.String("account-key")
	credsPath := ctx.String("credentials-file")
	useAwsRole := ctx.IsSet("use-aws-role")

	switch {
	case accessKey != "" && secretKey != "" && !useAwsRole: // S3 tier
		creds.AccessKey = accessKey
		creds.SecretKey = secretKey
	case useAwsRole:
		creds.AWSRole = true
	case accountName != "" && accountKey != "": // Azure tier
		creds.AccessKey = accountName
		creds.SecretKey = accountKey
	case credsPath != "": // GCS tier
		credsBytes, err := ioutil.ReadFile(credsPath)
		if err != nil {
			fatalIf(probe.NewError(err), "Failed to read credentials file")
		}
		creds.CredsJSON = credsBytes
	default:
		fatalIf(errInvalidArgument().Trace(args.Tail()...), "Insufficient credential information supplied to update remote tier target credentials")
	}

	if err := client.EditTier(globalContext, tierName, creds); err != nil {
		fatalIf(probe.NewError(err).Trace(args...), "Unable to edit remote tier")
	}

	printMsg(&tierMessage{
		op:       "edit",
		Status:   "success",
		TierName: tierName,
	})
	return nil
}
