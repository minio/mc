/*
 * MinIO Client (C) 2014, 2015 MinIO, Inc.
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

import "github.com/minio/cli"

var configCmd = cli.Command{
	Name:  "config",
	Usage: "configure MinIO client",
	Action: func(ctx *cli.Context) error {
		cli.ShowCommandHelp(ctx, ctx.Args().First())
		return nil
	},
	Hidden:          true,
	Before:          setGlobalsFromContext,
	HideHelpCommand: true,
	Flags:           globalFlags,
	Subcommands: []cli.Command{
		configHostCmd,
	},
}

var configHostCmd = cli.Command{
	Name:  "host",
	Usage: "add, remove and list hosts in configuration file",
	Action: func(ctx *cli.Context) error {
		cli.ShowCommandHelp(ctx, ctx.Args().First())
		return nil
	},
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		configHostAddCmd,
		configHostRemoveCmd,
		configHostListCmd,
	},
	HideHelpCommand: true,
}

var configHostAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "lookup",
		Value: "auto",
		Usage: "bucket lookup supported by the server. Valid options are '[dns, path, auto]'",
	},
	cli.StringFlag{
		Name:  "api",
		Usage: "API signature. Valid options are '[S3v4, S3v2]'",
	},
}

var configHostAddCmd = cli.Command{
	Name:      "add",
	ShortName: "a",
	Usage:     "add a new host to configuration file",
	Action: func(cli *cli.Context) error {
		return mainAliasSet(cli, true)
	},
	Before:          setGlobalsFromContext,
	Flags:           append(configHostAddFlags, globalFlags...),
	HideHelpCommand: true,
}

var configHostListCmd = cli.Command{
	Name:      "list",
	ShortName: "ls",
	Usage:     "list hosts in configuration file",
	Action: func(cli *cli.Context) error {
		return mainAliasList(cli, true)
	},
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
}

var configHostRemoveCmd = cli.Command{
	Name:      "remove",
	ShortName: "rm",
	Usage:     "remove a host from configuration file",
	Action: func(cli *cli.Context) error {
		return mainAliasRemove(cli, true)
	},
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
}
