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
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var adminTierAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "endpoint",
		Value: "",
		Usage: "remote tier endpoint. e.g https://s3.amazonaws.com",
	},
	cli.StringFlag{
		Name:  "region",
		Value: "",
		Usage: "remote tier region. e.g us-west-2",
	},
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
		Name:  "aws-role-arn",
		Usage: "use AWS S3 role name",
	},
	cli.StringFlag{
		Name:  "aws-web-identity-file",
		Usage: "use AWS S3 web identity file",
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
	cli.StringFlag{
		Name:  "bucket",
		Value: "",
		Usage: "remote tier bucket",
	},
	cli.StringFlag{
		Name:  "prefix",
		Value: "",
		Usage: "remote tier prefix",
	},
	cli.StringFlag{
		Name:  "storage-class",
		Value: "",
		Usage: "remote tier storage-class",
	},
	cli.BoolFlag{
		Name:   "force",
		Hidden: true,
		Usage:  "ignores in-use check for remote tier bucket/prefix",
	},
}

var adminTierAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a new remote tier target",
	Action:       mainAdminTierAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(globalFlags, adminTierAddFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TYPE ALIAS NAME [FLAGS]

TYPE:
  Transition objects to supported cloud storage backend tier. Supported values are minio, s3, azure and gcs.

NAME:
  Name of the remote tier target. e.g WARM-TIER

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Configure a new remote tier which transitions objects to a bucket in a MinIO deployment:
     {{.Prompt}} {{.HelpName}} minio myminio WARM-MINIO-TIER --endpoint https://warm-minio.com \
        --access-key ACCESSKEY --secret-key SECRETKEY --bucket mybucket --prefix myprefix/

  2. Configure a new remote tier which transitions objects to a bucket in Azure Blob Storage:
     {{.Prompt}} {{.HelpName}} azure myminio AZTIER --account-name ACCOUNT-NAME --account-key ACCOUNT-KEY \
        --bucket myazurebucket --prefix myazureprefix/

  3. Configure a new remote tier which transitions objects to a bucket in AWS S3 with STANDARD storage class:
     {{.Prompt}} {{.HelpName}} s3 myminio S3TIER --endpoint https://s3.amazonaws.com \
        --access-key ACCESSKEY --secret-key SECRETKEY --bucket mys3bucket --prefix mys3prefix/ \
        --storage-class "STANDARD" --region us-west-2

  4. Configure a new remote tier which transitions objects to a bucket in Google Cloud Storage:
     {{.Prompt}} {{.HelpName}} gcs myminio GCSTIER --credentials-file /path/to/credentials.json \
        --bucket mygcsbucket  --prefix mygcsprefix/
`,
}

// checkAdminTierAddSyntax validates all the positional arguments
func checkAdminTierAddSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 3 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
	if argsNr > 3 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier add command.")
	}
}

// The list of AWS S3 storage classes that can be used with MinIO ILM tiering
var supportedAWSTierSC = []string{"STANDARD", "REDUCED_REDUNDANCY", "STANDARD_IA"}

// fetchTierConfig returns a TierConfig given a tierName, a tierType and ctx to
// lookup command-line flags from. It exits with non-zero error code if any of
// the flags contain invalid values.
func fetchTierConfig(ctx *cli.Context, tierName string, tierType madmin.TierType) *madmin.TierConfig {
	switch tierType {
	case madmin.MinIO:
		accessKey := ctx.String("access-key")
		secretKey := ctx.String("secret-key")
		if accessKey == "" || secretKey == "" {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote tier requires access credentials", tierType))
		}
		bucket := ctx.String("bucket")
		if bucket == "" {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote tier requires target bucket", tierType))
		}

		endpoint := ctx.String("endpoint")
		if endpoint == "" {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote tier requires target endpoint", tierType))
		}

		minioOpts := []madmin.MinIOOptions{}
		prefix := ctx.String("prefix")
		if prefix != "" {
			minioOpts = append(minioOpts, madmin.MinIOPrefix(prefix))
		}

		region := ctx.String("region")
		if region != "" {
			minioOpts = append(minioOpts, madmin.MinIORegion(region))
		}

		minioCfg, e := madmin.NewTierMinIO(tierName, endpoint, accessKey, secretKey, bucket, minioOpts...)
		fatalIf(probe.NewError(e), "Invalid configuration for MinIO tier")

		return minioCfg

	case madmin.S3:
		accessKey := ctx.IsSet("access-key")
		secretKey := ctx.IsSet("secret-key")
		useAwsRole := ctx.IsSet("use-aws-role")
		awsRoleArn := ctx.IsSet("aws-role-arn")
		awsWebIdentity := ctx.IsSet("aws-web-identity-file")

		// Extensive flag check
		switch {
		case !accessKey && !secretKey && !useAwsRole && !awsRoleArn && !awsWebIdentity:
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s: No authentication mechanism was provided", tierType))
		case (accessKey || secretKey) && (useAwsRole || awsRoleArn || awsWebIdentity):
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s: Static credentials cannot be combined with AWS role authentication", tierType))
		case useAwsRole && (awsRoleArn || awsWebIdentity):
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s: --use-aws-role cannot be combined with --aws-role-arn or --aws-web-identity-file", tierType))
		case (awsRoleArn && !awsWebIdentity) || (!awsRoleArn && awsWebIdentity):
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s: Both --use-aws-role and --aws-web-identity-file are required to enable web identity token based authentication", tierType))
		case (accessKey && !secretKey) || (!accessKey && secretKey):
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s: Both --access-key and --secret-key are required to enable static credentials authentication", tierType))

		}

		bucket := ctx.String("bucket")
		if bucket == "" {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote tier requires target bucket", tierType))
		}

		s3Opts := []madmin.S3Options{}
		prefix := ctx.String("prefix")
		if prefix != "" {
			s3Opts = append(s3Opts, madmin.S3Prefix(prefix))
		}

		endpoint := ctx.String("endpoint")
		if endpoint != "" {
			s3Opts = append(s3Opts, madmin.S3Endpoint(endpoint))
		}

		region := ctx.String("region")
		if region != "" {
			s3Opts = append(s3Opts, madmin.S3Region(region))
		}

		s3SC := ctx.String("storage-class")
		if s3SC != "" {
			if !slices.Contains(supportedAWSTierSC, s3SC) {
				fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("unsupported storage-class type %s", s3SC))
			}
			s3Opts = append(s3Opts, madmin.S3StorageClass(s3SC))
		}
		if ctx.IsSet("use-aws-role") {
			s3Opts = append(s3Opts, madmin.S3AWSRole())
		}
		if ctx.IsSet("aws-role-arn") {
			s3Opts = append(s3Opts, madmin.S3AWSRoleARN(ctx.String("aws-role-arn")))
		}
		if ctx.IsSet("aws-web-identity-file") {
			s3Opts = append(s3Opts, madmin.S3AWSRoleWebIdentityTokenFile(ctx.String("aws-web-identity-file")))
		}
		s3Cfg, e := madmin.NewTierS3(tierName, ctx.String("access-key"), ctx.String("secret-key"), bucket, s3Opts...)
		fatalIf(probe.NewError(e), "Invalid configuration for AWS S3 compatible remote tier")

		return s3Cfg
	case madmin.Azure:
		accountName := ctx.String("account-name")
		accountKey := ctx.String("account-key")
		if accountName == "" {
			fatalIf(errDummy().Trace(), fmt.Sprintf("%s remote tier requires the storage account name", tierType))
		}

		if accountKey == "" && (ctx.String("az-sp-tenant-id") == "" || ctx.String("az-sp-client-id") == "" || ctx.String("az-sp-client-secret") == "") {
			fatalIf(errDummy().Trace(), fmt.Sprintf("%s remote tier requires static credentials OR service principal credentials", tierType))
		}

		bucket := ctx.String("bucket")
		if bucket == "" {
			fatalIf(errDummy().Trace(), fmt.Sprintf("%s remote tier requires target bucket", tierType))
		}

		azOpts := []madmin.AzureOptions{}
		endpoint := ctx.String("endpoint")
		if endpoint != "" {
			azOpts = append(azOpts, madmin.AzureEndpoint(endpoint))
		}

		region := ctx.String("region")
		if region != "" {
			azOpts = append(azOpts, madmin.AzureRegion(region))
		}

		prefix := ctx.String("prefix")
		if prefix != "" {
			azOpts = append(azOpts, madmin.AzurePrefix(prefix))
		}

		if ctx.String("az-sp-tenant-id") != "" || ctx.String("az-sp-client-id") != "" || ctx.String("az-sp-client-secret") != "" {
			azOpts = append(azOpts, madmin.AzureServicePrincipal(ctx.String("az-sp-tenant-id"), ctx.String("az-sp-client-id"), ctx.String("az-sp-client-secret")))
		}

		azCfg, e := madmin.NewTierAzure(tierName, accountName, accountKey, bucket, azOpts...)
		fatalIf(probe.NewError(e), "Invalid configuration for Azure Blob Storage remote tier")

		return azCfg
	case madmin.GCS:
		bucket := ctx.String("bucket")
		if bucket == "" {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote requires target bucket", tierType))
		}

		gcsOpts := []madmin.GCSOptions{}
		prefix := ctx.String("prefix")
		if prefix != "" {
			gcsOpts = append(gcsOpts, madmin.GCSPrefix(prefix))
		}

		region := ctx.String("region")
		if region != "" {
			gcsOpts = append(gcsOpts, madmin.GCSRegion(region))
		}

		credsPath := ctx.String("credentials-file")
		credsBytes, e := os.ReadFile(credsPath)
		fatalIf(probe.NewError(e), "Failed to read credentials file")

		gcsCfg, e := madmin.NewTierGCS(tierName, credsBytes, bucket, gcsOpts...)
		fatalIf(probe.NewError(e), "Invalid configuration for Google Cloud Storage remote tier")

		return gcsCfg
	}
	fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("Invalid remote tier type %s", tierType))
	return nil
}

type tierMessage struct {
	op         string
	Status     string            `json:"status"`
	TierName   string            `json:"tierName"`
	TierType   string            `json:"tierType"`
	Endpoint   string            `json:"tierEndpoint"`
	Bucket     string            `json:"bucket"`
	Prefix     string            `json:"prefix,omitempty"`
	Region     string            `json:"region,omitempty"`
	TierParams map[string]string `json:"tierParams,omitempty"`
}

// String returns string representation of msg
func (msg *tierMessage) String() string {
	switch msg.op {
	case "add":
		addMsg := fmt.Sprintf("Added remote tier %s of type %s", msg.TierName, msg.TierType)
		return console.Colorize("TierMessage", addMsg)
	case "rm":
		rmMsg := fmt.Sprintf("Removed remote tier %s", msg.TierName)
		return console.Colorize("TierMessage", rmMsg)
	case "verify":
		verifyMsg := fmt.Sprintf("Verified remote tier %s", msg.TierName)
		return console.Colorize("TierMessage", verifyMsg)
	case "check":
		checkMsg := fmt.Sprintf("Remote tier connectivity check for %s was successful", msg.TierName)
		return console.Colorize("TierMessage", checkMsg)
	case "edit":
		editMsg := fmt.Sprintf("Updated remote tier %s", msg.TierName)
		return console.Colorize("TierMessage", editMsg)
	}
	return ""
}

// JSON returns json encoded msg
func (msg *tierMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(msg, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

// SetTierConfig sets TierConfig related fields
func (msg *tierMessage) SetTierConfig(sCfg *madmin.TierConfig) {
	msg.TierName = sCfg.Name
	msg.TierType = sCfg.Type.String()
	msg.Endpoint = sCfg.Endpoint()
	msg.Bucket = sCfg.Bucket()
	msg.Prefix = sCfg.Prefix()
	msg.Region = sCfg.Region()
	switch sCfg.Type {
	case madmin.S3:
		msg.TierParams = map[string]string{
			"storageClass": sCfg.S3.StorageClass,
		}
	}
}

func mainAdminTierAdd(ctx *cli.Context) error {
	checkAdminTierAddSyntax(ctx)

	console.SetColor("TierMessage", color.New(color.FgGreen))

	args := ctx.Args()
	tierTypeStr := args.Get(0)
	tierType, e := madmin.NewTierType(tierTypeStr)
	fatalIf(probe.NewError(e), "Unsupported tier type")

	aliasedURL := args.Get(1)
	tierName := args.Get(2)
	if tierName == "" {
		fatalIf(errInvalidArgument(), "Tier name can't be empty")
	}

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	tCfg := fetchTierConfig(ctx, strings.ToUpper(tierName), tierType)
	ignoreInUse := ctx.Bool("force")
	if ignoreInUse {
		fatalIf(probe.NewError(client.AddTierIgnoreInUse(globalContext, tCfg)).Trace(args...), "Unable to configure remote tier target")
	} else {
		fatalIf(probe.NewError(client.AddTier(globalContext, tCfg)).Trace(args...), "Unable to configure remote tier target")
	}

	msg := &tierMessage{
		op:     ctx.Command.Name,
		Status: "success",
	}
	msg.SetTierConfig(tCfg)
	printMsg(msg)
	return nil
}
