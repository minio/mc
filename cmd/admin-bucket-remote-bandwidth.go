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
	"sort"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminBandwidthInfoCmdFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "unit",
		Value: "b",
		Usage: "[b|bi|B|Bi] Display bandwidth in bits (IEC [bi] or SI [b]) or bytes (IEC [Bi] or SI [B])",
	},
}

var adminBwInfoCmd = cli.Command{
	Name:         "bandwidth",
	Usage:        "Show bandwidth info for buckets on the MinIO server in bits or bytes per second. Ki,Bi,Mi,Gi represent IEC units.",
	Action:       mainAdminBwInfo,
	Before:       setGlobalsFromContext,
	OnUsageError: onUsageError,
	Flags:        append(globalFlags, adminBandwidthInfoCmdFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} FLAGS TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show the bandwidth usage for all the buckets in a MinIO server setup
     {{.Prompt}} {{.HelpName}} play/
  2. Show the bandwidth usage for the bucket 'source-bucket' in a MinIO server setup
     {{.Prompt}} {{.HelpName}} play/source-bucket
`,
}

func printTable(report madmin.Report, bits bool, iec bool) {
	bucketMaxLength := 63
	bucketColLength := 6
	var bucketKeys []string
	for bucket := range report.Report.BucketStats {
		if len(bucket) <= bucketMaxLength && len(bucket) > bucketColLength {
			bucketColLength = len(bucket)
		}
		bucketKeys = append(bucketKeys, bucket)
	}
	sort.Strings(bucketKeys)
	dspOrder := []col{colGreen} // Header
	for i := 0; i < len(report.Report.BucketStats); i++ {
		dspOrder = append(dspOrder, colGrey)
	}
	var printColors []*color.Color
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}

	cellText := make([][]string, len(report.Report.BucketStats)+1) // 1 for the header
	tbl := console.NewTable(printColors, []bool{false, false, false}, 0)
	bucketTitle := fmt.Sprintf("%-16v", "Bucket")
	cellText[0] = []string{
		bucketTitle,
		"Configured Max Bandwidth",
		"Current Bandwidth",
	}
	tbl.HeaderRowSeparator = true
	index := 1

	for _, bucket := range bucketKeys {
		values := report.Report.BucketStats[bucket]
		if len(bucket) > bucketMaxLength {
			bucket = bucket[:bucketMaxLength] + ".."
		}
		var mul uint64
		mul = 1
		if bits {
			mul = 8
		}
		limit := humanize.Bytes(uint64(values.LimitInBytesPerSecond) * mul)
		current := humanize.Bytes(uint64(values.CurrentBandwidthInBytesPerSecond) * mul)
		if iec {
			limit = humanize.IBytes(uint64(values.LimitInBytesPerSecond)*mul) + "/sec"
			current = humanize.IBytes(uint64(values.CurrentBandwidthInBytesPerSecond)*mul) + "/sec"
		}
		if bits {
			limit = strings.ToLower(limit) + "/sec"
			current = strings.ToLower(current) + "/sec"
		}
		if values.LimitInBytesPerSecond == 0 {
			limit = "N/A" // N/A means cluster bandwidth is not configured
		}
		cellText[index] = []string{
			bucket,
			limit,
			current,
		}
		index++
	}
	if len(report.Report.BucketStats) > 0 {
		err := tbl.DisplayTable(cellText)
		if err != nil {
			console.Error(err)
		}
	}
}
func checkAdminBwInfoSyntax(ctx *cli.Context) {
	u := ctx.String("unit")
	if u != "bi" &&
		u != "b" &&
		u != "Bi" &&
		u != "B" &&
		u != "" {
		cli.ShowCommandHelpAndExit(ctx, "bandwidth", globalErrorExitStatus)
	}
	if len(ctx.Args()) > 1 || len(ctx.Args()) == 0 {
		cli.ShowCommandHelpAndExit(ctx, "bandwidth", globalErrorExitStatus)
	}
}

func mainAdminBwInfo(ctx *cli.Context) {
	checkAdminBwInfoSyntax(ctx)
	aliasURL, bucket := getAliasAndBucket(ctx)
	client := getClient(aliasURL)
	reportCh := client.GetBucketBandwidth(globalContext, bucket)
	firstPrint := true
	bandwidthUnitsString := ctx.String("unit")
	for {
		select {
		case report := <-reportCh:
			if len(report.Report.BucketStats) == 0 {
				continue
			}
			if report.Err != nil {
				if strings.Contains(report.Err.Error(), "EOF") {
					continue
				}
				console.Error(report.Err)
			}
			printBandwidth(report, firstPrint, bandwidthUnitsString == "bi" || bandwidthUnitsString == "b",
				bandwidthUnitsString == "bi" || bandwidthUnitsString == "Bi")
			firstPrint = false
		case <-globalContext.Done():
			return
		}
	}
}

func printBandwidth(report madmin.Report, firstPrint bool, bits bool, iec bool) {
	rewindLines := len(report.Report.BucketStats) + 4
	if firstPrint {
		rewindLines = 0
	}
	if globalJSON {
		reportJSON, e := json.MarshalIndent(report, "", "  ")
		fatalIf(probe.NewError(e), "Unable to marshal to JSON")
		console.Println(string(reportJSON))
		time.Sleep(1 * time.Second)
		return
	}
	if len(report.Report.BucketStats) > 0 {
		console.RewindLines(rewindLines)
		// For the next iteration, rewind lines
		printTable(report, bits, iec)
	}
	time.Sleep(1 * time.Second)
}
