/*
 * MinIO Client (C) 2016, 2017, 2018 MinIO, Inc.
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
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/probe"
)

var (
	adminCredsFlags = []cli.Flag{}
)

const credsCmdName = "credential"

var adminCredsCmd = cli.Command{
	Name:   credsCmdName,
	Usage:  "change admin server access and secret keys",
	Action: mainAdminCreds,
	Flags:  append(adminCredsFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET ACCESSKEY SECRETKEY

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. Set new **admin** credential of a MinIO server represented by its alias 'alias'.
       $ {{.HelpName}} alias minio minio123

`,
}

// checkAdminCredsSyntax - validate all the passed arguments
func checkAdminCredsSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelpAndExit(ctx, credsCmdName, 1) // last argument is exit code
	}
}

func mainAdminCreds(ctx *cli.Context) error {

	setGlobalsFromContext(ctx)

	checkAdminCredsSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	// TODO: if accessKey and secretKey are not supplied we should
	// display the existing credential. This needs GetCredential
	// support from MinIO server.
	aliasedURL := args.First()
	accessKey, secretKey := args.Get(1), args.Get(2)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Change the credential of the specified MinIO server
	e := client.SetAdminCredentials(accessKey, secretKey)
	fatalIf(probe.NewError(e), "Unable to set new credential to '"+aliasedURL+"'.")

	return nil
}
