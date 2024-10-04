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
	"os"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
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
		Name:  "account-key",
		Value: "",
		Usage: "Azure Blob Storage account key",
	},
	cli.StringFlag{
		Name:  "az-sp-tenant-id",
		Value: "",
		Usage: "Directory ID for the Azure service principal account",
	},
	cli.StringFlag{
		Name:  "az-sp-client-id",
		Value: "",
		Usage: "The client ID of the Azure service principal account",
	},
	cli.StringFlag{
		Name:  "az-sp-client-secret",
		Value: "",
		Usage: "The client secret of the Azure service principal account",
	},
	cli.StringFlag{
		Name:  "credentials-file",
		Value: "",
		Usage: "path to Google Cloud Storage credentials file",
	},
}

var adminTierEditCmd = cli.Command{
	Name:         "edit",
	Usage:        "update an existing remote tier configuration",
	Action:       mainAdminTierEdit,
	Hidden:       true,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminTierEditFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS NAME

NAME:
  Name of remote tier. e.g WARM-TIER

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Update credentials for an existing Azure Blob Storage remote tier:
     {{.Prompt}} {{.HelpName}} myminio AZTIER --account-key ACCOUNT-KEY

  2. Update credentials for an existing AWS S3 compatible remote tier:
     {{.Prompt}} {{.HelpName}} myminio S3TIER --access-key ACCESS-KEY --secret-key SECRET-KEY

  3. Update credentials for an existing Google Cloud Storage remote tier:
     {{.Prompt}} {{.HelpName}} myminio GCSTIER --credentials-file /path/to/credentials.json
`,
}

// checkAdminTierEditSyntax - validate all the postitional arguments
func checkAdminTierEditSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 2 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
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
	credsPath := ctx.String("credentials-file")
	useAwsRole := ctx.IsSet("use-aws-role")

	// Azure, either account-key or one of the 3 service principal flags are required
	accountKey := ctx.String("account-key")
	azSPTenantID := ctx.String("az-sp-tenant-id")
	azSPClientID := ctx.String("az-sp-client-id")
	azSPClientSecret := ctx.String("az-sp-client-secret")

	switch {
	case accessKey != "" && secretKey != "" && !useAwsRole: // S3 tier
		creds.AccessKey = accessKey
		creds.SecretKey = secretKey
	case useAwsRole:
		creds.AWSRole = true
	case accountKey != "": // Azure tier, account key given
		creds.SecretKey = accountKey
	case azSPTenantID != "" || azSPClientID != "" || azSPClientSecret != "": // Azure tier, SP creds given
		creds.AzSP = madmin.ServicePrincipalAuth{
			TenantID:     azSPTenantID,
			ClientID:     azSPClientID,
			ClientSecret: azSPClientSecret,
		}
	case credsPath != "": // GCS tier
		credsBytes, e := os.ReadFile(credsPath)
		fatalIf(probe.NewError(e), "Unable to read credentials file at %s", credsPath)

		creds.CredsJSON = credsBytes
	default:
		fatalIf(errInvalidArgument().Trace(args.Tail()...), "Insufficient credential information supplied to update remote tier target credentials")
	}

	e := client.EditTier(globalContext, tierName, creds)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to edit remote tier")

	printMsg(&tierMessage{
		op:       ctx.Command.Name,
		Status:   "success",
		TierName: tierName,
	})
	return nil
}
