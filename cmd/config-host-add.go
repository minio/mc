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
	urlpkg "net/url"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
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

func mainConfigHostAdd(ctx *cli.Context) error {
	checkConfigHostAddSyntax(ctx)

	console.SetColor("HostMessage", color.New(color.FgGreen))

	args := ctx.Args()
	alias := args.Get(0)
	url := args.Get(1)
	accessKey := args.Get(2)
	secretKey := args.Get(3)
	api := args.Get(4)

	parsedURL, _ := urlpkg.Parse(url)

	if isGoogle(parsedURL.Host) {
		if api == "S3v4" {
			fatalIf(errInvalidArgument().Trace(api), "Unsupported API signature for url. Please use `mc config host add "+alias+" "+url+" "+accessKey+" "+secretKey+" S3v2` instead.")
		}
		api = "S3v2"
	}

	if api == "" {
		api = "S3v4"
	}
	hostCfg := hostConfigV8{
		URL:       url,
		AccessKey: accessKey,
		SecretKey: secretKey,
		API:       api,
	}
	addHost(alias, hostCfg) // Add a host with specified credentials.
	return nil
}
