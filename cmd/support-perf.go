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
	"archive/zip"
	gojson "encoding/json"
	"fmt"
	"os"
	"path/filepath"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var supportPerfFlags = append([]cli.Flag{
	cli.StringFlag{
		Name:  "duration",
		Usage: "duration the entire perf tests are run",
		Value: "10s",
	},
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "display per-server stats",
	},
	cli.StringFlag{
		Name:   "size",
		Usage:  "size of the object used for uploads/downloads",
		Value:  "64MiB",
		Hidden: true,
	},
	cli.IntFlag{
		Name:   "concurrent",
		Usage:  "number of concurrent requests per server",
		Value:  32,
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "bucket",
		Usage:  "provide a custom bucket name to use (NOTE: bucket must be created prior)",
		Hidden: true, // Hidden for now.
	},
	// Drive test specific flags.
	cli.StringFlag{
		Name:   "filesize",
		Usage:  "total amount of data read/written to each drive",
		Value:  "1GiB",
		Hidden: true,
	},
	cli.StringFlag{
		Name:   "blocksize",
		Usage:  "read/write block size",
		Value:  "4MiB",
		Hidden: true,
	},
	cli.BoolFlag{
		Name:   "serial",
		Usage:  "run tests on drive(s) one-by-one",
		Hidden: true,
	},
}, subnetCommonFlags...)

