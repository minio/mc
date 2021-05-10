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
