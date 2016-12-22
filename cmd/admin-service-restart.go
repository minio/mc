/*
 * Minio Client (C) 2016 Minio, Inc.
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
	"net/url"

	"github.com/minio/cli"
	"github.com/minio/minio/pkg/madmin"
	"github.com/minio/minio/pkg/probe"
)

var (
	adminServiceRestartFlags = []cli.Flag{
		cli.BoolFlag{
			Name:  "all",
			Usage: "Control all nodes in the cluster",
		},
	}
)

var adminServiceRestartCmd = cli.Command{
	Name:   "restart",
	Usage:  "Restart a minio server",
	Action: mainAdminServiceRestart,
	Flags:  append(adminServiceRestartFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
   mc admin service {{.Name}} - {{.Usage}}

USAGE:
   mc admin service {{.Name}} ALIAS

FLAGS:
  {{range .Flags}}{{.}}
  {{end}}
EXAMPLES:
    1. Restart a Minio server represented by its alias 'play'.
       $ mc admin service {{.Name}} play/
`,
}

// checkAdminServiceRestartSyntax - validate all the passed arguments
func checkAdminServiceRestartSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 2 {
		cli.ShowCommandHelpAndExit(ctx, "restart", 1) // last argument is exit code
	}
}

func mainAdminServiceRestart(ctx *cli.Context) error {

	setGlobalsFromContext(ctx)
	checkAdminServiceRestartSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Fetch the server's address stored in config
	_, _, hostCfg, err := expandAlias(aliasedURL)
	if err != nil {
		return err.ToGoError()
	}

	// Check if alias exists
	if hostCfg == nil {
		fatalIf(errInvalidAliasedURL(aliasedURL).Trace(aliasedURL), "The specified alias is not found.")
	}

	// Parse the server address
	url, e := url.Parse(hostCfg.URL)
	if e != nil {
		return e
	}

	isSSL := (url.Scheme == "https")

	// Create a new Minio Admin Client
	client, e := madmin.New(url.Host, hostCfg.AccessKey, hostCfg.SecretKey, isSSL)
	if e != nil {
		return e
	}

	// Restart the specified Minio server
	e = client.ServiceRestart()
	fatalIf(probe.NewError(e), "Cannot restart server.")

	return nil
}
