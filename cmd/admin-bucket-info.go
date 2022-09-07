// Copyright (c) 2022 MinIO, Inc.
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
	"path/filepath"
	"sort"
	"strings"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminBucketInfoFlags = []cli.Flag{}

type adminBucketInfoMessage struct {
	Status    string                 `json:"status"`
	URL       string                 `json:"url"`
	UsageInfo madmin.BucketUsageInfo `json:"usage"`
	Props     BucketInfo             `json:"props"`
}

type histogramDef struct {
	start, end uint64
	text       string
}

var histogramTagsDesc = map[string]histogramDef{
	"LESS_THAN_1024_B":          {0, 1024, "less than 1024 bytes"},
	"BETWEEN_1024_B_AND_1_MB":   {1024, 1024 * 1024, "between 1024 bytes and 1 MB"},
	"BETWEEN_1_MB_AND_10_MB":    {1024 * 1024, 10 * 1024 * 1024, "between 1 MB and 10 MB"},
	"BETWEEN_10_MB_AND_64_MB":   {10 * 1024 * 1024, 64 * 1024 * 1024, "between 10 MB and 64 MB"},
	"BETWEEN_64_MB_AND_128_MB":  {64 * 1024 * 1024, 128 * 1024 * 1024, "between 64 MB and 128 MB"},
	"BETWEEN_128_MB_AND_512_MB": {128 * 1024 * 1024, 512 * 1024 * 1024, "between 128 MB and 512 MB"},
	"GREATER_THAN_512_MB":       {512 * 1024 * 1024, 0, "greater than 512 MB"},
}

// Return a sorted list of histograms
func sortHistogramTags() (orderedTags []string) {
	orderedTags = make([]string, 0, len(histogramTagsDesc))
	for tag := range histogramTagsDesc {
		orderedTags = append(orderedTags, tag)
	}
	sort.Slice(orderedTags, func(i, j int) bool {
		return histogramTagsDesc[orderedTags[i]].start < histogramTagsDesc[orderedTags[j]].start
	})
	return
}

func countDigits(num uint64) (count uint) {
	for num > 0 {
		num /= 10
		count++
	}
	return
}

func (bi adminBucketInfoMessage) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, console.Colorize("Title", "Usage:\n"))

	fmt.Fprintf(&b, "%16s: %s\n", "Total size", console.Colorize("Count", humanize.IBytes(bi.UsageInfo.Size)))
	fmt.Fprintf(&b, "%16s: %s\n", "Objects count", console.Colorize("Count", humanize.Comma(int64(bi.UsageInfo.ObjectsCount))))
	fmt.Fprintf(&b, "%16s: %s\n", "Versions count", console.Colorize("Count", humanize.Comma(int64(bi.UsageInfo.VersionsCount))))
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, console.Colorize("Title", "Properties:\n"))
	fmt.Fprintf(&b, prettyPrintBucketMetadata(bi.Props))

	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, console.Colorize("Title", "Object sizes histogram:\n"))

	var maxDigits uint
	for _, val := range bi.UsageInfo.ObjectSizesHistogram {
		if d := countDigits(val); d > maxDigits {
			maxDigits = d
		}
	}

	sortedTags := sortHistogramTags()
	for _, tagName := range sortedTags {
		val, ok := bi.UsageInfo.ObjectSizesHistogram[tagName]
		if ok {
			fmt.Fprintf(&b, "   %*d object(s) %s\n", maxDigits, val, histogramTagsDesc[tagName].text)
		}
	}

	return b.String()
}

func (bi adminBucketInfoMessage) JSON() string {
	jsonMessageBytes, e := json.MarshalIndent(bi, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(jsonMessageBytes)
}

var adminBucketInfoCmd = cli.Command{
	Name:         "info",
	Usage:        "display bucket information",
	Action:       mainAdminBucketInfo,
	OnUsageError: onUsageError,
	Before:       setGlobalsFromContext,
	Flags:        append(adminBucketInfoFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Display the usage data and configuration of a bucket on MinIO.
     {{.Prompt}} {{.HelpName}} myminio/mybucket
`,
}

// checkAdminBucketInfoSyntax - validate all the passed arguments
func checkAdminBucketInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, ctx.Command.Name, 1) // last argument is exit code
	}
}

// mainAdminBucketInfo is the handler for "mc admin bucket info" command.
func mainAdminBucketInfo(ctx *cli.Context) error {
	checkAdminBucketInfoSyntax(ctx)

	console.SetColor("Title", color.New(color.Bold, color.FgBlue))
	console.SetColor("Count", color.New(color.FgGreen))
	console.SetColor("Metadata", color.New(color.FgWhite))
	console.SetColor("Key", color.New(color.FgCyan))
	console.SetColor("Value", color.New(color.FgYellow))
	console.SetColor("Unset", color.New(color.FgRed))
	console.SetColor("Set", color.New(color.FgGreen))

	// Get the alias parameter from cli
	args := ctx.Args()
	aliasedURL := args.Get(0)

	// Create a new MinIO Admin Client
	adminClient, err := newAdminClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	s3Client, err := newClient(aliasedURL)
	fatalIf(err, "Unable to initialize admin connection.")

	aliasedURL = filepath.ToSlash(aliasedURL)
	splits := splitStr(aliasedURL, "/", 3)
	bucket := splits[1]

	duinfo, e := adminClient.DataUsageInfo(globalContext)
	fatalIf(probe.NewError(e).Trace(args...), "Unable to get data usage")

	bi, err := s3Client.GetBucketInfo(globalContext)
	fatalIf(err.Trace(args...), "Unable to get bucket properties")

	bu, ok := duinfo.BucketsUsage[bucket]
	if !ok {
		fatalIf(errDummy().Trace(args...), "Unable to get bucket usage info. Bucket usage is not ready yet.")
	}

	printMsg(adminBucketInfoMessage{
		Status:    "success",
		URL:       aliasedURL,
		UsageInfo: bu,
		Props:     bi,
	})

	return nil
}
