/*
 * MinIO Client (C) 2016, 2017 MinIO, Inc.
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

var (
	adminFlags = []cli.Flag{}
)

const (
	// dot represents a list item, for eg. server status - online (green) or offline (red)
	dot = "●"
	// check represents successful operation
	check = "✔"
)

var adminCmdSubcommands = []cli.Command{
	adminServiceCmd,
	adminServerUpdateCmd,
	adminInfoCmd,
	adminUserCmd,
	adminGroupCmd,
	adminPolicyCmd,
	adminConfigCmd,
	adminHealCmd,
	adminProfileCmd,
	adminBwInfoCmd,
	adminTopCmd,
	adminTraceCmd,
	adminConsoleCmd,
	adminPrometheusCmd,
	adminKMSCmd,
	adminHealthCmd,
	adminSubnetCmd,
	adminBucketCmd,
}

var adminCmd = cli.Command{
	Name:            "admin",
	Usage:           "manage MinIO servers",
	Action:          mainAdmin,
	Subcommands:     adminCmdSubcommands,
	HideHelpCommand: true,
	Before:          setGlobalsFromContext,
	Flags:           append(adminFlags, globalFlags...),
}

const dateTimeFormatFilename = "2006-01-02T15-04-05.999999-07-00"

// mainAdmin is the handle for "mc admin" command.
func mainAdmin(ctx *cli.Context) error {
	commandNotFound(ctx, adminCmdSubcommands)
	return nil
	// Sub-commands like "service", "heal", "top" have their own main.
}
