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

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/bucket/bandwidth"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminBwInfoCmd = cli.Command{
	Name:   "bandwidth",
	Usage:  "Show bandwidth info for buckets stored on the MinIO server",
	Action: mainAdminBwInfo,
	Before: setGlobalsFromContext,
	Flags:  append(adminBwFlags, globalFlags...),
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
  1. Show the bandwidth usage for the MinIO server
     {{.Prompt}} {{.HelpName}} --buckets bucket1 --buckets bucket2 play/
  2. Show the bandwidth usage for a distributed MinIO servers setup
     {{.Prompt}} {{.HelpName}} --buckets bucket1 --buckets bucket2 --servers minio01/ --servers minio02/ --servers minio03/ --servers minio04/
`,
}

var adminBwFlags = []cli.Flag{
	cli.StringSliceFlag{
		Name:  "buckets",
		Usage: "format '--buckets <bucket>', multiple values allowed for multiple buckets",
	},
	cli.StringSliceFlag{
		Name:  "servers",
		Usage: "format '--servers <server>', multiple values allowed for multiple servers for a distributed setup",
	},
}

// Wrap single server "Info" message together with fields "Status" and "Error"
type bandwidthSingleStruct struct {
	Status  string                       `json:"status"`
	Error   string                       `json:"error,omitempty"`
	Server  string                       `json:"server,omitempty"`
	Buckets []string                     `json:"buckets,omitempty"`
	Info    madmin.BandwidthSampleResult `json:"info,omitempty"`
}

func (b bandwidthSingleStruct) String() (msg string) {
	if b.Status == "error" {
		fatal(probe.NewError(errors.New(b.Error)), "Unable to get service status")
	}
	// Color palette initialization
	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	console.SetColor("InfoHeader", color.New(color.FgRed, color.Bold))
	console.SetColor("InfoFail", color.New(color.FgRed, color.Bold))
	console.SetColor("BlinkLoad", color.New(color.BlinkSlow, color.FgCyan))
	msg += fmt.Sprintf("%s  %s\n", console.Colorize("Info", dot), console.Colorize("PrintB", b.Server))
	for _, bucket := range b.Buckets {
		sumAg := fmt.Sprintf("%.4f", b.Info.SampleResult[bucket].SumAggregateBandwidth)
		sumMv := fmt.Sprintf("%.4f", b.Info.SampleResult[bucket].SumMovingBandwidth)
		avgAg := fmt.Sprintf("%.4f", b.Info.SampleResult[bucket].AvgAggregateBandwidth)
		avgMv := fmt.Sprintf("%.4f", b.Info.SampleResult[bucket].AvgMovingBandwidth)
		msg += fmt.Sprintf("   Bucket: %s\n", console.Colorize("Info", bucket))
		msg += fmt.Sprintf("      Sum Aggregate Bandwidth  : %s\n", console.Colorize("Info", sumAg))
		msg += fmt.Sprintf("      Sum Moving Bandwidth     : %s\n", console.Colorize("Info", sumMv))
		msg += fmt.Sprintf("      Avg Aggregate Bandwidth  : %s\n", console.Colorize("Info", avgAg))
		msg += fmt.Sprintf("      Avg Moving Bandwidth     : %s\n", console.Colorize("Info", avgMv))
	}
	return msg
}

func (b bandwidthSingleStruct) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(b, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// Wrap distributed server "Info" message
type bandwidthDistributedStruct struct {
	Info map[string]madmin.BandwidthSampleVal
}

func (b bandwidthDistributedStruct) String() (msg string) {
	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	for bucket, sampleVal := range b.Info {
		sumAg := fmt.Sprintf("%.4f", sampleVal.SumAggregateBandwidth)
		sumMv := fmt.Sprintf("%.4f", sampleVal.SumMovingBandwidth)
		avgAg := fmt.Sprintf("%.4f", sampleVal.AvgAggregateBandwidth)
		avgMv := fmt.Sprintf("%.4f", sampleVal.AvgMovingBandwidth)
		msg += fmt.Sprintf("   Bucket: %s\n", console.Colorize("Info", bucket))
		msg += fmt.Sprintf("      Sum Aggregate Bandwidth   : %s\n", console.Colorize("Info", sumAg))
		msg += fmt.Sprintf("      Sum Moving Bandwidth      : %s\n", console.Colorize("Info", sumMv))
		msg += fmt.Sprintf("      Avg Aggregate Bandwidth   : %s\n", console.Colorize("Info", avgAg))
		msg += fmt.Sprintf("      Avg Moving Bandwidth      : %s\n", console.Colorize("Info", avgMv))
	}
	return msg
}

func (b bandwidthDistributedStruct) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(b, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// Returns a collection fetched for all the buckets
// from each of the server
func fetchInitialBandwidthData(servers []string, buckets []string) map[string][]bandwidth.Details {
	// Building Server to Bandwidth data map.
	bwDistributedCollection := make(map[string][]bandwidth.Report)
	bwDistributedSample := make(map[string][]bandwidth.Details)
	for _, server := range servers {
		client, err := newAdminClient(server)
		fatalIf(err, "Unable to initialize admin connection with "+server)
		bwSampleResult, e := client.SingleSampleBandwidthInfo(globalContext, buckets)
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to get data from server "+server)
		}
		bwDistributedCollection[server] = bwSampleResult
		for _, sample := range bwSampleResult {
			for bucket, value := range sample.BucketStats {
				elements, ok := bwDistributedSample[bucket]
				if !ok {
					var bwDetails []bandwidth.Details
					bwDistributedSample[bucket] = append(bwDetails, value)
				} else {
					bwDistributedSample[bucket] = append(elements, value)
				}
			}
		}
	}
	return bwDistributedSample
}

func fetchBandwidthDataBucket(servers []string, buckets []string) (map[string][]bandwidth.Report, map[string][]bandwidth.Details) {
	// Building Server to Bandwidth data map.
	bwDistributedCollection := make(map[string][]bandwidth.Report)
	bwDistributedSample := make(map[string][]bandwidth.Details)
	for _, server := range servers {
		client, err := newAdminClient(server)
		fatalIf(err, "Unable to initialize admin connection with "+server)
		bwSampleResult, e := client.AllSampleBandwidthInfo(globalContext, buckets)
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to get data from server "+server)
		}
		bwDistributedCollection[server] = bwSampleResult
		for _, sample := range bwSampleResult {
			for bucket, value := range sample.BucketStats {
				elements, ok := bwDistributedSample[bucket]
				if !ok {
					var bwDetails []bandwidth.Details
					bwDistributedSample[bucket] = append(bwDetails, value)
				} else {
					bwDistributedSample[bucket] = append(elements, value)
				}
			}
		}
	}
	return bwDistributedCollection, bwDistributedSample
}

// Returns a collection fetched for all the buckets from each of the server
// map[string][]bandwidth.Report : map server - Slice of report data.
// map[string][]BandwidthSampleVal : map of bucket - Slice of Sample Values.
func fetchBandwidthData(servers []string, buckets []string) (map[string][]bandwidth.Report, map[string]madmin.BandwidthSampleVal) {
	// Building Server to Bandwidth data map.
	bwDistributedCollection := make(map[string][]bandwidth.Report)
	bwDistributedSample := make(map[string]madmin.BandwidthSampleVal)
	for _, server := range servers {
		client, err := newAdminClient(server)
		fatalIf(err, "Unable to initialize admin connection with "+server)
		bwSampleResult, e := client.AllSampleBandwidthInfo(globalContext, buckets)
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to get data from server "+server)
		}
		bwDistributedCollection[server] = bwSampleResult
		for _, sample := range bwSampleResult {
			for bucket, value := range sample.BucketStats {
				elem, ok := bwDistributedSample[bucket]
				if !ok {
					bwDistributedSample[bucket] = madmin.BandwidthSampleVal{
						SumAggregateBandwidth: value.AggregateBandwidth,
						SumMovingBandwidth:    value.MovingBandwidth,
						AvgAggregateBandwidth: value.AggregateBandwidth,
						AvgMovingBandwidth:    value.MovingBandwidth,
						SampleSetCount:        1,
					}
				} else {
					elem.SampleSetCount++
					elem.SumAggregateBandwidth += value.AggregateBandwidth
					elem.SumMovingBandwidth += value.MovingBandwidth
					elem.AvgAggregateBandwidth = float64(elem.SumAggregateBandwidth) / float64(elem.SampleSetCount)
					elem.AvgMovingBandwidth = float64(elem.SumMovingBandwidth) / float64(elem.SampleSetCount)
					bwDistributedSample[bucket] = elem
				}
			}
		}
	}
	return bwDistributedCollection, bwDistributedSample
}

// buildInitialTableValues - Input is bandwidth.Report for various servers, arranged per server.
// Output - create a map of buckets to Sample data from all those various servers
func buildInitialTableValues(bwDistributedCollection map[string][]bandwidth.Report) map[string]madmin.BandwidthSampleVal {
	// Buckets to Calculated data in this structure
	bwDistributedSample := make(map[string]madmin.BandwidthSampleVal)
	for _, result := range bwDistributedCollection {
		for _, bwMsg := range result {
			for bucket, value := range bwMsg.BucketStats {
				elem, ok := bwDistributedSample[bucket]
				if !ok {
					bwDistributedSample[bucket] = madmin.BandwidthSampleVal{
						SumAggregateBandwidth: value.AggregateBandwidth,
						SumMovingBandwidth:    value.MovingBandwidth,
						AvgAggregateBandwidth: value.AggregateBandwidth,
						AvgMovingBandwidth:    value.MovingBandwidth,
						SampleSetCount:        1,
					}
				} else {
					elem.SampleSetCount++
					elem.SumAggregateBandwidth += value.AggregateBandwidth
					elem.SumMovingBandwidth += value.MovingBandwidth
					elem.AvgAggregateBandwidth = float64(elem.SumAggregateBandwidth) / float64(elem.SampleSetCount)
					elem.AvgMovingBandwidth = float64(elem.SumMovingBandwidth) / float64(elem.SampleSetCount)
					bwDistributedSample[bucket] = elem
				}
			}
		}
	}
	return bwDistributedSample
}

func buildSampleTableDataBucket(bwDistributedCollection map[string][]bandwidth.Details) map[string]madmin.BandwidthSampleVal {
	bwDistributedSample := make(map[string]madmin.BandwidthSampleVal)
	for bucket, detailsArr := range bwDistributedCollection {
		for _, statsValue := range detailsArr {
			elem, ok := bwDistributedSample[bucket]
			if !ok {
				bwDistributedSample[bucket] = madmin.BandwidthSampleVal{
					SumAggregateBandwidth: statsValue.AggregateBandwidth,
					SumMovingBandwidth:    statsValue.MovingBandwidth,
					AvgAggregateBandwidth: statsValue.AggregateBandwidth,
					AvgMovingBandwidth:    statsValue.MovingBandwidth,
					SampleSetCount:        1,
				}
			} else {
				elem.SampleSetCount++
				elem.SumAggregateBandwidth += statsValue.AggregateBandwidth
				elem.SumMovingBandwidth += statsValue.MovingBandwidth
				elem.AvgAggregateBandwidth = float64(elem.SumAggregateBandwidth) / float64(elem.SampleSetCount)
				elem.AvgMovingBandwidth = float64(elem.SumMovingBandwidth) / float64(elem.SampleSetCount)
				bwDistributedSample[bucket] = elem
			}
		}
	}
	return bwDistributedSample
}
func printTable(bwDistributedSample map[string]madmin.BandwidthSampleVal) {
	dspOrder := []col{colRed, colGrey}
	printColors := []*color.Color{}
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}
	for bucket, values := range bwDistributedSample {
		t := console.NewTable(printColors, []bool{false, false, false, false, false}, 1)
		cellText := make([][]string, 2)
		cellText[0] = []string{
			fmt.Sprintf("Bucket"),
			fmt.Sprintf("Sum Aggregate Bandwidth"),
			fmt.Sprintf("Sum Moving Bandwidth"),
			fmt.Sprintf("Avg Aggregate Bandwidth"),
			fmt.Sprintf("Avg Moving Bandwidth"),
		}
		cellText[1] = []string{
			fmt.Sprintf(console.Colorize("InfoHeader", bucket)),
			fmt.Sprintf("%.4f", values.SumAggregateBandwidth),
			fmt.Sprintf("%.4f", values.SumMovingBandwidth),
			fmt.Sprintf("%.4f", values.AvgAggregateBandwidth),
			fmt.Sprintf("%.4f", values.AvgMovingBandwidth),
		}
		t.DisplayTable(cellText)
	}
}

func checkAdminBwInfoSyntax(ctx *cli.Context) {
	if len(ctx.Args()) > 1 {
		cli.ShowCommandHelpAndExit(ctx, "bandwidth", globalErrorExitStatus)
	}
}

func mainAdminBwInfo(ctx *cli.Context) error {
	checkAdminBwInfoSyntax(ctx)
	// Set color preference of command outputs
	console.SetColor("ConfigHeading", color.New(color.Bold, color.FgHiRed))
	console.SetColor("ConfigFG", color.New(color.FgHiWhite))

	var urlStr string
	args := ctx.Args()
	buckets := ctx.StringSlice("buckets")
	if len(buckets) == 0 {
		fatalIf(errInvalidArgument(), "Buckets argument is empty. We need a minimum of one bucket to observe")
	}
	console.PrintC(console.Colorize("BlinkLoad", "COLLECTING SAMPLES & CALCULATING SUM & AVERAGE. These are the initial values..\n"))
	if len(ctx.Args()) == 1 {
		var bwInfo bandwidthSingleStruct
		urlStr = args.Get(0)
		// Create a new MinIO Admin Client
		// For a particular server an admin client.
		client, err := newAdminClient(urlStr)
		fatalIf(err, "Unable to initialize admin connection.")
		getInitValues := true
		bwSampleResult, e := client.SampleBandwidthInfo(globalContext, buckets, getInitValues)
		if e != nil {
			bwInfo.Status = "error"
			bwInfo.Error = e.Error()
			bwInfo.Info = madmin.BandwidthSampleResult{}
		} else {
			bwInfo.Status = "success"
			bwInfo.Error = ""
			bwInfo.Info = madmin.BandwidthSampleResult{}
		}
		bwInfo.Info = bwSampleResult
		bwInfo.Server = urlStr
		bwInfo.Buckets = buckets
		console.RewindLines(1)
		printMsg(bandwidthSingleStruct(bwInfo))

		for {
			getInitValues = false
			bwSampleResult, e = client.SampleBandwidthInfo(globalContext, buckets, getInitValues)
			if e != nil {
				bwInfo.Status = "error"
				bwInfo.Error = e.Error()
				bwInfo.Info = madmin.BandwidthSampleResult{}
			} else {
				bwInfo.Status = "success"
				bwInfo.Error = ""
				bwInfo.Info = madmin.BandwidthSampleResult{}
			}
			bwInfo.Info = bwSampleResult
			bwInfo.Server = urlStr
			bwInfo.Buckets = buckets
			// 5 is the number of lines for 1 bucket. len(buckets) for the newlines + 2 for the newlines above
			console.RewindLines((len(buckets) * 5) + len(buckets))
			printMsg(bandwidthSingleStruct(bwInfo))
		}
	}
	servers := ctx.StringSlice("servers")
	if len(servers) == 0 {
		pErr := probe.NewError(errors.New("No servers mentioned to sample from"))
		fatalIf(pErr, "Please mention 1 server or a server cluster with --servers")
	}
	initBwCollection := fetchInitialBandwidthData(servers, buckets)
	initSampleVal := buildSampleTableDataBucket(initBwCollection)
	console.RewindLines(1)
	printTable(initSampleVal)
	rewindLines := (4 * len(buckets))
	for {
		_, bwBucketSamples := fetchBandwidthDataBucket(servers, buckets)
		bwBucketSampleVal := buildSampleTableDataBucket(bwBucketSamples)
		printTable(bwBucketSampleVal)
		console.RewindLines(rewindLines)
	}
}
