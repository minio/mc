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
	"crypto/rand"
	"encoding/hex"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// configHostAdd command flags.
var (
	configHostAddFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "encrypt",
			Usage: "Enable client-side encryption of objects",
		},
		cli.StringFlag{
			Name:  "key",
			Usage: "Specify a symmetric key",
		},
		cli.StringFlag{
			Name:  "public-key",
			Usage: "Specify the path of the public key",
		},
		cli.StringFlag{
			Name:  "private-key",
			Usage: "Specify the path of the private key",
		},
	}
)

var configHostAddCmd = cli.Command{
	Name:            "add",
	ShortName:       "a",
	Usage:           "Add a new host to configuration file.",
	Action:          mainConfigHostAdd,
	Before:          setGlobalsFromContext,
	Flags:           append(configHostAddFlags, globalFlags...),
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

  3. Encrypt objects Amazon S3 storage service under "mys3" alias using a symmetric key.
     $ set +o history
     $ {{.HelpName}} mys3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 \
     		--encrypt --key="my-secret-key-16"
     $ set -o history

  4. Encrypt objects Amazon S3 storage service under "mys3" alias unsing an asymmetric key.
     $ set +o history
     $ {{.HelpName}} mys3 https://s3.amazonaws.com BKIKJAA5BMMU2RHO6IBB V8f1CwQqAcwo80UEIJEjc5gVQUSSx5ohQ9GSrr12 \
     		--encrypt --public-key=/path/to/public.key --private-key=/path/to/private.key
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
		fatalIf(errDummy().Trace(alias), "Invalid alias `"+alias+"`.")
	}

	if !isValidHostURL(url) {
		fatalIf(errDummy().Trace(url),
			"Invalid URL `"+url+"`.")
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

	encrypt := ctx.Bool("encrypt")
	symKey := ctx.String("key")
	privateKeyPath := ctx.String("private-key")
	publicKeyPath := ctx.String("public-key")

	// If --key or --private-key or --public-key are specified but --encrypt is not passed
	if !encrypt && (symKey != "" || privateKeyPath != "" || publicKeyPath != "") {
		fatalIf(errInvalidArgument().Trace(),
			"You should explicitly pass --encrypt flag")
	}

	// If --encrypt and --key are passed, reject --public-key and --private-key flags
	if symKey != "" && (privateKeyPath != "" || publicKeyPath != "") {
		fatalIf(errInvalidArgument().Trace(),
			"You cannot pass both symmetric and asymmetric keys.")
	}

	// Check if both of private and public keys are specified
	if (privateKeyPath != "" && publicKeyPath == "") || (privateKeyPath == "" && publicKeyPath != "") {
		fatalIf(errInvalidArgument().Trace(),
			"Both private-key & public-key should be specified.")
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
	if api == "" {
		api = "S3v4"
	}

	encryptConfig := encryptionConfigV9{}

	if ctx.Bool("encrypt") {
		symKey := ctx.String("key")
		privateKeyPath := ctx.String("private-key")
		publicKeyPath := ctx.String("public-key")

		if symKey != "" {
			// Enable AES encryption
			encryptConfig.AES = AESConfigV9{
				Enable: true,
				Key:    symKey,
			}
		} else if privateKeyPath != "" && publicKeyPath != "" {
			// Enable RSA encryption
			encryptConfig.RSA = RSAConfigV9{
				Enable:     true,
				PublicKey:  publicKeyPath,
				PrivateKey: privateKeyPath,
			}
		} else {
			// The user asked to enable encryption without specifying
			// any encryption symmetric or asymmetric keys. In that case,
			// we generate a custom AES key.
			randomBytes := make([]byte, 16)
			_, e := rand.Read(randomBytes)
			fatalIf(probe.NewError(e), "Unable to generate a random key.")
			genSymKey := hex.EncodeToString(randomBytes)
			encryptConfig.AES = AESConfigV9{
				Enable: true,
				Key:    genSymKey,
			}
		}
	}

	hostCfg := hostConfigV9{
		URL:        url,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
		API:        api,
		Encryption: encryptConfig,
	}

	addHost(alias, hostCfg) // Add a host with specified credentials.
	return nil
}
