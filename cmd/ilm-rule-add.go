// Copyright (c) 2015-2022 MinIO, Inc.
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
	"github.com/minio/pkg/v3/console"
)

var ilmAddCmd = cli.Command{
	Name:         "add",
	Usage:        "add a lifecycle configuration rule for a bucket",
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
  1. Add a lifecycle rule with a transition and a noncurrent version transition action for objects with prefix doc/ whose size is greater than 1MiB in mybucket.
     Tiers must exist in MinIO. Use existing tiers or add new tiers.

     {{.Prompt}} mc ilm tier add minio myminio MINIOTIER-1 --endpoint https://warm-minio-1.com \
         --access-key ACCESSKEY --secret-key SECRETKEY --bucket bucket1 --prefix prefix1

     {{.Prompt}} {{.HelpName}} --prefix "doc/" --size-gt 1MiB --transition-days "90" --transition-tier "MINIOTIER-1" \
         --noncurrent-transition-days "45" --noncurrent-transition-tier "MINIOTIER-1" \
         myminio/mybucket/

  2. Add a lifecycle rule with an expiration action for all objects in mybucket.
     {{.Prompt}} {{.HelpName}} --expire-days "200" myminio/mybucket

  3. Add a lifecycle rule with an expiration and a noncurrent version expiration action for all objects with prefix doc/ in mybucket.
     {{.Prompt}} {{.HelpName}} --prefix "doc/" --expire-days "300" --noncurrent-expire-days "100" \
          myminio/mybucket/
`,
}

var ilmAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "prefix",
		Usage: "object prefix",
	},
	cli.StringFlag{
		Name:  "tags",
		Usage: "key value pairs of the form '<key1>=<value1>&<key2>=<value2>&<key3>=<value3>'",
	},
	cli.StringFlag{
		Name:  "size-lt",
		Usage: "objects with size less than this value will be selected for the lifecycle action",
	},
	cli.StringFlag{
		Name:  "size-gt",
		Usage: "objects with size greater than this value will be selected for the lifecycle action",
	},
	cli.StringFlag{
		Name:   "expiry-date",
		Usage:  "format 'YYYY-MM-DD' the date of expiration",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "expiry-days",
		Usage:  "the number of days to expiration",
		Hidden: true,
	},
	cli.StringFlag{
		Name:  "expire-days",
		Usage: "number of days to expire",
	},
	cli.BoolFlag{
		Name:   "expired-object-delete-marker",
		Usage:  "remove delete markers with no parallel versions",
		Hidden: true,
	},
	cli.BoolFlag{
		Name:  "expire-delete-marker",
		Usage: "expire zombie delete markers",
	},
	cli.StringFlag{
		Name:   "transition-date",
		Usage:  "format 'YYYY-MM-DD' for the date to transition",
		Hidden: true,
	},
	cli.StringFlag{
		Name:  "transition-days",
		Usage: "number of days to transition",
	},
	cli.StringFlag{
		Name:   "storage-class",
		Usage:  "storage class for current version to transition into. MinIO supports tiers configured via `mc-admin-tier-add`.",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "tier",
		Usage:  "remote tier where current versions transition to",
		Hidden: true,
	},
	cli.StringFlag{
		Name:  "transition-tier",
		Usage: "remote tier name to transition",
	},
	cli.IntFlag{
		Name:   "noncurrentversion-expiration-days",
		Usage:  "the number of days to remove noncurrent versions",
		Hidden: true,
	},
	cli.StringFlag{
		Name:  "noncurrent-expire-days",
		Usage: "number of days to expire noncurrent versions",
	},
	cli.IntFlag{
		Name:   "newer-noncurrentversions-expiration",
		Usage:  "the number of noncurrent versions to retain",
		Hidden: true,
	},
	cli.IntFlag{
		Name:  "noncurrent-expire-newer",
		Usage: "number of newer noncurrent versions to retain",
	},
	cli.IntFlag{
		Name:   "noncurrentversion-transition-days",
		Usage:  "the number of days to transition noncurrent versions",
		Hidden: true,
	},
	cli.IntFlag{
		Name:  "noncurrent-transition-days",
		Usage: "number of days to transition noncurrent versions",
	},
	cli.IntFlag{
		Name:   "newer-noncurrentversions-transition",
		Usage:  "the number of noncurrent versions to retain. If there are this many more recent noncurrent versions they will be transitioned",
		Hidden: true,
	},
	cli.IntFlag{
		Name:   "noncurrent-transition-newer",
		Usage:  "number of noncurrent versions to retain in hot tier",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "noncurrentversion-transition-storage-class",
		Usage:  "storage class for noncurrent versions to transition into",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "noncurrentversion-tier",
		Usage:  "remote tier where noncurrent versions transition to",
		Hidden: true,
	},
	cli.StringFlag{
		Name:  "noncurrent-transition-tier",
		Usage: "remote tier name to transition",
	},
	cli.BoolFlag{
		Name:  "expire-all-object-versions",
		Usage: "expire all object versions",
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
	fatalIf(probe.NewError(e), "Unable to encode as JSON.")
	return string(msgBytes)
}

// Validate user given arguments
func checkILMAddSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, globalErrorExitStatus)
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
	lfcCfg, _, err := client.GetLifecycle(ctx)
	if err != nil {
		if e := err.ToGoError(); minio.ToErrorResponse(e).Code == "NoSuchLifecycleConfiguration" {
			lfcCfg = lifecycle.NewConfiguration()
		} else {
			fatalIf(err.Trace(args...), "Unable to fetch lifecycle rules for "+urlStr)
		}
	}

	opts, err := ilm.GetLifecycleOptions(cliCtx)
	fatalIf(err.Trace(args...), "Unable to generate new lifecycle rules for the input")

	newRule, err := opts.ToILMRule()
	fatalIf(err.Trace(args...), "Unable to generate new lifecycle rules for the input")

	lfcCfg.Rules = append(lfcCfg.Rules, newRule)

	fatalIf(client.SetLifecycle(ctx, lfcCfg).Trace(urlStr), "Unable to add this lifecycle rule")

	printMsg(ilmAddMessage{
		Status: "success",
		Target: urlStr,
		ID:     opts.ID,
	})

	return nil
}
