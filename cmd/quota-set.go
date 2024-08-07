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
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var quotaSetFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "size",
		Usage: "set a hard quota, disallowing writes after quota is reached",
	},
}

var quotaSetCmd = cli.Command{
	Name:         "set",
	Usage:        "set bucket quota",
	Action:       mainQuotaSet,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(quotaSetFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET [--size QUOTA]

QUOTA
  Quota accepts human-readable case-insensitive number.
  Suffixes such as "k", "m", "g" and "t" referring to the metric units KB,
  MB, GB and TB respectively. Adding an "i" to these prefixes, uses the IEC
  units, so that "gi" refers to "gibibyte" or "GiB". A "b" at the end is
  also accepted. Without suffixes the unit is bytes.

  The MinIO object scanner checks a bucket's quota each time it is scanned.
  If the scanner determines a bucket has met or exceeded its quota, MinIO
  rejects subsequent object write requests until the scanner determines the
  bucket no longer exceeds its quota.

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Set hard quota of 1gb for a bucket "mybucket" on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket --size 1GB
`,
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
			fmt.Sprintf("Successfully set bucket quota of %s on `%s`", humanize.IBytes(q.Quota), q.Bucket))
	case "clear":
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

// checkQuotaSetSyntax - validate all the passed arguments
func checkQuotaSetSyntax(ctx *cli.Context) {
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

// mainQuotaSet is the handler for "mc quota set" command.
func mainQuotaSet(ctx *cli.Context) error {
	checkQuotaSetSyntax(ctx)

	console.SetColor("QuotaMessage", color.New(color.FgGreen))
	console.SetColor("QuotaInfo", color.New(color.FgBlue))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	_, targetURL := url2Alias(args[0])
	if !ctx.IsSet("size") {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"--size flag needs to be set.")
	}
	qType := madmin.HardQuota
	quotaStr := ctx.String("size")
	quota, e := humanize.ParseBytes(quotaStr)
	fatalIf(probe.NewError(e).Trace(quotaStr), "Unable to parse quota")

	fatalIf(probe.NewError(client.SetBucketQuota(globalContext, targetURL, &madmin.BucketQuota{
		Quota: quota,
		Type:  qType,
	})).Trace(args...), "Unable to set bucket quota")

	printMsg(quotaMessage{
		op:        ctx.Command.Name,
		Bucket:    targetURL,
		Quota:     quota,
		QuotaType: string(qType),
		Status:    "success",
	})

	return nil
}
