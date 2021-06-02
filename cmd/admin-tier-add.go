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
	"fmt"
	"io/ioutil"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
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
  {{.HelpName}} TIER_TYPE TARGET NAME [TIER_FLAGS]

NAME:
  Name of remote tier target. e.g WARM-TIER

TIER_TYPE:
  Cloud storage backend where objects specified by bucket lifecycle configuration can be transitioned to.
  Supported values are s3, azure and gcs.

TIER_FLAGS:
  Tier type specific flags.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Configure a new remote tier which transitions objects to a bucket in Azure Blob Storage.
     {{.Prompt}} {{.HelpName}} azure myminio AZTIER --account-name foobar --account-key foobar123 --region us-east-1 --bucket testbucket --prefix testprefix/

  2. Configure a new remote tier which transitions objects to a bucket in AWS S3 with STANDARD storage class.
     {{.Prompt}} {{.HelpName}} s3 myminio S3TIER --endpoint https://s3.amazonaws.com --access-key foobar \
        --secret-key foobar123 --region us-east-1 --bucket testbucket --prefix testprefix/ --storage-class "STANDARD"

  3. Configure a new remote tier which transitions objects to a bucket in Google Cloud Storage.
     {{.Prompt}} {{.HelpName}} s3 myminio GCSTIER --credentials-file /path/to/credentials.json --region us-east-1 --bucket testbucket --prefix testprefix/

  4. Configure a new remote tier which transitions objects to a bucket in AWS S3 with STANDARD storage class using aws role.
	 {{.Prompt}} {{.HelpName}} s3 myminio S3TIER --endpoint https://s3.amazonaws.com --use-aws-role \
	 	--region us-east-1 --bucket testbucket --prefix testprefix/ --storage-class "STANDARD"
`,
}

// checkAdminTierAddSyntax validates all the positional arguments
func checkAdminTierAddSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 3 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
	if argsNr > 3 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier add command.")
	}
}

const (
	s3Standard          = "STANDARD"
	s3ReducedRedundancy = "REDUCED_REDUNDANCY"
)

// fetchTierConfig returns a TierConfig given a tierName, a tierType and ctx to
// lookup command-line flags from. It exits with non-zero error code if any of
// the flags contain invalid values.
func fetchTierConfig(ctx *cli.Context, tierName string, tierType madmin.TierType) *madmin.TierConfig {
	switch tierType {
	case madmin.S3:
		accessKey := ctx.String("access-key")
		secretKey := ctx.String("secret-key")
		useAwsRole := ctx.IsSet("use-aws-role")
		if accessKey == "" && secretKey == "" && !useAwsRole {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote tier requires access credentials or AWS role", tierType))
		}
		if (accessKey != "" || secretKey != "") && useAwsRole {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote tier requires access credentials", tierType))
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
			if s3SC != s3Standard && s3SC != s3ReducedRedundancy {
				fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("unsupported storage-class type %s", s3SC))
			}
			s3Opts = append(s3Opts, madmin.S3StorageClass(s3SC))
		}
		if ctx.IsSet("use-aws-role") {
			s3Opts = append(s3Opts, madmin.S3AWSRole())
		}
		s3Cfg, err := madmin.NewTierS3(tierName, accessKey, secretKey, bucket, s3Opts...)
		if err != nil {
			fatalIf(probe.NewError(err), "Invalid configuration for AWS S3 compatible remote tier")
		}

		return s3Cfg
	case madmin.Azure:
		accountName := ctx.String("account-name")
		accountKey := ctx.String("account-key")
		if accountName == "" || accountKey == "" {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote tier requires access credentials", tierType))
		}

		bucket := ctx.String("bucket")
		if bucket == "" {
			fatalIf(errInvalidArgument().Trace(), fmt.Sprintf("%s remote tier requires target bucket", tierType))
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

		azCfg, err := madmin.NewTierAzure(tierName, accountName, accountKey, bucket, azOpts...)
		if err != nil {
			fatalIf(probe.NewError(err), "Invalid configuration for Azure Blob Storage remote tier")
		}

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
		credsBytes, err := ioutil.ReadFile(credsPath)
		if err != nil {
			fatalIf(probe.NewError(err), "Failed to read credentials file")
		}

		gcsCfg, err := madmin.NewTierGCS(tierName, credsBytes, bucket, gcsOpts...)
		if err != nil {
			fatalIf(probe.NewError(err), "Invalid configuration for Google Cloud Storage remote tier")
		}

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
	case "ls":
		// nothing to do here; ls has its own type to use with printMsg
	case "rm":
		rmMsg := fmt.Sprintf("Removed remote tier %s", msg.TierName)
		return console.Colorize("TierMessage", rmMsg)
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
	tierType, err := madmin.NewTierType(tierTypeStr)
	fatalIf(probe.NewError(err), "Unsupported tier type")

	aliasedURL := args.Get(1)
	tierName := args.Get(2)
	if tierName == "" {
		fatalIf(errInvalidArgument(), "Tier name can't be empty")
	}

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	tCfg := fetchTierConfig(ctx, tierName, tierType)
	if err = client.AddTier(globalContext, tCfg); err != nil {
		fatalIf(probe.NewError(err).Trace(args...), "Unable to configure remote tier target")
	}

	msg := &tierMessage{
		op:     "add",
		Status: "success",
	}
	msg.SetTierConfig(tCfg)
	printMsg(msg)
	return nil
}
