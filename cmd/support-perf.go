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
	"archive/zip"
	gojson "encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
)

var supportPerfFlags = append([]cli.Flag{
	cli.StringFlag{
		Name:  "size",
		Usage: "size of the object used for uploads/downloads",
		Value: "64MiB",
	},
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "display per-server stats",
	},
	cli.StringFlag{
		Name:   "duration",
		Usage:  "maximum duration each perf tests are run",
		Value:  "10s",
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
	cli.BoolFlag{
		Name:   "noclear",
		Usage:  "do not clear bucket after running object perf test",
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
	Flags:           supportPerfFlags,
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
     {{.Prompt}} {{.HelpName}} myminio --airgap
`,
}

// PerfTestOutput - stores the final output of performance test(s)
type PerfTestOutput struct {
	ObjectResults          *ObjTestResults             `json:"object,omitempty"`
	NetResults             *NetTestResults             `json:"network,omitempty"`
	SiteReplicationResults *SiteReplicationTestResults `json:"siteReplication,omitempty"`
	DriveResults           *DriveTestResults           `json:"drive,omitempty"`
	ClientResults          *ClientResult               `json:"client,omitempty"`
	Error                  string                      `json:"error,omitempty"`
}

// DriveTestResult - result of the drive performance test on a given endpoint
type DriveTestResult struct {
	Endpoint string             `json:"endpoint"`
	Perf     []madmin.DrivePerf `json:"perf,omitempty"`
	Error    string             `json:"error,omitempty"`
}

// DriveTestResults - results of drive performance test across all endpoints
type DriveTestResults struct {
	Results []DriveTestResult `json:"servers"`
}

// ObjTestResults - result of the object performance test
type ObjTestResults struct {
	ObjectSize int               `json:"objectSize"`
	Threads    int               `json:"threads"`
	PUTResults ObjPUTPerfResults `json:"PUT"`
	GETResults ObjGETPerfResults `json:"GET"`
}

// ObjStats - Object performance stats
type ObjStats struct {
	Throughput    uint64 `json:"throughput"`
	ObjectsPerSec uint64 `json:"objectsPerSec"`
}

// ObjStatServer - Server level object performance stats
type ObjStatServer struct {
	Endpoint string   `json:"endpoint"`
	Perf     ObjStats `json:"perf"`
	Error    string   `json:"error,omitempty"`
}

// ObjPUTPerfResults - Object PUT performance results
type ObjPUTPerfResults struct {
	Perf    ObjPUTStats     `json:"perf"`
	Servers []ObjStatServer `json:"servers"`
}

// ObjPUTStats - PUT stats of all the servers
type ObjPUTStats struct {
	Throughput    uint64         `json:"throughput"`
	ObjectsPerSec uint64         `json:"objectsPerSec"`
	Response      madmin.Timings `json:"responseTime"`
}

// ObjGETPerfResults - Object GET performance results
type ObjGETPerfResults struct {
	Perf    ObjGETStats     `json:"perf"`
	Servers []ObjStatServer `json:"servers"`
}

// ObjGETStats - GET stats of all the servers
type ObjGETStats struct {
	ObjPUTStats
	TTFB madmin.Timings `json:"ttfb,omitempty"`
}

// NetStats - Network performance stats
type NetStats struct {
	TX uint64 `json:"tx"`
	RX uint64 `json:"rx"`
}

// NetTestResult - result of the network performance test for given endpoint
type NetTestResult struct {
	Endpoint string   `json:"endpoint"`
	Perf     NetStats `json:"perf"`
	Error    string   `json:"error,omitempty"`
}

// NetTestResults - result of the network performance test across all endpoints
type NetTestResults struct {
	Results []NetTestResult `json:"servers"`
}

// ClientResult - result of the network from client to server
type ClientResult struct {
	BytesSent uint64 `json:"bytesSent"`
	TimeSpent int64  `json:"timeSpent"`
	Endpoint  string `json:"endpoint"`
	Error     string `json:"error"`
}

// SiteNetStats - status for siteNet
type SiteNetStats struct {
	TX              uint64        `json:"tx"` // transfer rate in bytes
	TXTotalDuration time.Duration `json:"txTotalDuration"`
	RX              uint64        `json:"rx"` // received rate in bytes
	RXTotalDuration time.Duration `json:"rxTotalDuration"`
	TotalConn       uint64        `json:"totalConn"`
}

// SiteReplicationTestNodeResult - result of the network performance test for site-replication
type SiteReplicationTestNodeResult struct {
	Endpoint string       `json:"endpoint"`
	Perf     SiteNetStats `json:"perf"`
	Error    string       `json:"error,omitempty"`
}

// SiteReplicationTestResults - result of the network performance test across all site-replication
type SiteReplicationTestResults struct {
	Results []SiteReplicationTestNodeResult `json:"servers"`
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

// String - dummy function to confirm to the 'message' interface. Not used.
func (p PerfTestOutput) String() string {
	return ""
}

// JSON - jsonified output of the perf tests
func (p PerfTestOutput) JSON() string {
	JSONBytes, e := json.MarshalIndent(p, "", "    ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON")
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
		if args[0] == "drive" || args[0] == "net" || args[0] == "object" || args[0] == "site-replication" {
			showCommandHelpAndExit(ctx, 1)
		}
		aliasedURL = args[0]

	case 2:
		perfType = args[0]
		aliasedURL = args[1]
	default:
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}

	// Main execution
	execSupportPerf(ctx, aliasedURL, perfType)

	return nil
}

func convertDriveTestResult(dr madmin.DriveSpeedTestResult) DriveTestResult {
	return DriveTestResult{
		Endpoint: dr.Endpoint,
		Perf:     dr.DrivePerf,
		Error:    dr.Error,
	}
}

func convertDriveTestResults(driveResults []madmin.DriveSpeedTestResult) *DriveTestResults {
	if driveResults == nil {
		return nil
	}
	results := []DriveTestResult{}
	for _, dr := range driveResults {
		results = append(results, convertDriveTestResult(dr))
	}
	r := DriveTestResults{
		Results: results,
	}
	return &r
}

func convertClientResult(result *madmin.ClientPerfResult) *ClientResult {
	if result == nil || result.TimeSpent <= 0 {
		return nil
	}
	return &ClientResult{
		BytesSent: result.BytesSend,
		TimeSpent: result.TimeSpent,
		Endpoint:  result.Endpoint,
		Error:     result.Error,
	}
}

func convertSiteReplicationTestResults(netResults *madmin.SiteNetPerfResult) *SiteReplicationTestResults {
	if netResults == nil {
		return nil
	}
	results := []SiteReplicationTestNodeResult{}
	for _, nr := range netResults.NodeResults {
		results = append(results, SiteReplicationTestNodeResult{
			Endpoint: nr.Endpoint,
			Error:    nr.Error,
			Perf: SiteNetStats{
				TX:              nr.TX,
				TXTotalDuration: nr.TXTotalDuration,
				RX:              nr.RX,
				RXTotalDuration: nr.RXTotalDuration,
				TotalConn:       nr.TotalConn,
			},
		})
	}
	r := SiteReplicationTestResults{
		Results: results,
	}
	return &r
}

func convertNetTestResults(netResults *madmin.NetperfResult) *NetTestResults {
	if netResults == nil {
		return nil
	}
	results := []NetTestResult{}
	for _, nr := range netResults.NodeResults {
		results = append(results, NetTestResult{
			Endpoint: nr.Endpoint,
			Error:    nr.Error,
			Perf: NetStats{
				TX: nr.TX,
				RX: nr.RX,
			},
		})
	}
	r := NetTestResults{
		Results: results,
	}
	return &r
}

func convertObjStatServers(ss []madmin.SpeedTestStatServer) []ObjStatServer {
	out := []ObjStatServer{}
	for _, s := range ss {
		out = append(out, ObjStatServer{
			Endpoint: s.Endpoint,
			Perf: ObjStats{
				Throughput:    s.ThroughputPerSec,
				ObjectsPerSec: s.ObjectsPerSec,
			},
			Error: s.Err,
		})
	}
	return out
}

func convertPUTStats(stats madmin.SpeedTestStats) ObjPUTStats {
	return ObjPUTStats{
		Throughput:    stats.ThroughputPerSec,
		ObjectsPerSec: stats.ObjectsPerSec,
		Response:      stats.Response,
	}
}

func convertPUTResults(stats madmin.SpeedTestStats) ObjPUTPerfResults {
	return ObjPUTPerfResults{
		Perf:    convertPUTStats(stats),
		Servers: convertObjStatServers(stats.Servers),
	}
}

func convertGETResults(stats madmin.SpeedTestStats) ObjGETPerfResults {
	return ObjGETPerfResults{
		Perf: ObjGETStats{
			ObjPUTStats: convertPUTStats(stats),
			TTFB:        stats.TTFB,
		},
		Servers: convertObjStatServers(stats.Servers),
	}
}

func convertObjTestResults(objResult *madmin.SpeedTestResult) *ObjTestResults {
	if objResult == nil {
		return nil
	}
	result := ObjTestResults{
		ObjectSize: objResult.Size,
		Threads:    objResult.Concurrent,
	}
	result.PUTResults = convertPUTResults(objResult.PUTStats)
	result.GETResults = convertGETResults(objResult.GETStats)
	return &result
}

func updatePerfOutput(r PerfTestResult, out *PerfTestOutput) {
	switch r.Type {
	case DrivePerfTest:
		out.DriveResults = convertDriveTestResults(r.DriveResult)
	case ObjectPerfTest:
		out.ObjectResults = convertObjTestResults(r.ObjectResult)
	case NetPerfTest:
		out.NetResults = convertNetTestResults(r.NetResult)
	case SiteReplicationPerfTest:
		out.SiteReplicationResults = convertSiteReplicationTestResults(r.SiteReplicationResult)
	case ClientPerfTest:
		out.ClientResults = convertClientResult(r.ClientResult)
	default:
		fatalIf(errDummy().Trace(), fmt.Sprintf("Invalid test type %d", r.Type))
	}
}

func convertPerfResult(r PerfTestResult) PerfTestOutput {
	out := PerfTestOutput{}
	updatePerfOutput(r, &out)
	return out
}

func convertPerfResults(results []PerfTestResult) PerfTestOutput {
	out := PerfTestOutput{}
	for _, r := range results {
		updatePerfOutput(r, &out)
	}
	return out
}

func execSupportPerf(ctx *cli.Context, aliasedURL, perfType string) {
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

	// If results still not available, don't write anything
	if len(results) == 0 {
		console.Fatalln("No performance reports were captured, please report this issue")
	} else {
		resultFileNamePfx := fmt.Sprintf("%s-perf_%s", filepath.Clean(alias), UTCNow().Format("20060102150405"))
		resultFileName := resultFileNamePfx + ".json"

		regInfo := GetClusterRegInfo(getAdminInfo(aliasedURL), alias)
		tmpFileName, e := zipPerfResult(convertPerfResults(results), resultFileName, regInfo)
		fatalIf(probe.NewError(e), "Unable to generate zip file from performance results")

		if globalAirgapped {
			console.Infoln()
			savePerfResultFile(tmpFileName, resultFileNamePfx)
			return
		}

		uploadURL := SubnetUploadURL("perf")
		reqURL, headers := prepareSubnetUploadURL(uploadURL, alias, apiKey)

		_, e = (&SubnetFileUploader{
			alias:             alias,
			FilePath:          tmpFileName,
			ReqURL:            reqURL,
			Headers:           headers,
			DeleteAfterUpload: true,
		}).UploadFileToSubnet()
		if e != nil {
			errorIf(probe.NewError(e), "Unable to upload performance results to SUBNET portal")
			savePerfResultFile(tmpFileName, resultFileNamePfx)
			return
		}

		console.Infoln("Uploaded performance report to SUBNET successfully")
	}
}

func savePerfResultFile(tmpFileName, resultFileNamePfx string) {
	zipFileName := resultFileNamePfx + ".zip"
	e := moveFile(tmpFileName, zipFileName)
	fatalIf(probe.NewError(e), fmt.Sprintf("Unable to move %s -> %s", tmpFileName, zipFileName))
	console.Infof("MinIO performance report saved at %s, please upload to SUBNET portal manually\n", zipFileName)
}

func runPerfTests(ctx *cli.Context, aliasedURL, perfType string) []PerfTestResult {
	resultCh := make(chan PerfTestResult)
	results := []PerfTestResult{}
	defer close(resultCh)

	tests := []string{perfType}
	if len(perfType) == 0 {
		// by default run all tests
		tests = []string{"net", "drive", "object", "client"}
	}

	for _, t := range tests {
		switch t {
		case "drive":
			mainAdminSpeedTestDrive(ctx, aliasedURL, resultCh)
		case "object":
			mainAdminSpeedTestObject(ctx, aliasedURL, resultCh)
		case "net":
			mainAdminSpeedTestNetperf(ctx, aliasedURL, resultCh)
		case "site-replication":
			mainAdminSpeedTestSiteReplication(ctx, aliasedURL, resultCh)
		case "client":
			mainAdminSpeedTestClientPerf(ctx, aliasedURL, resultCh)
		default:
			showCommandHelpAndExit(ctx, 1) // last argument is exit code
		}

		if !globalJSON {
			results = append(results, <-resultCh)
		}
	}

	return results
}

func writeJSONObjToZip(zipWriter *zip.Writer, obj any, filename string) error {
	writer, e := zipWriter.Create(filename)
	if e != nil {
		return e
	}

	return gojson.NewEncoder(writer).Encode(obj)
}

// compress MinIO performance output
func zipPerfResult(perfOutput PerfTestOutput, resultFilename string, regInfo ClusterRegistrationInfo) (string, error) {
	// Create perf results zip file
	tmpArchive, e := os.CreateTemp("", "mc-perf-*.zip")

	if e != nil {
		return "", e
	}
	defer tmpArchive.Close()

	zipWriter := zip.NewWriter(tmpArchive)
	defer zipWriter.Close()

	e = writeJSONObjToZip(zipWriter, perfOutput, resultFilename)
	if e != nil {
		return "", e
	}

	e = writeJSONObjToZip(zipWriter, regInfo, "cluster.info")
	if e != nil {
		return "", e
	}

	return tmpArchive.Name(), nil
}
