/*
 * MinIO Client (C) 2016 MinIO, Inc.
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
	eventFlags = []cli.Flag{}
)

var eventCmd = cli.Command{
	Name:            "event",
	Usage:           "manage object notifications",
	HideHelpCommand: true,
	Action:          mainEvent,
	Before:          setGlobalsFromContext,
	Flags:           append(eventFlags, globalFlags...),
	Subcommands: []cli.Command{
		eventAddCmd,
		eventRemoveCmd,
		eventListCmd,
	},
}

// mainEvent is the handle for "mc event" command.
func mainEvent(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "add", "remove", "list" have their own main.
}
