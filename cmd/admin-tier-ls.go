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
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	madmin "github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminTierListCmd = cli.Command{
	Name:         "ls",
	Usage:        "lists configured remote tier targets",
	Action:       mainAdminTierList,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        globalFlags,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} ALIAS

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. List remote tier targets configured on 'myminio':
     {{.Prompt}} {{.HelpName}} myminio
`,
}

// checkAdminTierListSyntax - validate all the passed arguments
func checkAdminTierListSyntax(ctx *cli.Context) {
	argsNr := len(ctx.Args())
	if argsNr < 1 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
	if argsNr > 1 {
		fatalIf(errInvalidArgument().Trace(ctx.Args().Tail()...),
			"Incorrect number of arguments for tier-ls subcommand.")
	}
}

type tierLSRowHdr int

const (
	tierLSNameHdr tierLSRowHdr = iota
	tierLSTypeHdr
	tierLSEndpointHdr
	tierLSBucketHdr
	tierLSPrefixHdr
	tierLSRegionHdr
	tierLSStorageClassHdr
)

var tierLSRowNames = []string{
	"Name",
	"Type",
	"Endpoint",
	"Bucket",
	"Prefix",
	"Region",
	"Storage-Class",
}

var tierLSColorScheme = []*color.Color{
	color.New(color.FgYellow),
	color.New(color.FgCyan),
	color.New(color.FgGreen),
	color.New(color.FgHiWhite),
	color.New(color.FgHiWhite),
	color.New(color.FgHiWhite),
	color.New(color.FgCyan),
}

func storageClass(t *madmin.TierConfig) string {
	switch t.Type {
	case madmin.S3:
		return t.S3.StorageClass
	case madmin.Azure:
		return t.Azure.StorageClass
	case madmin.GCS:
		return t.GCS.StorageClass
	default:
		return ""
	}
}

type tierLS []*madmin.TierConfig

func (t tierLS) NumRows() int {
	return len(([]*madmin.TierConfig)(t))
}

func (t tierLS) NumCols() int {
	return len(tierLSRowNames)
}

func (t tierLS) EmptyMessage() string {
	return "No remote tier has been configured"
}

func (t tierLS) ToRow(i int, ls []int) []string {
	row := make([]string, len(tierLSRowNames))
	if i == -1 {
		copy(row, tierLSRowNames)
	} else {
		tc := t[i]
		row[tierLSNameHdr] = tc.Name
		row[tierLSTypeHdr] = tc.Type.String()
		row[tierLSEndpointHdr] = tc.Endpoint()
		row[tierLSBucketHdr] = tc.Bucket()
		row[tierLSPrefixHdr] = tc.Prefix()
		row[tierLSRegionHdr] = tc.Region()
		row[tierLSStorageClassHdr] = storageClass(tc)

	}

	// update ls to accommodate this row's values
	for i := range tierLSRowNames {
		if ls[i] < len(row[i]) {
			ls[i] = len(row[i])
		}
	}
	return row
}

var _ tabulator = (tierLS)(nil)

type tierListMessage struct {
	Status  string               `json:"status"`
	Context *cli.Context         `json:"-"`
	Tiers   []*madmin.TierConfig `json:"tiers"`
}

// String method returns a tabular listing of remote tier configurations.
func (msg *tierListMessage) String() string {
	return toTable(tierLS(msg.Tiers))
}

// JSON method returns JSON encoding of msg.
func (msg *tierListMessage) JSON() string {
	b, _ := json.Marshal(msg)
	return string(b)
}

func mainAdminTierList(ctx *cli.Context) error {
	checkAdminTierListSyntax(ctx)

	for i, color := range tierLSColorScheme {
		console.SetColor(tierLSRowNames[i], color)
	}

	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	client, cerr := newAdminClient(aliasedURL)
	fatalIf(cerr, "Unable to initialize admin connection.")

	tiers, err := client.ListTiers(globalContext)
	if err != nil {
		fatalIf(probe.NewError(err).Trace(args...), "Unable to list configured remote tier targets")
	}

	printMsg(&tierListMessage{
		Status:  "success",
		Context: ctx,
		Tiers:   tiers,
	})
	return nil
}
