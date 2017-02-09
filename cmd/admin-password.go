/*
 * Minio Client (C) 2016, 2017 Minio, Inc.
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
	"github.com/minio/minio/pkg/probe"
)

var (
	adminPasswordFlags = []cli.Flag{}
)

var adminPasswordCmd = cli.Command{
	Name:   "password",
	Usage:  "Change server access and secret keys.",
	Action: mainAdminPassword,
	Flags:  append(adminPasswordFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc admin {{.Name}} - {{.Usage}}

USAGE:
   mc admin {{.Name}} ALIAS ACCESS_KEY SECRET_KEY

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
    1. Set new credentials of a Minio server represented by its alias 'play'.
       $ mc admin {{.Name}} play/ minio minio123
`,
}

// checkAdminPasswordSyntax - validate all the passed arguments
func checkAdminPasswordSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 3 {
		cli.ShowCommandHelpAndExit(ctx, "password", 1) // last argument is exit code
	}
}

func mainAdminPassword(ctx *cli.Context) error {

	setGlobalsFromContext(ctx)

	checkAdminPasswordSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)
	accessKey := args.Get(1)
	secretKey := args.Get(2)

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Cannot get a configured admin connection.")

	// Change the password of the specified Minio server
	e := client.ServiceSetCredentials(accessKey, secretKey)
	fatalIf(probe.NewError(e), "Unable to set new credentials to '"+aliasedURL+"'")

	return nil
}
