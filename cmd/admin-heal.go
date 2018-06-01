/*
 * Minio Client (C) 2017, 2018 Minio, Inc.
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
	"path/filepath"

	"github.com/fatih/color"
	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/madmin"
)

var adminHealFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "recursive, r",
		Usage: "Heal recursively",
	},
	cli.BoolFlag{
		Name:  "dry-run, n",
		Usage: "Only inspect data, but do not mutate",
	},
	cli.BoolFlag{
		Name:  "force-start, f",
		Usage: "Force start a new heal sequence",
	},
}

var adminHealCmd = cli.Command{
	Name:            "heal",
	Usage:           "Heal disks, buckets and objects on Minio server",
	Action:          mainAdminHeal,
	Before:          setGlobalsFromContext,
	Flags:           append(adminHealFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
    1. To format newly replaced disks in a Minio server with alias 'play'
       $ {{.HelpName}} play

    2. Heal 'testbucket' in a Minio server with alias 'play'
       $ {{.HelpName}} play/testbucket/

    3. Heal all objects under 'dir' prefix
       $ {{.HelpName}} --recursive play/testbucket/dir/

    4. Issue a dry-run heal operation to inspect objects health but not heal them
       $ {{.HelpName}} --dry-run play

    5. Issue a dry-run heal operation to inspect objects health under 'dir' prefix
       $ {{.HelpName}} --recursive --dry-run play/testbucket/dir/

`,
}

func checkAdminHealSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "heal", 1) // last argument is exit code
	}
}

// mainAdminHeal - the entry function of heal command
func mainAdminHeal(ctx *cli.Context) error {

	// Check for command syntax
	checkAdminHealSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	console.SetColor("Heal", color.New(color.FgGreen, color.Bold))
	console.SetColor("HealUpdateUI", color.New(color.FgYellow, color.Bold))

	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Cannot initialize admin client.")
		return nil
	}

	// Compute bucket and object from the aliased URL
	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket, prefix := splits[1], splits[2]

	opts := madmin.HealOpts{
		Recursive: ctx.Bool("recursive"),
		DryRun:    ctx.Bool("dry-run"),
	}
	forceStart := ctx.Bool("force-start")
	healStart, _, herr := client.Heal(bucket, prefix, opts, "", forceStart)
	errorIf(probe.NewError(herr), "Failed to start heal sequence.")

	ui := uiData{
		Bucket:                bucket,
		Prefix:                prefix,
		Client:                client,
		ClientToken:           healStart.ClientToken,
		ForceStart:            forceStart,
		HealOpts:              &opts,
		ObjectsByOnlineDrives: make(map[int]int64),
		HealthCols:            make(map[col]int64),
		CurChan:               cursorAnimate(),
	}
	errorIf(
		probe.NewError(ui.DisplayAndFollowHealStatus()),
		"Unable to display follow heal status.",
	)

	return nil
}
