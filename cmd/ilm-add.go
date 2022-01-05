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
  1. Add a lifecycle rule with an expiration action for all objects in mybucket.
     {{.Prompt}} {{.HelpName}} --expiry-days "200" myminio/mybucket

  2. Add a lifecycle rule with a transition and a noncurrent version transition action for objects with prefix doc/ in mybucket.
     {{.Prompt}} {{.HelpName}} --transition-days "90" --storage-class "MINIOTIER-1" \
          --noncurrentversion-transition-days "45" --noncurrentversion-transition-storage-class "MINIOTIER2" \
          myminio/mybucket/doc

  3. Add a lifecycle rule with an expiration and a noncurrent version expiration action for all objects with prefix doc/ in mybucket.
     {{.Prompt}} {{.HelpName}} --expiry-days "300" --noncurrentversion-expiration-days "100" \
          myminio/mybucket/doc
`,
}

var ilmAddFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "tags",
		Usage: "format '<key1>=<value1>&<key2>=<value2>&<key3>=<value3>', multiple values allowed for multiple key/value pairs",
	},
	cli.StringFlag{
		Name:   "expiry-date",
		Usage:  "format 'YYYY-MM-DD' the date of expiration",
		Hidden: true,
	},
	cli.StringFlag{
		Name:  "expiry-days",
		Usage: "the number of days to expiration",
	},
	cli.StringFlag{
		Name:   "transition-date",
		Usage:  "format 'YYYY-MM-DD' for the date to transition",
		Hidden: true,
	},
	cli.StringFlag{
		Name:  "transition-days",
		Usage: "the number of days to transition",
	},
	cli.StringFlag{
		Name:  "storage-class",
		Usage: "storage class for current version to transition into. MinIO supports any warm tier configured via `mc-admin-tier-add`",
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
		Usage: "storage class for noncurrent versions to transition into",
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

	opts, err := ilm.GetLifecycleOptions(cliCtx)
	fatalIf(err.Trace(args...), "Unable to generate new lifecycle rules for the input")

	lfcCfg, err = opts.ToConfig(lfcCfg)
	fatalIf(err.Trace(args...), "Unable to generate new lifecycle rules for the input")

	fatalIf(client.SetLifecycle(ctx, lfcCfg).Trace(urlStr), "Unable to add this lifecycle rule")

	printMsg(ilmAddMessage{
		Status: "success",
		Target: urlStr,
		ID:     opts.ID,
	})

	return nil
}
