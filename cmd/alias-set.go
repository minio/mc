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
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/pkg/console"
	"golang.org/x/crypto/ssh/terminal"
)

const cred = "YellowItalics"

var aliasSetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "path",
		Value: "auto",
		Usage: "bucket path lookup supported by the server. Valid options are '[auto, on, off]'",
	},
	cli.StringFlag{
		Name:  "api",
		Usage: "API signature. Valid options are '[S3v4, S3v2]'",
	},
}

var aliasSetCmd = cli.Command{
	Name:      "set",
	ShortName: "s",
	Usage:     "set a new alias to configuration file",
	Action: func(cli *cli.Context) error {
		return mainAliasSet(cli, false)
	},
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(aliasSetFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS URL ACCESSKEY SECRETKEY

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add MinIO service under "myminio" alias. For security reasons turn off bash history momentarily.
     {{.DisableHistory}}
     {{.Prompt}} {{.HelpName}} myminio http://localhost:9000 minio minio123
     {{.EnableHistory}}

  2. Add MinIO service under "myminio" alias, to use dns style bucket lookup. For security reasons
     turn off bash history momentarily.
     {{.DisableHistory}}
     {{.Prompt}} {{.HelpName}} myminio http://localhost:9000 minio minio123 --api "s3v4" --path "off"
     {{.EnableHistory}}

  3. Add Amazon S3 storage service under "mys3" alias. For security reasons turn off bash history momentarily.
     {{.DisableHistory}}
     {{.Prompt}} {{.HelpName}} mys3 https://s3.amazonaws.com \
                 BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
     {{.EnableHistory}}

  4. Add Amazon S3 storage service under "mys3" alias, prompting for keys.
     {{.Prompt}} {{.HelpName}} mys3 https://s3.amazonaws.com --api "s3v4" --path "off"
     Enter Access Key: BKIKJAA5BMMU2RHO6IBB
     Enter Secret Key: V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12

  5. Add Amazon S3 storage service under "mys3" alias using piped keys.
     {{.DisableHistory}}
     {{.Prompt}} echo -e "BKIKJAA5BMMU2RHO6IBB\nV8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12" | \
                 {{.HelpName}} mys3 https://s3.amazonaws.com --api "s3v4" --path "off"
     {{.EnableHistory}}
`,
}

// checkAliasSetSyntax - verifies input arguments to 'alias set'.
func checkAliasSetSyntax(ctx *cli.Context, accessKey string, secretKey string, deprecated bool) {
	args := ctx.Args()
	argsNr := len(args)
	if argsNr > 4 || argsNr < 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for alias set command.")
	}

	alias := cleanAlias(args.Get(0))
	url := args.Get(1)
	api := ctx.String("api")
	path := ctx.String("path")
	bucketLookup := ctx.String("lookup")

	if !isValidAlias(alias) {
		fatalIf(errInvalidAlias(alias), "Invalid alias.")
	}

	if !isValidHostURL(url) {
		fatalIf(errInvalidURL(url), "Invalid URL.")
	}

	if !isValidAccessKey(accessKey) {
		fatalIf(errInvalidArgument().Trace(accessKey),
			"Invalid access key `"+accessKey+"`.")
	}

	if !isValidSecretKey(secretKey) {
		fatalIf(errInvalidArgument().Trace(secretKey),
			"Invalid secret key `"+secretKey+"`.")
	}

	if api != "" && !isValidAPI(api) { // Empty value set to default "S3v4".
		fatalIf(errInvalidArgument().Trace(api),
			"Unrecognized API signature. Valid options are `[S3v4, S3v2]`.")
	}

	if deprecated {
		if !isValidLookup(bucketLookup) {
			fatalIf(errInvalidArgument().Trace(bucketLookup),
				"Unrecognized bucket lookup. Valid options are `[dns,auto, path]`.")
		}
	} else {
		if !isValidPath(path) {
			fatalIf(errInvalidArgument().Trace(bucketLookup),
				"Unrecognized path value. Valid options are `[auto, on, off]`.")
		}
	}
}

// setAlias - set an alias config.
func setAlias(alias string, aliasCfgV10 aliasConfigV10) aliasMessage {
	mcCfgV10, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config `"+mustGetMcConfigPath()+"`.")

	// Add new host.
	mcCfgV10.Aliases[alias] = aliasCfgV10

	err = saveMcConfig(mcCfgV10)
	fatalIf(err.Trace(alias), "Unable to update hosts in config version `"+mustGetMcConfigPath()+"`.")

	return aliasMessage{
		Alias:     alias,
		URL:       aliasCfgV10.URL,
		AccessKey: aliasCfgV10.AccessKey,
		SecretKey: aliasCfgV10.SecretKey,
		API:       aliasCfgV10.API,
		Path:      aliasCfgV10.Path,
	}
}

// probeS3Signature - auto probe S3 server signature: issue a Stat call
// using v4 signature then v2 in case of failure.
func probeS3Signature(ctx context.Context, accessKey, secretKey, url string) (string, *probe.Error) {
	probeBucketName := randString(60, rand.NewSource(time.Now().UnixNano()), "probe-bucket-sign-")
	// Test s3 connection for API auto probe
	s3Config := &Config{
		// S3 connection parameters
		Insecure:  globalInsecure,
		AccessKey: accessKey,
		SecretKey: secretKey,
		HostURL:   urlJoinPath(url, probeBucketName),
		Debug:     globalDebug,
	}

	probeSignatureType := func(stype string) (string, *probe.Error) {
		s3Config.Signature = stype
		s3Client, err := S3New(s3Config)
		if err != nil {
			return "", err
		}

		if _, err := s3Client.Stat(ctx, StatOptions{}); err != nil {
			e := err.ToGoError()
			if _, ok := e.(BucketDoesNotExist); ok {
				// Bucket doesn't exist, means signature probing worked successfully.
				return stype, nil
			}
			// AccessDenied means Stat() is not allowed but credentials are valid.
			// AccessDenied is only returned when policy doesn't allow HeadBucket
			// operations.
			if minio.ToErrorResponse(err.ToGoError()).Code == "AccessDenied" {
				return stype, nil
			}

			// For any other errors we fail.
			return "", err.Trace(s3Config.Signature)
		}
		return stype, nil
	}

	stype, err := probeSignatureType("s3v4")
	if err != nil {
		if stype, err = probeSignatureType("s3v2"); err != nil {
			return "", err.Trace("s3v4", "s3v2")
		}
		return stype, nil
	}
	return stype, nil
}

// BuildS3Config constructs an S3 Config and does
// signature auto-probe when needed.
func BuildS3Config(ctx context.Context, url, accessKey, secretKey, api, path string) (*Config, *probe.Error) {
	s3Config := NewS3Config(url, &aliasConfigV10{
		AccessKey: accessKey,
		SecretKey: secretKey,
		URL:       url,
		Path:      path,
	})

	// If api is provided we do not auto probe signature, this is
	// required in situations when signature type is provided by the user.
	if api != "" {
		s3Config.Signature = api
		return s3Config, nil
	}
	// Probe S3 signature version
	api, err := probeS3Signature(ctx, accessKey, secretKey, url)
	if err != nil {
		return nil, err.Trace(url, accessKey, secretKey, api, path)
	}

	s3Config.Signature = api
	// Success.
	return s3Config, nil
}

// fetchAliasKeys - returns the user accessKey and secretKey
func fetchAliasKeys(args cli.Args) (string, string) {
	accessKey := ""
	secretKey := ""
	console.SetColor(cred, color.New(color.FgYellow, color.Italic))
	isTerminal := terminal.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	argsNr := len(args)

	if argsNr == 2 {
		if isTerminal {
			fmt.Printf("%s", console.Colorize(cred, "Enter Access Key: "))
		}
		value, _, _ := reader.ReadLine()
		accessKey = string(value)
	} else {
		accessKey = args.Get(2)
	}

	if argsNr == 2 || argsNr == 3 {
		if isTerminal {
			fmt.Printf("%s", console.Colorize(cred, "Enter Secret Key: "))
			bytePassword, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
			fmt.Printf("\n")
			secretKey = string(bytePassword)
		} else {
			value, _, _ := reader.ReadLine()
			secretKey = string(value)
		}
	} else {
		secretKey = args.Get(3)
	}

	return accessKey, secretKey
}

func mainAliasSet(cli *cli.Context, deprecated bool) error {
	console.SetColor("AliasMessage", color.New(color.FgGreen))
	var (
		args  = cli.Args()
		alias = cleanAlias(args.Get(0))
		url   = trimTrailingSeparator(args.Get(1))
		api   = cli.String("api")
		path  = cli.String("path")
	)

	// Support deprecated lookup flag
	if deprecated {
		lookup := strings.ToLower(strings.TrimSpace(cli.String("lookup")))
		switch lookup {
		case "", "auto":
			path = "auto"
		case "path":
			path = "on"
		case "dns":
			path = "off"
		default:
		}
	}

	accessKey, secretKey := fetchAliasKeys(args)
	checkAliasSetSyntax(cli, accessKey, secretKey, deprecated)

	ctx, cancelAliasAdd := context.WithCancel(globalContext)
	defer cancelAliasAdd()

	s3Config, err := BuildS3Config(ctx, url, accessKey, secretKey, api, path)
	fatalIf(err.Trace(cli.Args()...), "Unable to initialize new alias from the provided credentials.")

	msg := setAlias(alias, aliasConfigV10{
		URL:       s3Config.HostURL,
		AccessKey: s3Config.AccessKey,
		SecretKey: s3Config.SecretKey,
		API:       s3Config.Signature,
		Path:      path,
	}) // Add an alias with specified credentials.

	msg.op = "set"
	if deprecated {
		msg.op = "add"
	}

	printMsg(msg)
	return nil
}
