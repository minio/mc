/*
 * MinIO Client, (C) 2015 MinIO, Inc.
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
)

var (
	sessionFlags = []cli.Flag{}
)

// Resume interrupted operations.
var sessionCmd = cli.Command{
	Name:            "session",
	Usage:           "resume interrupted operations",
	Action:          mainSession,
	Flags:           append(sessionFlags, globalFlags...),
	Before:          setGlobalsFromContext,
	HideHelpCommand: true,
	Subcommands: []cli.Command{
		sessionList,
		sessionClear,
		sessionResume,
	},
}

// mainSession - handle for the 'mc session' command.
func mainSession(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "list", "clear", "resume" have their own main.
}
