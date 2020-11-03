/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
)

var adminSubnetCmd = cli.Command{
	Name:   "subnet",
	Usage:  "Subnet related commands",
	Action: mainAdminSubnet,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		adminSubnetHealthCmd,
		// adminSubnetRegister to be added
	},
	HideHelpCommand: true,
}

// mainAdminSubnet is the handle for "mc admin subnet" command.
func mainAdminSubnet(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "health", "register" have their own main.
}

// Deprecated - to be removed in a future release
// mainAdminSubnet is the handle for "mc admin subnet" command.
func mainAdminOBD(ctx *cli.Context) error {
	color.Yellow("Deprecated - please use 'mc admin subnet health'")
	return nil
}

var adminHealthCmd = cli.Command{
	Name:               "health",
	Aliases:            []string{"obd"},
	Usage:              "Deprecated - please use 'mc admin subnet health'",
	Action:             mainAdminOBD,
	CustomHelpTemplate: `{{.Usage}}`,
	Hidden:             true,
}
