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
	"github.com/minio/minio/pkg/console"
)

var adminUserSACmd = cli.Command{
	Name:   "sa",
	Usage:  "manage service accounts",
	Action: mainAdminUserSA,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
	Subcommands: []cli.Command{
		adminUserSAGenerateCmd,
		adminUserSAShowCmd,
	},
}

func setSACommandColors() {
	console.SetColor("SA", color.New(color.FgCyan, color.Bold))
	console.SetColor("AccessKey", color.New(color.FgYellow))
	console.SetColor("SecretKey", color.New(color.FgRed))
	console.SetColor("SessionToken", color.New(color.FgBlue))
}

func mainAdminUserSA(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
	// Sub-commands like "get", "set" have their own main.
}
