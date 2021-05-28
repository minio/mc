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

import (
	"context"

	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/cmd/ilm"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/minio/pkg/console"
)

var ilmAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a lifecycle configuration rule to existing (if any) rule(s) on a bucket",
	Action:       mainILMAdd,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(ilmAddFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
DESCRIPTION:
  Add a lifecycle configuration rule.

EXAMPLES:
  1. Add expiration rule on mybucket.
     {{.Prompt}} {{.HelpName}} --expiry-days "200" myminio/mybucket

  2. Add expiry and transition date rules on a prefix in mybucket.
     {{.Prompt}} {{.HelpName}} --expiry-date "2025-09-17" --transition-date "2025-05-01" \
          --storage-class "GLACIER" s3/mybucket/doc

  3. Add expiry and transition days rules on a prefix in mybucket.
     {{.Prompt}} {{.HelpName}} --expiry-days "300" --transition-days "200" \
          --storage-class "GLACIER" s3/mybucket/doc
`,
}

var ilmAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "tags",
		Usage: "format '<key1>=<value1>&<key2>=<value2>&<key3>=<value3>', multiple values allowed for multiple key/value pairs",
	},
	cli.StringFlag{
		Name:  "expiry-date",
		Usage: "format 'YYYY-MM-DD' the date of expiration",
	},
	cli.StringFlag{
		Name:  "expiry-days",
		Usage: "the number of days to expiration",
	},
	cli.StringFlag{
		Name:  "transition-date",
		Usage: "format 'YYYY-MM-DD' for the date to transition",
	},
	cli.StringFlag{
		Name:  "transition-days",
		Usage: "the number of days to transition",
	},
	cli.StringFlag{
		Name:  "storage-class",
		Usage: "storage class for transition (STANDARD_IA, ONEZONE_IA, GLACIER. Etc).",
	},
	cli.BoolFlag{
		Name:  "disable",
		Usage: "disable the rule",
	},
	cli.BoolFlag{
		Name:  "expired-object-delete-marker",
		Usage: "remove delete markers with no parallel versions",
	},
	cli.IntFlag{
		Name:  "noncurrentversion-expiration-days",
		Usage: "the number of days to remove noncurrent versions",
	},
	cli.IntFlag{
		Name:  "noncurrentversion-transition-days",
		Usage: "the number of days to transition noncurrent versions",
	},
	cli.StringFlag{
		Name:  "noncurrentversion-transition-storage-class",
		Usage: "the transition storage class for noncurrent versions",
	},
}

type ilmAddMessage struct {
	Status string `json:"status"`
	Target string `json:"target"`
	ID     string `json:"id"`
}

func (i ilmAddMessage) String() string {
	return console.Colorize(ilmThemeResultSuccess, "Lifecycle configuration rule added with ID `"+i.ID+"` to "+i.Target+".")
}

func (i ilmAddMessage) JSON() string {
	msgBytes, e := json.MarshalIndent(i, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(msgBytes)
}

// Validate user given arguments
func checkILMAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "add", globalErrorExitStatus)
	}
}

// Calls SetBucketLifecycle with the XML representation of lifecycleConfiguration type.
func mainILMAdd(cliCtx *cli.Context) error {
	ctx, cancelILMAdd := context.WithCancel(globalContext)
	defer cancelILMAdd()

	checkILMAddSyntax(cliCtx)
	setILMDisplayColorScheme()
	args := cliCtx.Args()
	urlStr := args.Get(0)

	client, err := newClient(urlStr)
	fatalIf(err.Trace(urlStr), "Unable to initialize client for "+urlStr)

	// Configuration that is already set.
	lfcCfg, err := client.GetLifecycle(ctx)
	if err != nil {
		if e := err.ToGoError(); minio.ToErrorResponse(e).Code == "NoSuchLifecycleConfiguration" {
			lfcCfg = lifecycle.NewConfiguration()
		} else {
			fatalIf(err.Trace(args...), "Unable to fetch lifecycle rules for "+urlStr)
		}
	}

	opts := ilm.GetLifecycleOptions(cliCtx)
	lfcCfg, err = opts.ToConfig(lfcCfg)
	fatalIf(err.Trace(args...), "Unable to generate new lifecycle rules for the input")

	fatalIf(client.SetLifecycle(ctx, lfcCfg).Trace(urlStr), "Unable to set new lifecycle rules")

	printMsg(ilmAddMessage{
		Status: "success",
		Target: urlStr,
		ID:     opts.ID,
	})

	return nil
}
