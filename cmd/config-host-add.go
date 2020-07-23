/*
 * MinIO Client (C) 2017 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio/pkg/console"
	"golang.org/x/crypto/ssh/terminal"
)

const cred = "YellowItalics"

var hostAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "lookup",
		Value: "auto",
		Usage: "bucket lookup supported by the server. Valid options are '[dns,path,auto]'",
	},
	cli.StringFlag{
		Name:  "api",
		Usage: "API signature. Valid options are '[S3v4, S3v2]'",
	},
}
var configHostAddCmd = cli.Command{
	Name:            "add",
	ShortName:       "a",
	Usage:           "add a new host to configuration file",
	Action:          mainConfigHostAdd,
	Before:          setGlobalsFromContext,
	Flags:           append(hostAddFlags, globalFlags...),
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
     {{.Prompt}} {{.HelpName}} myminio http://localhost:9000 minio minio123 --api "s3v4" --lookup "dns"
     {{.EnableHistory}}

  3. Add Amazon S3 storage service under "mys3" alias. For security reasons turn off bash history momentarily.
     {{.DisableHistory}}
     {{.Prompt}} {{.HelpName}} mys3 https://s3.amazonaws.com \
                 BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
     {{.EnableHistory}}

  4. Add Amazon S3 storage service under "mys3" alias, prompting for keys.
     {{.Prompt}} {{.HelpName}} mys3 https://s3.amazonaws.com --api "s3v4" --lookup "dns"
     Enter Access Key: BKIKJAA5BMMU2RHO6IBB
     Enter Secret Key: V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12

  5. Add Amazon S3 storage service under "mys3" alias using piped keys.
     {{.DisableHistory}}
     {{.Prompt}} echo -e "BKIKJAA5BMMU2RHO6IBB\nV8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12" | \
                 {{.HelpName}} mys3 https://s3.amazonaws.com --api "s3v4" --lookup "dns"
     {{.EnableHistory}}
`,
}

// checkConfigHostAddSyntax - verifies input arguments to 'config host add'.
func checkConfigHostAddSyntax(ctx *cli.Context, accessKey string, secretKey string) {
	args := ctx.Args()
	argsNr := len(args)
	if argsNr > 4 || argsNr < 2 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for host add command.")
	}

	alias := cleanAlias(args.Get(0))
	url := args.Get(1)
	api := ctx.String("api")
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

	if !isValidLookup(bucketLookup) {
		fatalIf(errInvalidArgument().Trace(bucketLookup),
			"Unrecognized bucket lookup. Valid options are `[dns,auto, path]`.")
	}
}

// addHost - add a host config.
func addHost(alias string, hostCfgV9 hostConfigV9) {
	mcCfgV9, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config `"+mustGetMcConfigPath()+"`.")

	// Add new host.
	mcCfgV9.Hosts[alias] = hostCfgV9

	err = saveMcConfig(mcCfgV9)
	fatalIf(err.Trace(alias), "Unable to update hosts in config version `"+mustGetMcConfigPath()+"`.")

	printMsg(hostMessage{
		op:        "add",
		Alias:     alias,
		URL:       hostCfgV9.URL,
		AccessKey: hostCfgV9.AccessKey,
		SecretKey: hostCfgV9.SecretKey,
		API:       hostCfgV9.API,
		Lookup:    hostCfgV9.Lookup,
	})
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
func BuildS3Config(ctx context.Context, url, accessKey, secretKey, api, lookup string) (*Config, *probe.Error) {

	s3Config := NewS3Config(url, &hostConfigV9{
		AccessKey: accessKey,
		SecretKey: secretKey,
		URL:       url,
		Lookup:    lookup,
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
		return nil, err.Trace(url, accessKey, secretKey, api, lookup)
	}

	s3Config.Signature = api
	// Success.
	return s3Config, nil
}

// fetchHostKeys - returns the user accessKey and secretKey
func fetchHostKeys(args cli.Args) (string, string) {
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

func mainConfigHostAdd(cli *cli.Context) error {

	console.SetColor("HostMessage", color.New(color.FgGreen))
	var (
		args   = cli.Args()
		alias  = cleanAlias(args.Get(0))
		url    = trimTrailingSeparator(args.Get(1))
		api    = cli.String("api")
		lookup = cli.String("lookup")
	)
	accessKey, secretKey := fetchHostKeys(args)
	checkConfigHostAddSyntax(cli, accessKey, secretKey)

	ctx, cancelHostAdd := context.WithCancel(globalContext)
	defer cancelHostAdd()

	s3Config, err := BuildS3Config(ctx, url, accessKey, secretKey, api, lookup)
	fatalIf(err.Trace(cli.Args()...), "Unable to initialize new config from the provided credentials.")

	addHost(alias, hostConfigV9{
		URL:       s3Config.HostURL,
		AccessKey: s3Config.AccessKey,
		SecretKey: s3Config.SecretKey,
		API:       s3Config.Signature,
		Lookup:    lookup,
	}) // Add a host with specified credentials.
	return nil
}
