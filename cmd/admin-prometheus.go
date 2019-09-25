/*
 * MinIO Client (C) 2019 MinIO, Inc.
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

var adminPrometheusCmd = cli.Command{
	Name:            "prometheus",
	Usage:           "manages prometheus config",
	Action:          mainAdminPrometheus,
	Before:          setGlobalsFromContext,
	Flags:           globalFlags,
	HideHelpCommand: true,
	Subcommands: []cli.Command{
		adminPrometheusGenerateCmd,
	},
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
`,
}

// mainAdminPrometheus is the handle for "mc admin prometheus" command.
func mainAdminPrometheus(ctx *cli.Context) error {
	cli.ShowCommandHelp(ctx, ctx.Args().First())
	return nil
}
