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

import "github.com/minio/cli"

var (
	eventsFlags = []cli.Flag{}
)

var eventsCmd = cli.Command{
	Name:            "events",
	Usage:           "Manage object notifications.",
	HideHelpCommand: true,
	Action:          mainEvents,
	Before:          setGlobalsFromContext,
	Flags:           append(eventsFlags, globalFlags...),
	Subcommands: []cli.Command{
		eventsAddCmd,
		eventsRemoveCmd,
		eventsListCmd,
	},
}

// mainEvents is the handle for "mc events" command.
func mainEvents(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "add", "remove", "list" have their own main.
}
