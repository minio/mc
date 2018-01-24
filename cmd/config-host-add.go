/*
 * Minio Client (C) 2017 Minio, Inc.
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
	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

var configHostAddCmd = cli.Command{
	Name:            "add",
	ShortName:       "a",
	Usage:           "Add a new host to configuration file.",
	Action:          mainConfigHostAdd,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS URL ACCESS-KEY SECRET-KEY [API]

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Add Amazon S3 storage service under "mys3" alias. For security reasons turn off bash history momentarily.
     $ set +o history
     $ {{.HelpName}} mys3 https://s3.amazonaws.com \
                 BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
     $ set -o history

  2. Add Amazon S3 accelerated storage service under "mys3-accel" alias. For security reasons turn off bash history momentarily.
     $ set +o history
     $ {{.HelpName}} mys3-accel https://s3-accelerate.amazonaws.com \
                 BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12
     $ set -o history

  3. Add Amazon S3 IAM temporary credentials with limited access, please make sure to override the signature probe by explicitly
     providing the signature type.
     $ set +o history
     $ {{.HelpName}} mys3-iam https://s3.amazonaws.com \
                 BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 s3v4
     $ set -o history

`,
}

// checkConfigHostAddSyntax - verifies input arguments to 'config host add'.
func checkConfigHostAddSyntax(ctx *cli.Context) {
	args := ctx.Args()
	argsNr := len(args)
	if argsNr < 4 || argsNr > 5 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for host add command.")
	}

	alias := args.Get(0)
	url := args.Get(1)
	accessKey := args.Get(2)
	secretKey := args.Get(3)
	api := args.Get(4)

	if !isValidAlias(alias) {
		fatalIf(errInvalidAlias(alias), "Invalid alias")
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
}

// addHost - add a host config.
func addHost(alias string, hostCfgV8 hostConfigV8) {
	mcCfgV8, err := loadMcConfig()
	fatalIf(err.Trace(globalMCConfigVersion), "Unable to load config `"+mustGetMcConfigPath()+"`.")

	// Add new host.
	mcCfgV8.Hosts[alias] = hostCfgV8

	err = saveMcConfig(mcCfgV8)
	fatalIf(err.Trace(alias), "Unable to update hosts in config version `"+mustGetMcConfigPath()+"`.")

	printMsg(hostMessage{
		op:        "add",
		Alias:     alias,
		URL:       hostCfgV8.URL,
		AccessKey: hostCfgV8.AccessKey,
		SecretKey: hostCfgV8.SecretKey,
		API:       hostCfgV8.API,
	})
}

// probeS3Signature - auto probe S3 server signature: issue a Stat call
// using v4 signature then v2 in case of failure.
func probeS3Signature(accessKey, secretKey, url string) (string, *probe.Error) {
	// Test s3 connection for API auto probe
	s3Config := &Config{
		// S3 connection parameters
		Insecure:  globalInsecure,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Signature: "s3v4",
		HostURL:   urlJoinPath(url, "probe-bucket-sign"),
	}

	s3Client, err := s3New(s3Config)
	if err != nil {
		return "", err
	}

	if _, err = s3Client.Stat(false, false); err != nil {
		switch err.ToGoError().(type) {
		case BucketDoesNotExist:
			// Bucket doesn't exist, means signature probing worked V4.
		default:
			// Attempt with signature v2, since v4 seem to have failed.
			s3Config.Signature = "s3v2"
			s3Client, err = s3New(s3Config)
			if err != nil {
				return "", err
			}
			if _, err = s3Client.Stat(false, false); err != nil {
				switch err.ToGoError().(type) {
				case BucketDoesNotExist:
					// Bucket doesn't exist, means signature probing worked with V2.
				default:
					return "", err
				}
			}
		}
	}

	return s3Config.Signature, nil
}

// buildS3Config constructs an S3 Config and does
// signature auto-probe when needed.
func buildS3Config(args cli.Args) (*Config, *probe.Error) {
	var (
		url       = args.Get(0)
		accessKey = args.Get(1)
		secretKey = args.Get(2)
		api       = args.Get(3)
	)

	s3Config := newS3Config(url, &hostConfigV8{
		AccessKey: accessKey,
		SecretKey: secretKey,
		URL:       url,
	})

	// If api is provided we do not auto probe signature, this is
	// required in situations when signature type is provided by the user.
	if api != "" {
		s3Config.Signature = api
		return s3Config, nil
	}

	// Probe S3 signature version
	api, err := probeS3Signature(accessKey, secretKey, url)
	if err != nil {
		return nil, err.Trace(args...)
	}

	s3Config.Signature = api

	// Success.
	return s3Config, nil
}

func mainConfigHostAdd(ctx *cli.Context) error {
	checkConfigHostAddSyntax(ctx)

	console.SetColor("HostMessage", color.New(color.FgGreen))

	s3Config, err := buildS3Config(ctx.Args().Tail())
	fatalIf(err.Trace(ctx.Args()...), "Unable to initialize new config from the provided credentials")

	addHost(ctx.Args().Get(0), hostConfigV8{
		URL:       s3Config.HostURL,
		AccessKey: s3Config.AccessKey,
		SecretKey: s3Config.SecretKey,
		API:       s3Config.Signature,
	}) // Add a host with specified credentials.
	return nil
}