var supportPerfCmd = cli.Command{
	Name:            "perf",
	Usage:           "upload object, network and drive performance analysis",
	Action:          mainSupportPerf,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportPerfFlags, supportGlobalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [COMMAND] [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

EXAMPLES:
  1. Upload object storage, network, and drive performance analysis for cluster with alias 'myminio' to SUBNET
     {{.Prompt}} {{.HelpName}} myminio
  2. Run object storage, network, and drive performance tests on cluster with alias 'myminio', save and upload to SUBNET manually
     {{.Prompt}} {{.HelpName}} --airgap myminio
`,
}

func objectTestVerboseResult(result *madmin.SpeedTestResult) (msg string) {
	msg += "PUT:\n"
	for _, node := range result.PUTStats.Servers {
		msg += fmt.Sprintf("   * %s: %s/s %s objs/s", node.Endpoint, humanize.IBytes(node.ThroughputPerSec), humanize.Comma(int64(node.ObjectsPerSec)))
		if node.Err != "" {
			msg += " Err: " + node.Err
		}
		msg += "\n"
	}

	msg += "GET:\n"
	for _, node := range result.GETStats.Servers {
		msg += fmt.Sprintf("   * %s: %s/s %s objs/s", node.Endpoint, humanize.IBytes(node.ThroughputPerSec), humanize.Comma(int64(node.ObjectsPerSec)))
		if node.Err != "" {
			msg += " Err: " + node.Err
		}
		msg += "\n"
	}

	return msg
}

func objectTestShortResult(result *madmin.SpeedTestResult) (msg string) {
	msg += fmt.Sprintf("MinIO %s, %d servers, %d drives, %s objects, %d threads",
		result.Version, result.Servers, result.Disks,
		humanize.IBytes(uint64(result.Size)), result.Concurrent)

	return msg
}

func (p PerfTestResult) String() string {
	return ""
}

// JSON - jsonified update message.
func (p PerfTestResult) JSON() string {
	JSONBytes, e := json.MarshalIndent(p, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(JSONBytes)
}

var globalPerfTestVerbose bool

func mainSupportPerf(ctx *cli.Context) error {
	args := ctx.Args()

	// the alias parameter from cli
	aliasedURL := ""
	perfType := ""
	switch len(args) {
	case 1:
		// cannot use alias by the name 'drive' or 'net'
		if args[0] == "drive" || args[0] == "net" || args[0] == "object" {
			showCommandHelpAndExit(ctx, "perf", 1)
		}
		aliasedURL = args[0]

	case 2:
		perfType = args[0]
		aliasedURL = args[1]
	default:
		showCommandHelpAndExit(ctx, "perf", 1) // last argument is exit code
	}

	// Main execution
	execSupportPerf(ctx, aliasedURL, perfType)

	return nil
}

func execSupportPerf(ctx *cli.Context, aliasedURL string, perfType string) {
	alias, apiKey := initSubnetConnectivity(ctx, aliasedURL, true)
	if len(apiKey) == 0 {
		// api key not passed as flag. Check that the cluster is registered.
		apiKey = validateClusterRegistered(alias, true)
	}

	results := runPerfTests(ctx, aliasedURL, perfType)
	if globalJSON {
		// No file to be saved or uploaded to SUBNET in case of `--json`
		return
	}

	resultFileNamePfx := fmt.Sprintf("%s-perf_%s", filepath.Clean(alias), UTCNow().Format("20060102150405"))
	resultFileName := resultFileNamePfx + ".json"

	regInfo := getClusterRegInfo(getAdminInfo(aliasedURL), alias)
	tmpFileName, e := zipPerfResult(results, resultFileName, regInfo)
	fatalIf(probe.NewError(e), "Error creating zip from perf test results:")

	if globalAirgapped {
		savePerfResultFile(tmpFileName, resultFileNamePfx, alias)
		return
	}

	uploadURL := subnetUploadURL("perf", tmpFileName)
	reqURL, headers := prepareSubnetUploadURL(uploadURL, alias, tmpFileName, apiKey)

	_, e = uploadFileToSubnet(alias, tmpFileName, reqURL, headers)
	if e != nil {
		console.Errorln("Unable to upload perf test results to SUBNET portal: " + e.Error())
		savePerfResultFile(tmpFileName, resultFileNamePfx, alias)
		return
	}

	clr := color.New(color.FgGreen, color.Bold)
	clr.Println("uploaded successfully to SUBNET.")
}

func savePerfResultFile(tmpFileName string, resultFileNamePfx string, alias string) {
	zipFileName := resultFileNamePfx + ".zip"
	e := moveFile(tmpFileName, zipFileName)
	fatalIf(probe.NewError(e), fmt.Sprintf("Error moving temp file %s to %s:", tmpFileName, zipFileName))
	console.Infoln("MinIO performance report saved at", zipFileName)
}

func runPerfTests(ctx *cli.Context, aliasedURL string, perfType string) []PerfTestResult {
	resultCh := make(chan PerfTestResult)
	results := []PerfTestResult{}
	defer close(resultCh)

	tests := []string{perfType}
	if len(perfType) == 0 {
		// by default run all tests
		tests = []string{"net", "drive", "object"}
	}

	for _, t := range tests {
		switch t {
		case "drive":
			mainAdminSpeedTestDrive(ctx, aliasedURL, resultCh)
		case "object":
			mainAdminSpeedTestObject(ctx, aliasedURL, resultCh)
		case "net":
			mainAdminSpeedTestNetperf(ctx, aliasedURL, resultCh)
		default:
			showCommandHelpAndExit(ctx, "perf", 1) // last argument is exit code
		}

		if !globalJSON {
			results = append(results, <-resultCh)
		}
	}

	return results
}

func writeJSONObjToZip(zipWriter *zip.Writer, obj interface{}, filename string) error {
	writer, e := zipWriter.Create(filename)
	if e != nil {
		return e
	}

	enc := gojson.NewEncoder(writer)
	if e = enc.Encode(obj); e != nil {
		return e
	}

	return nil
}

// compress MinIO performance output
func zipPerfResult(perfResult []PerfTestResult, resultFilename string, regInfo ClusterRegistrationInfo) (string, error) {
	// Create profile zip file
	tmpArchive, e := os.CreateTemp("", "mc-perf-")

	if e != nil {
		return "", e
	}
	defer tmpArchive.Close()

	zipWriter := zip.NewWriter(tmpArchive)
	defer zipWriter.Close()

	e = writeJSONObjToZip(zipWriter, perfResult, resultFilename)
	if e != nil {
		return "", e
	}

	e = writeJSONObjToZip(zipWriter, regInfo, "cluster.info")
	if e != nil {
		return "", e
	}

	return tmpArchive.Name(), nil
}
