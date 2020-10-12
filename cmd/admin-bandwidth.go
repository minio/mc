/*
 * MinIO Client (C) 2020 MinIO, Inc.
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
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/bandwidth"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminBwFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "bits",
		Usage: "Default. Display current bandwidth usage in bits",
	},
	cli.BoolFlag{
		Name:  "bytes",
		Usage: "Display current bandwidth usage in bytes",
	},
	cli.BoolFlag{
		Name:  "iec-bits",
		Usage: "Display current bandwidth usage in IEC standard(bits)",
	},
	cli.BoolFlag{
		Name:  "iec-bytes",
		Usage: "Display current bandwidth usage in IEC standard(bits)",
	},
}

var adminBwInfoCmd = cli.Command{
	Name:   "bandwidth",
	Usage:  "Show bandwidth info for buckets on the MinIO server",
	Action: mainAdminBwInfo,
	Before: setGlobalsFromContext,
	Flags:  append(globalFlags, adminBwFlags...),
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

// enum types for bandwidth display
const (
	displayBWinBits = iota
	displayBWinBytes
	displayBWinIECBits
	displayBWinIECBytes
)

// Wrap single server "Info" message together with fields "Status" and "Error"
type bandwidthInfoPerBucket struct {
	Status string                       `json:"status"`
	Error  string                       `json:"error,omitempty"`
	Server string                       `json:"server,omitempty"`
	Bucket string                       `json:"bucket,omitempty"`
	Info   map[string]bandwidth.Details `json:"info,omitempty"`
}

func (b bandwidthInfoPerBucket) String() (msg string) {
	if b.Status == "error" {
		fatal(probe.NewError(errors.New(b.Error)), "Unable to get service status")
	}

	// Color palette initialization
	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	msg += fmt.Sprintf("%s  %s\n", console.Colorize("Info", dot), console.Colorize("PrintB", b.Server))
	for bucket, sample := range b.Info {
		limitStr := fmt.Sprintf("%d", sample.LimitInBytesPerSecond)
		curStr := fmt.Sprintf("%.4f", sample.CurrentBandwidthInBytesPerSecond)
		msg += fmt.Sprintf("   Bucket: %s\n", console.Colorize("Info", bucket))
		msg += fmt.Sprintf("      Limit  : %s\n", console.Colorize("Info", limitStr))
		msg += fmt.Sprintf("      Current Bandwidth     : %s\n", console.Colorize("Info", curStr))
	}
	return msg
}

func (b bandwidthInfoPerBucket) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(b, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(statusJSONBytes)
}

// getBandwidthDataBucket For the given server & the bucket in the server fetch a sample collection of bandwidth details
// Return for the samples obtained arrange them bucket - array of details, error if any.
func getBandwidthDataBucket(client *madmin.AdminClient, bucket string) (map[string]bandwidth.Details, error) {
	var buckets []string
	var bwRep bandwidth.Report
	var e error
	buckets = append(buckets, bucket)
	if bwRep, e = client.GetBucketBandwidth(globalContext, buckets...); e != nil {
		return nil, e
	}
	return bwRep.BucketStats, nil
}

func getBandwidthTableValues(client *madmin.AdminClient, targetURL string) map[string]bandwidth.Details {
	bwSampleCollection, err := getBandwidthDataBucket(client, targetURL)
	fatalIf(probe.NewError(err), "Unable to fetch bandwidth data for "+targetURL)
	return bwSampleCollection
}

func removeStaleBWEntries(server string, bwDisplaySample map[string]bandwidth.Details) map[string]bandwidth.Details {
	var rmBuckets []string
	for bucket := range bwDisplaySample {
		urlprefix := server + "/" + bucket
		if !isURLPrefixExists(urlprefix, false) {
			rmBuckets = append(rmBuckets, bucket)
		}
	}
	for _, bucket := range rmBuckets {
		delete(bwDisplaySample, bucket)
	}
	return bwDisplaySample
}

// printBandwidthTable - Prints the table.
func printBandwidthTable(bwDisplaySample map[string]bandwidth.Details, bwDisplayUnit int) {
	bucketMinLength := 6
	bucketMaxLength := 25
	var bucketKeys []string
	for bucket := range bwDisplaySample {
		if len(bucket) > bucketMinLength && len(bucket) <= bucketMaxLength {
			bucketMinLength = len(bucket)
		} else if len(bucket) >= bucketMaxLength {
			bucketMinLength = bucketMaxLength
		}
		bucketKeys = append(bucketKeys, bucket)
	}
	// Buckets will be displayed in the sorted order.
	sort.Strings(bucketKeys)
	// Color arrangement for the table.
	dspOrder := []col{colGreen} // Header
	for i := 0; i < len(bwDisplaySample); i++ {
		dspOrder = append(dspOrder, colGrey)
	}
	printColors := []*color.Color{}
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}
	// Table cell initialization of the header for the table.
	cellText := make([][]string, len(bwDisplaySample)+1) // 1 for the header
	tbl := console.NewTable(printColors, []bool{false, false, false}, 10)
	bucketTitle := fmt.Sprintf(fmt.Sprintf("%%-%ds", (bucketMinLength/2)), fmt.Sprintf(fmt.Sprintf("%%%ds", (bucketMinLength/2)+2), "Bucket"))
	var rateTitle string
	switch bwDisplayUnit {
	case displayBWinBits:
		rateTitle = "(bits per second)"
	case displayBWinBytes:
		rateTitle = "(bytes per second)"
	case displayBWinIECBits:
		rateTitle = "(bits per second,IEC)"
	case displayBWinIECBytes:
		rateTitle = "(bytes per second,IEC)"
	}
	cellText[0] = []string{
		bucketTitle,
		"Limit " + rateTitle,
		"Current Bandwidth " + rateTitle,
	}

	// Header separator between header & rows.
	tbl.HeaderRowSeparator = true
	index := 1

	// Initialize row/column cell values
	for _, bucket := range bucketKeys {
		values := bwDisplaySample[bucket]
		var limitStr, curBwStr string
		switch bwDisplayUnit {
		case displayBWinBits:
			limitStr = humanize.Bytes(uint64(values.LimitInBytesPerSecond * 8))
			curBwStr = humanize.Bytes(uint64(values.CurrentBandwidthInBytesPerSecond * 8))
		case displayBWinBytes:
			limitStr = humanize.Bytes(uint64(values.LimitInBytesPerSecond))
			curBwStr = humanize.Bytes(uint64(values.CurrentBandwidthInBytesPerSecond))
		case displayBWinIECBits:
			limitStr = humanize.IBytes(uint64(values.LimitInBytesPerSecond * 8))
			curBwStr = humanize.IBytes(uint64(values.CurrentBandwidthInBytesPerSecond * 8))
		case displayBWinIECBytes:
			limitStr = humanize.IBytes(uint64(values.LimitInBytesPerSecond))
			curBwStr = humanize.IBytes(uint64(values.CurrentBandwidthInBytesPerSecond))
		}
		limitStr = strings.ReplaceAll(limitStr, "B", "")
		curBwStr = strings.ReplaceAll(curBwStr, "B", "")
		limitStr = fmt.Sprintf("%-12s", fmt.Sprintf("%12s", limitStr))
		curBwStr = fmt.Sprintf("%-16s", fmt.Sprintf("%20s", curBwStr))
		if len(bucket) > bucketMinLength {
			bucket = bucket[:(bucketMinLength-2)] + ".."
		}
		cellText[index] = []string{
			bucket,
			limitStr,
			curBwStr,
		}
		index++
	}

	// Display table if data length is more than 0.
	if len(bwDisplaySample) > 0 {
		tbl.DisplayTable(cellText)
	}
}

func showBandwidthJSON(server string, targetURL string, bwDisp map[string]bandwidth.Details) {
	var bwInfo bandwidthInfoPerBucket
	bwInfo.Info = bwDisp
	bwInfo.Server = server
	bwInfo.Bucket = targetURL
	printMsg(bandwidthInfoPerBucket(bwInfo))
	console.Println()
}

func showBandwidthValues(server string, bwDisplayUnit int, rewindLines int, bwDisp map[string]bandwidth.Details) {
	if len(bwDisp) > 0 {
		console.RewindLines(rewindLines)
		printBandwidthTable(bwDisp, bwDisplayUnit)
	}
}

func checkAdminBwInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) > 1 || len(ctx.Args()) == 0 {
		cli.ShowCommandHelpAndExit(ctx, "bandwidth", globalErrorExitStatus)
	}
}

func mainAdminBwInfo(ctx *cli.Context) error {
	checkAdminBwInfoSyntax(ctx)
	args := ctx.Args()
	server := args.Get(0)
	_, targetURL := url2Alias(args[0])
	if !isURLPrefixExists(server, false) {
		e := errors.New("target does not exist")
		fatalIf(probe.NewError(e), "Unable to obtain bandwidth data from "+server)
	} else if len(targetURL) == 0 {
		server = cleanAlias(server)
	}
	firstPrint := true
	rewindLines := 1
	bwDisplayUnit := displayBWinBits
	switch {
	case ctx.Bool("bytes"):
		bwDisplayUnit = displayBWinBytes
	case ctx.Bool("iec-bits"):
		bwDisplayUnit = displayBWinIECBits
	case ctx.Bool("iec-bytes"):
		bwDisplayUnit = displayBWinIECBytes
	}
	client, err := newAdminClient(server)
	fatalIf(err.Trace(), "Unable to initialize admin connection")

	if globalJSON {
		bwDispVal := getBandwidthTableValues(client, targetURL)
		showBandwidthJSON(server, targetURL, bwDispVal)
		return nil
	}

	console.PrintC(console.Colorize("BlinkLoad", "Fetching bandwidth data...\n"))
	for {
		// In for loop fetch bandwidth data( for all buckets or just one bucket).
		select {
		case <-globalContext.Done():
			// Exit for Ctrl-c
			os.Exit(0)
		default:
			bwDispVal := getBandwidthTableValues(client, targetURL)
			// show table (or JSON), sleep for 1 second before getting sample values again for display
			removeStaleBWEntries(server, bwDispVal)
			showBandwidthValues(server, bwDisplayUnit, rewindLines, bwDispVal)
			firstPrint = firstPrint && (len(bwDispVal) == 0)
			// Calculate # of rewind lines for next iteration
			if len(bwDispVal) > 0 {
				rewindLines = len(bwDispVal) + 4
			} else if !firstPrint {
				rewindLines = 0
			}
			time.Sleep(1 * time.Second)
		}
	}
}
