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
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/bandwidth"
	"github.com/minio/minio/pkg/console"
)

var adminBwInfoCmd = cli.Command{
	Name:   "bandwidth",
	Usage:  "Show bandwidth info for buckets on the MinIO server",
	Action: mainAdminBwInfo,
	Before: setGlobalsFromContext,
	Flags:  globalFlags,
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

// BandwidthDisplayVal - structure that holds info for the user
// which only shows the data calculated from sampled output
type BandwidthDisplayVal struct {
	LimitInBytesPerSecond            int64   `json:"limit"`
	CurrentBandwidthInBytesPerSecond float64 `json:"currentBandwidth"`
}

// BandwidthSampleResult - contains Bucket - BandwidthDisplayVal map
// Sampled from value of all servers
type BandwidthSampleResult struct {
	SampleResult map[string]BandwidthDisplayVal
}

// Wrap single server "Info" message together with fields "Status" and "Error"
type bandwidthInfoPerBucket struct {
	Status string                `json:"status"`
	Error  string                `json:"error,omitempty"`
	Server string                `json:"server,omitempty"`
	Bucket string                `json:"bucket,omitempty"`
	Info   BandwidthSampleResult `json:"info,omitempty"`
}

func (b bandwidthInfoPerBucket) String() (msg string) {
	if b.Status == "error" {
		fatal(probe.NewError(errors.New(b.Error)), "Unable to get service status")
	}
	// Color palette initialization
	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	msg += fmt.Sprintf("%s  %s\n", console.Colorize("Info", dot), console.Colorize("PrintB", b.Server))
	for bucket, sample := range b.Info.SampleResult {
		avgAg := fmt.Sprintf("%d", sample.LimitInBytesPerSecond)
		avgMv := fmt.Sprintf("%.4f", sample.CurrentBandwidthInBytesPerSecond)
		msg += fmt.Sprintf("   Bucket: %s\n", console.Colorize("Info", bucket))
		msg += fmt.Sprintf("      Limit  : %s\n", console.Colorize("Info", avgAg))
		msg += fmt.Sprintf("      Current Bandwidth     : %s\n", console.Colorize("Info", avgMv))
	}
	return msg
}

func (b bandwidthInfoPerBucket) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(b, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

func getSampleBucketCollection(server string, bucket string, sampleCount int) (bwSampleCollection map[string][]bandwidth.Details, err error) {
	client, pErr := newAdminClient(server)
	bwSampleCollection = make(map[string][]bandwidth.Details)
	var buckets []string
	var bwRep bandwidth.Report
	buckets = append(buckets, bucket)
	for index := 0; index < sampleCount; index++ {
		fatalIf(pErr, "Unable to initialize admin connection with "+server)
		if bwRep, err = client.GetBucketBandwidth(globalContext, buckets...); err != nil {
			return nil, err
		}
		for bucket, elem := range bwRep.BucketStats {
			if details, ok := bwSampleCollection[bucket]; !ok {
				var detailArr []bandwidth.Details
				detailArr = append(detailArr, elem)
				bwSampleCollection[bucket] = detailArr
			} else {
				details = append(details, elem)
				bwSampleCollection[bucket] = details
			}
		}
	}
	return bwSampleCollection, err
}

func fetchBandwidthDataBucket(server string, bucket string) (map[string][]bandwidth.Details, error) {
	sampleCount := 1
	return getSampleBucketCollection(server, bucket, sampleCount)
}

func buildSampleTableDataBucket(bwSampleCollection map[string][]bandwidth.Details) map[string]BandwidthDisplayVal {
	bwDisplayCollection := make(map[string]BandwidthDisplayVal)
	for bucket, sampleArr := range bwSampleCollection {
		for _, sample := range sampleArr {
			bwDisplayCollection[bucket] = BandwidthDisplayVal{
				LimitInBytesPerSecond:            sample.LimitInBytesPerSecond,
				CurrentBandwidthInBytesPerSecond: sample.CurrentBandwidthInBytesPerSecond,
			}
		}
	}
	return bwDisplayCollection
}

func printTable(bwDisplaySample map[string]BandwidthDisplayVal) {
	bucketMaxLength := 16
	bucketColLength := 6
	var bucketKeys []string
	for bucket := range bwDisplaySample {
		if len(bucket) <= bucketMaxLength && len(bucket) > bucketColLength {
			bucketColLength = len(bucket)
		}
		bucketKeys = append(bucketKeys, bucket)
	}
	sort.Strings(bucketKeys)
	dspOrder := []col{colGreen} // Header
	for i := 0; i < len(bwDisplaySample); i++ {
		dspOrder = append(dspOrder, colGrey)
	}
	printColors := []*color.Color{}
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}

	cellText := make([][]string, len(bwDisplaySample)+1) // 1 for the header
	tbl := console.NewTable(printColors, []bool{false, false, false}, 5)
	bucketTitle := fmt.Sprintf("%-16v", "Bucket")
	cellText[0] = []string{
		bucketTitle,
		"Limit      ",
		"Current Bandwidth",
	}
	tbl.HeaderRowSeparator = true
	index := 1

	for _, bucket := range bucketKeys {
		values := bwDisplaySample[bucket]
		if len(bucket) > bucketMaxLength {
			bucket = bucket[:12] + ".."
		}
		cellText[index] = []string{
			bucket,
			humanize.IBytes(uint64(values.LimitInBytesPerSecond)),
			humanize.IBytes(uint64(values.CurrentBandwidthInBytesPerSecond)),
		}
		index++
	}
	if len(bwDisplaySample) > 0 {
		tbl.DisplayTable(cellText)
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
	console.PrintC(console.Colorize("BlinkLoad", "Fetching bandwidth data...\n"))
	_, targetURL := url2Alias(args[0])
	rewindLines := 1
	firstPrint := true
	for {
		select {
		case <-globalContext.Done():
			os.Exit(0)
		default:
			bwSampleCollection, err := fetchBandwidthDataBucket(server, targetURL)
			fatalIf(probe.NewError(err), "Unable to fetch bandwidth data for "+args[0])
			bwBucketDispVal := buildSampleTableDataBucket(bwSampleCollection)
			if globalJSON {
				var bwInfo bandwidthInfoPerBucket
				bwInfo.Info.SampleResult = bwBucketDispVal
				bwInfo.Server = server
				bwInfo.Bucket = targetURL
				printMsg(bandwidthInfoPerBucket(bwInfo))
				console.Println()
			} else {
				if len(bwBucketDispVal) > 0 {
					console.RewindLines(rewindLines)
					// For the next iteration, rewind lines
					rewindLines = len(bwBucketDispVal) + 4
					printTable(bwBucketDispVal)
					firstPrint = false
				} else {
					rewindLines = 0
					if firstPrint {
						rewindLines = 1
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	}
}
