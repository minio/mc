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
	"fmt"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminQuotaFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "fifo",
		Usage: "set fifo quota, allowing automatic deletion of older content",
	},
	cli.StringFlag{
		Name:  "hard",
		Usage: "set a hard quota, disallowing writes after quota is reached",
	},
	cli.BoolFlag{
		Name:  "clear",
		Usage: "clears bucket quota configured for bucket",
	},
}

// quotaMessage container for content message structure
type quotaMessage struct {
	op        string
	Status    string `json:"status"`
	Bucket    string `json:"bucket"`
	Quota     uint64 `json:"quota,omitempty"`
	QuotaType string `json:"type,omitempty"`
}

func (q quotaMessage) String() string {
	switch q.op {
	case "set":
		return console.Colorize("QuotaMessage",
			fmt.Sprintf("Successfully set bucket quota of %s with %s type on `%s`", humanize.IBytes(q.Quota), q.QuotaType, q.Bucket))
	case "unset":
		return console.Colorize("QuotaMessage",
			fmt.Sprintf("Successfully cleared bucket quota configured on `%s`", q.Bucket))
	default:
		return console.Colorize("QuotaInfo",
			fmt.Sprintf("Bucket `%s` has %s quota of %s", q.Bucket, q.QuotaType, humanize.IBytes(q.Quota)))
	}
}

func (q quotaMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(q, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

var adminBucketQuotaCmd = cli.Command{
	Name:         "quota",
	Usage:        "manage bucket quota",
	Action:       mainAdminBucketQuota,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminQuotaFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [--fifo QUOTA | --hard QUOTA | --clear]

QUOTA
  quota accepts human-readable case-insensitive number
  suffixes such as "k", "m", "g" and "t" referring to the metric units KB,
  MB, GB and TB respectively. Adding an "i" to these prefixes, uses the IEC
  units, so that "gi" refers to "gibibyte" or "GiB". A "b" at the end is
  also accepted. Without suffixes the unit is bytes.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Display bucket quota configured for "mybucket" on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket

  2. Set FIFO quota for a bucket "mybucket" on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --fifo 10GB

  3. Set hard quota of 1gb for a bucket "mybucket" on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --hard 1GB

  4. Clear bucket quota configured for bucket "mybucket" on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --clear
`,
}

// checkAdminBucketQuotaSyntax - validate all the passed arguments
func checkAdminBucketQuotaSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}

	if ctx.IsSet("hard") && ctx.IsSet("fifo") {
		fatalIf(errInvalidArgument(), "Only one of --hard or --fifo flags can be set")
	}
	if (ctx.IsSet("hard") || ctx.IsSet("fifo")) && len(ctx.Args()) == 0 {
		fatalIf(errInvalidArgument().Trace(ctx.Args()...), "please specify bucket and quota")
	}
	if ctx.IsSet("clear") && len(ctx.Args()) == 0 {
		fatalIf(errInvalidArgument().Trace(ctx.Args()...), "clear flag must be passed with target alone")
	}
}

// mainAdminBucketQuota is the handler for "mc admin bucket quota" command.
func mainAdminBucketQuota(ctx *cli.Context) error {
	checkAdminBucketQuotaSyntax(ctx)

	console.SetColor("QuotaMessage", color.New(color.FgGreen))
	console.SetColor("QuotaInfo", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")
	quotaStr := ctx.String("fifo")
	if ctx.IsSet("hard") {
		quotaStr = ctx.String("hard")
	}
	_, targetURL := url2Alias(args[0])
	if ctx.IsSet("fifo") || ctx.IsSet("hard") && len(args) == 1 {
		qType := madmin.FIFOQuota
		if ctx.IsSet("hard") {
			qType = madmin.HardQuota
		}
		quota, e := humanize.ParseBytes(quotaStr)
		fatalIf(probe.NewError(e).Trace(quotaStr), "Unable to parse quota")
		if e = client.SetBucketQuota(globalContext, targetURL, &madmin.BucketQuota{Quota: quota, Type: qType}); e != nil {
			fatalIf(probe.NewError(e).Trace(args...), "Unable to set bucket quota")
		}
		printMsg(quotaMessage{
			op:        "set",
			Bucket:    targetURL,
			Quota:     quota,
			QuotaType: string(qType),
			Status:    "success",
		})
	} else if ctx.Bool("clear") && len(args) == 1 {
		if err := client.SetBucketQuota(globalContext, targetURL, &madmin.BucketQuota{}); err != nil {
			fatalIf(probe.NewError(err).Trace(args...), "Unable to clear bucket quota config")
		}
		printMsg(quotaMessage{
			op:     "unset",
			Bucket: targetURL,
			Status: "success",
		})

	} else {
		qCfg, e := client.GetBucketQuota(globalContext, targetURL)
		fatalIf(probe.NewError(e).Trace(args...), "Unable to get bucket quota")
		printMsg(quotaMessage{
			op:        "get",
			Bucket:    targetURL,
			Quota:     qCfg.Quota,
			QuotaType: string(qCfg.Type),
			Status:    "success",
		})
	}

	return nil
}
