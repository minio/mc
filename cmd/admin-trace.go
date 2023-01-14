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
	"bytes"
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/gzip"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/klauspost/compress/zstd"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v2"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

var adminTraceFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "print verbose trace",
	},
	cli.BoolFlag{
		Name:  "all, a",
		Usage: "trace all call types",
	},
	cli.StringSliceFlag{
		Name:  "call",
		Usage: "trace only matching call types. See CALL TYPES below for list. (default: s3)",
	},
	cli.IntSliceFlag{
		Name:  "status-code",
		Usage: "trace only matching status code",
	},
	cli.StringSliceFlag{
		Name:  "method",
		Usage: "trace only matching HTTP method",
	},
	cli.StringSliceFlag{
		Name:  "funcname",
		Usage: "trace only matching func name",
	},
	cli.StringSliceFlag{
		Name:  "path",
		Usage: "trace only matching path",
	},
	cli.StringSliceFlag{
		Name:  "node",
		Usage: "trace only matching servers",
	},
	cli.StringSliceFlag{
		Name:  "request-header",
		Usage: "trace only matching request headers",
	},
	cli.BoolFlag{
		Name:  "errors, e",
		Usage: "trace only failed requests",
	},
	cli.BoolFlag{
		Name:  "filter-request",
		Usage: "trace calls only with request bytes greater than this threshold, use with filter-size",
	},
	cli.BoolFlag{
		Name:  "filter-response",
		Usage: "trace calls only with response bytes greater than this threshold, use with filter-size",
	},
	cli.BoolFlag{
		Name:  "response-duration",
		Usage: "trace calls only with response duration greater than this threshold (e.g. `5ms`)",
	},
	cli.StringFlag{
		Name:  "filter-size",
		Usage: "filter size, use with filter (see UNITS)",
	},
	cli.StringFlag{
		Name:  "rotate-trace-logs-size",
		Usage: "maximum size of a rotated trace log file (see UNITS)",
	},
	cli.StringFlag{
		Name:  "rotate-trace-logs-frequency",
		Usage: "frequency, in terms of time, of trace log file rotations (see time UNITS), minimum 1 minute",
	},
	cli.StringFlag{
		Name:  "rotate-trace-logs-pattern",
		Usage: "pattern (based on golang https://go.dev/src/time/format.go) of directories and files into which trace log files are placed",
	},
	cli.StringFlag{
		Name:  "rotate-trace-logs-compress",
		Usage: "compress in place using 'gzip' or 'zstd' (Zstandard)",
	},
}

// traceCallTypes contains all call types and flags to apply when selected.
var traceCallTypes = map[string]func(o *madmin.ServiceTraceOpts) (help string){
	"storage":   func(o *madmin.ServiceTraceOpts) string { o.Storage = true; return "Trace Storage calls" },
	"internal":  func(o *madmin.ServiceTraceOpts) string { o.Internal = true; return "Trace Internal RPC calls" },
	"s3":        func(o *madmin.ServiceTraceOpts) string { o.S3 = true; return "Trace S3 API calls" },
	"os":        func(o *madmin.ServiceTraceOpts) string { o.OS = true; return "Trace Operating System calls" },
	"scanner":   func(o *madmin.ServiceTraceOpts) string { o.Scanner = true; return "Trace Scanner calls" },
	"bootstrap": func(o *madmin.ServiceTraceOpts) string { o.Bootstrap = true; return "Trace Bootstrap operations" },
	"healing": func(o *madmin.ServiceTraceOpts) string {
		o.Healing = true
		return "Trace Healing operations (alias: heal)"
	},
	"batch-replication": func(o *madmin.ServiceTraceOpts) string {
		o.BatchReplication = true
		return "Trace Batch Replication (alias: brep)"
	},
	"decommission": func(o *madmin.ServiceTraceOpts) string {
		o.Decommission = true
		return "Trace Decommission operations (alias: decom)"
	},
	"rebalance": func(o *madmin.ServiceTraceOpts) string {
		o.Rebalance = true
		return "Trace Server Pool Rebalancing operations"
	},
	"replication-resync": func(o *madmin.ServiceTraceOpts) string {
		o.ReplicationResync = true
		return "Trace Replication Resync operations (alias: resync)"
	},
}

// traceCallTypes contains aliases (short versions) of
var traceCallTypeAliases = map[string]func(o *madmin.ServiceTraceOpts) string{
	"heal":   traceCallTypes["healing"],
	"decom":  traceCallTypes["decommission"],
	"resync": traceCallTypes["replication-resync"],
	"brep":   traceCallTypes["batch-replication"],
}

func traceCallsHelp() string {
	var help []string
	o := madmin.ServiceTraceOpts{}
	const padkeyLen = 19
	for k, fn := range traceCallTypes {
		pad := ""
		if len(k) < padkeyLen {
			pad = strings.Repeat(" ", padkeyLen-len(k))
		}
		help = append(help, fmt.Sprintf("  %s: %s%s", k, pad, fn(&o)))
	}
	sort.Strings(help)
	return strings.Join(help, "\n")
}

var adminTraceCmd = cli.Command{
	Name:            "trace",
	Usage:           "show http trace for MinIO server",
	Action:          mainAdminTrace,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminTraceFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}

CALL TYPES:
` + traceCallsHelp() + `
UNITS
  --filter-size flags (use with --filter-response or --filter-request), as 
  well as the --rotate-trace-logs-size flag, accept human-readable case-
  insensitive number suffixes such as "k", "m", "g" and "t" referring to 
  the metric units KB, MB, GB and TB respectively. Adding an "i" to these 
  prefixes, uses the IEC units, so that "gi" refers to "gibibyte" or "GiB". 
  A "b" at the end is also accepted. Without suffixes the unit is bytes.

  --rotate-trace-logs-frequency accepts the human-readable number suffixes 
  "m", "h", "d", "w", "y" referring to the time units minutes, hours, days, 
  weeks, and years respectively. 
  If pattern-based naming is used, then the corresponding trace log files 
  will indicate the lower bound of the current time interval. 
  e.g. minio-log-18-00-00.log will contain logs from [18:00..20:00)

EXAMPLES:
  1. Show verbose console trace for MinIO server
     {{.Prompt}} {{.HelpName}} -v -a myminio

  2. Show trace only for failed requests for MinIO server
    {{.Prompt}} {{.HelpName}} -v -e myminio

  3. Show verbose console trace for requests with '503' status code
    {{.Prompt}} {{.HelpName}} -v --status-code 503 myminio

  4. Show console trace for a specific path
    {{.Prompt}} {{.HelpName}} --path my-bucket/my-prefix/* myminio

  5. Show console trace for requests with '404' and '503' status code
    {{.Prompt}} {{.HelpName}} --status-code 404 --status-code 503 myminio
  
  6. Show trace only for requests bytes greater than 1MB
    {{.Prompt}} {{.HelpName}} --filter-request --filter-size 1MB myminio

  7. Show trace only for response bytes greater than 1MB
    {{.Prompt}} {{.HelpName}} --filter-response --filter-size 1MB myminio
  
  8. Show trace only for requests operations duration greater than 5ms
     {{.Prompt}} {{.HelpName}} --response-duration 5ms myminio

  9. Log to a new default trace file trace when the previous log trace file reaches 1MB then compress in place using gzip
     {{.Prompt}} {{.HelpName}} --rotate-trace-logs-size 1MB --rotate-trace-logs-compress gzip myminio

  10. Log to a new trace file trace every hour, in a directory hierarchy to log files with timestamp suffixes
     {{.Prompt}} {{.HelpName}} --rotate-trace-logs-frequency 1h --rotate-trace-logs-pattern "/tmp/minio/[2006-01-02]/minio-log-[15-04-05].log" myminio
`,
}

const traceTimeFormat = "2006-01-02T15:04:05.000"

var colors = []color.Attribute{color.FgCyan, color.FgWhite, color.FgYellow, color.FgGreen}

func checkAdminTraceSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
	filterFlag := ctx.Bool("filter-request") || ctx.Bool("filter-response")
	if filterFlag && ctx.String("filter-size") == "" {
		// filter must use with filter-size flags
		showCommandHelpAndExit(ctx, 1)
	}

	if ctx.Bool("all") && len(ctx.StringSlice("call")) > 0 {
		fatalIf(errDummy().Trace(), "You cannot specify both --all and --call flags at the same time.")
	}

	if ctx.String("rotate-trace-logs-size") != "" && ctx.String("rotate-trace-logs-frequency") != "" {
		fatalIf(errDummy().Trace(), "You cannot specify both --rotate-trace-logs-size and --rotate-trace-logs-frequency flags at the same time.")
	}

	if ctx.String("rotate-trace-logs-pattern") != "" && ctx.String("rotate-trace-logs-size") == "" && ctx.String("rotate-trace-logs-frequency") == "" {
		fatalIf(errDummy().Trace(), "You cannot specify flag --rotate-trace-logs-pattern if neither --rotate-trace-logs-size nor --rotate-trace-logs-frequency flags are specified as well.")
	}
}

func printTrace(verbose bool, traceInfo madmin.ServiceTraceInfo) {
	if verbose {
		printMsg(traceMessage{ServiceTraceInfo: traceInfo})
	} else {
		printMsg(shortTrace(traceInfo))
	}
}

type matchString struct {
	val     string
	reverse bool
}

type matchOpts struct {
	statusCodes              []int
	methods                  []string
	funcNames                []string
	apiPaths                 []string
	nodes                    []string
	reqHeaders               []matchString
	requestSize              uint64
	responseSize             uint64
	rotateTraceLogsSize      uint64
	rotateTraceLogsFrequency time.Duration
	rotateTraceLogsPattern   string
	rotateTraceLogsCompress  string
}

var (
	traceLogsEnabled, traceLogsStarted                     bool
	traceLogsStartTime                                     time.Time
	traceLogsIntervalType                                  time.Duration
	traceLogsFilePath, traceLogsPattern, traceLogsCompress string
	traceLogsFile                                          *os.File
	traceLogsFileZstdEncoder                               *zstd.Encoder
	traceLogsFileSize                                      int
)

func matchTrace(opts matchOpts, traceInfo madmin.ServiceTraceInfo) bool {
	// Filter request path if passed by the user
	if len(opts.apiPaths) > 0 {
		matched := false
		for _, apiPath := range opts.apiPaths {
			if pathMatch(path.Join("/", apiPath), traceInfo.Trace.Path) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Filter response status codes if passed by the user
	if len(opts.statusCodes) > 0 {
		matched := false
		for _, code := range opts.statusCodes {
			if traceInfo.Trace.HTTP != nil && traceInfo.Trace.HTTP.RespInfo.StatusCode == code {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}

	}

	// Filter request method if passed by the user
	if len(opts.methods) > 0 {
		matched := false
		for _, method := range opts.methods {
			if traceInfo.Trace.HTTP != nil && traceInfo.Trace.HTTP.ReqInfo.Method == method {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}

	}

	if len(opts.funcNames) > 0 {
		matched := false
		// Filter request function handler names if passed by the user.
		for _, funcName := range opts.funcNames {
			if nameMatch(funcName, traceInfo.Trace.FuncName) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(opts.nodes) > 0 {
		matched := false
		// Filter request by node if passed by the user.
		for _, node := range opts.nodes {
			if nameMatch(node, traceInfo.Trace.NodeName) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(opts.reqHeaders) > 0 && traceInfo.Trace.HTTP != nil {
		matched := false
		for _, hdr := range opts.reqHeaders {
			headerFound := false
			for traceHdr, traceVals := range traceInfo.Trace.HTTP.ReqInfo.Headers {
				for _, traceVal := range traceVals {
					if headerMatch(hdr.val, traceHdr+": "+traceVal) {
						headerFound = true
						goto exitFindingHeader
					}
				}
			}
		exitFindingHeader:
			if !hdr.reverse && headerFound || hdr.reverse && !headerFound {
				matched = true
				goto exitMatchingHeader
			}
		}
	exitMatchingHeader:
		if !matched {
			return false
		}
	}

	if opts.rotateTraceLogsSize > 0 {
		if traceLogsFileSize >= int(opts.rotateTraceLogsSize) {
			// Reset trace log file size tracker
			traceLogsFileSize = 0
			// Prepare to create a new log file
			traceLogsStartTime = time.Now()
			// Compress current file
			if traceLogsCompress != "" {
				compressTraceLogsFile()
			}
			// Create new trace log file
			traceLogsFile = createTraceLogsFile()
		}
	}

	if opts.rotateTraceLogsFrequency > 0 {
		currentTime := time.Now()
		if !traceLogsStarted {
			traceLogsStarted = true
			traceLogsStartTime = traceLogsStartTime.Add(time.Duration(opts.rotateTraceLogsFrequency))
		}
		nextTraceLogsStartTime := traceLogsStartTime
		for currentTime.After(nextTraceLogsStartTime) {
			// Find latest time slice
			// Prepare to create a new log file
			nextTraceLogsStartTime = nextTraceLogsStartTime.Add(time.Duration(opts.rotateTraceLogsFrequency))

			if nextTraceLogsStartTime.After(currentTime) || nextTraceLogsStartTime.Equal(currentTime) {
				// Compress current file
				if traceLogsCompress != "" {
					compressTraceLogsFile()
				}
				// Create new trace log file
				traceLogsFile = createTraceLogsFile()
				traceLogsStartTime = nextTraceLogsStartTime
			}
		}
	}

	if opts.requestSize > 0 && traceInfo.Trace.HTTP.CallStats.InputBytes < int(opts.requestSize) {
		return false
	}

	if opts.responseSize > 0 && traceInfo.Trace.HTTP.CallStats.OutputBytes < int(opts.responseSize) {
		return false
	}

	return true
}

func matchingOpts(ctx *cli.Context) (opts matchOpts) {
	opts.statusCodes = ctx.IntSlice("status-code")
	opts.methods = ctx.StringSlice("method")
	opts.funcNames = ctx.StringSlice("funcname")
	opts.apiPaths = ctx.StringSlice("path")
	opts.nodes = ctx.StringSlice("node")
	for _, s := range ctx.StringSlice("request-header") {
		ms := matchString{}
		ms.reverse = strings.HasPrefix(s, "!")
		ms.val = strings.TrimPrefix(s, "!")
		opts.reqHeaders = append(opts.reqHeaders, ms)
	}
	var e error
	var requestSize, responseSize uint64
	var rotateTraceLogsSize uint64
	var rotateTraceLogsFrequency time.Duration
	var rotateTraceLogsPattern, rotateTraceLogsCompress string
	if ctx.Bool("filter-request") && ctx.String("filter-size") != "" {
		requestSize, e = humanize.ParseBytes(ctx.String("filter-size"))
		fatalIf(probe.NewError(e).Trace(ctx.String("filter-size")), "Unable to parse input bytes.")
	}

	if ctx.Bool("filter-response") && ctx.String("filter-size") != "" {
		responseSize, e = humanize.ParseBytes(ctx.String("filter-size"))
		fatalIf(probe.NewError(e).Trace(ctx.String("filter-size")), "Unable to parse input bytes.")
	}

	if ctx.String("rotate-trace-logs-size") != "" {
		rotateTraceLogsSize, e = humanize.ParseBytes(ctx.String("rotate-trace-logs-size"))
		fatalIf(probe.NewError(e).Trace(ctx.String("rotate-trace-logs-size")), "Unable to parse input trace logs bytes.")
	}

	if ctx.String("rotate-trace-logs-frequency") != "" {
		rotateTraceLogsFrequency, e = time.ParseDuration(ctx.String("rotate-trace-logs-frequency"))
		fatalIf(probe.NewError(e).Trace(ctx.String("rotate-trace-logs-frequency")), "Unable to parse input trace logs frequency.")
		// Enforce minimum frequency
		if rotateTraceLogsFrequency < time.Minute {
			fatalIf(errDummy().Trace(), "Frequency must be a minimum of every 1 minute.")
		}
		// Obtain time interval type. Default to day
		pattern := regexp.MustCompile(`.*([a-zA-Z])+`)
		subMatches := pattern.FindStringSubmatch(ctx.String("rotate-trace-logs-frequency"))
		switch subMatches[1] {
		case "m":
			traceLogsIntervalType = time.Second * 60
		case "h":
			traceLogsIntervalType = time.Second * 60 * 60
		case "d":
			traceLogsIntervalType = time.Second * 60 * 60 * 24
		default:
			traceLogsIntervalType = time.Second * 60 * 60 * 24
		}
	}

	if ctx.String("rotate-trace-logs-pattern") != "" {
		rotateTraceLogsPattern = ctx.String("rotate-trace-logs-pattern")
		traceLogsPattern = rotateTraceLogsPattern
	}

	if ctx.String("rotate-trace-logs-compress") != "" {
		rotateTraceLogsCompress = ctx.String("rotate-trace-logs-compress")
		traceLogsCompress = rotateTraceLogsCompress
	}

	opts.requestSize = requestSize
	opts.responseSize = responseSize
	opts.rotateTraceLogsSize = rotateTraceLogsSize
	opts.rotateTraceLogsFrequency = rotateTraceLogsFrequency
	opts.rotateTraceLogsPattern = rotateTraceLogsPattern
	opts.rotateTraceLogsCompress = rotateTraceLogsCompress
	return
}

// Write trace log file
func writeTracelogsFile(log string) {
	traceLogsFileSize += len(log)
	if _, e := traceLogsFile.WriteString(log + "\n"); e != nil {
		traceLogsFile.Close()
		fatalIf(probe.NewError(e), "Unable to write trace log file `"+defineTraceLogsFile()+"`")
	}
}

// Compress trace log file
func compressTraceLogsFile() {
	// Create separate fd for asynch compress
	switch traceLogsCompress {
	case "gzip":
		go compressGzip(traceLogsFile, traceLogsFilePath)
	case "zstd":
		go compressZstd(traceLogsFile, traceLogsFilePath)
	default:
		break
	}
}

// gzip
func compressGzip(asynchTraceLogsFile *os.File, asynchTraceLogsFilePath string) {
	// Remove raw file
	defer func() {
		if e := os.Remove(asynchTraceLogsFilePath); e != nil {
			// Ignore "no such file or directory". Another log archive already contains this data
			if errors.Is(e, os.ErrNotExist) {
				return
			}
			fatalIf(probe.NewError(e), "Unable to remove current trace log file `"+asynchTraceLogsFilePath+"`")
		}
	}()
	compressFile, e := os.OpenFile(asynchTraceLogsFilePath+".gz", os.O_CREATE|os.O_RDWR, 0o644)
	if e != nil {
		compressFile.Close()
		asynchTraceLogsFile.Close()
		fatalIf(probe.NewError(e), "Unable to open trace log file archive for gzip `"+asynchTraceLogsFilePath+"`")
	}
	asynchTraceLogsFile.Seek(0, io.SeekStart)
	writer := gzip.NewWriter(compressFile)
	_, e = io.Copy(writer, asynchTraceLogsFile)
	writer.Close()
	compressFile.Close()
	asynchTraceLogsFile.Close()
	fatalIf(probe.NewError(e), "Unable to write trace log file archive for gzip `"+asynchTraceLogsFilePath+"`")
}

// zstandard
func compressZstd(asynchTraceLogsFile *os.File, asynchTraceLogsFilePath string) {
	// Remove raw file
	defer func() {
		if e := os.Remove(asynchTraceLogsFilePath); e != nil {
			fatalIf(probe.NewError(e), "Unable to remove current trace log file `"+asynchTraceLogsFilePath+"`")
		}
	}()
	compressFile, e := os.OpenFile(asynchTraceLogsFilePath+".zst", os.O_CREATE|os.O_RDWR, 0o644)
	if e != nil {
		compressFile.Close()
		asynchTraceLogsFile.Close()
		fatalIf(probe.NewError(e), "Unable to open trace log file archive for zstd `"+asynchTraceLogsFilePath+"`")
	}
	contentSize, _ := asynchTraceLogsFile.Seek(0, io.SeekCurrent)
	asynchTraceLogsFile.Seek(0, io.SeekStart)

	// Reuse encoder to reuse resources and avoid unnecessary allocation
	if traceLogsFileZstdEncoder == nil {
		traceLogsFileZstdEncoder, e = zstd.NewWriter(nil)
		if e != nil {
			traceLogsFileZstdEncoder.Close()
			compressFile.Close()
			asynchTraceLogsFile.Close()
			fatalIf(probe.NewError(e), "Unable to create encoder for zstd `"+asynchTraceLogsFilePath+"`")
		}
	}
	traceLogsFileZstdEncoder.ResetContentSize(compressFile, contentSize)

	_, e = io.Copy(traceLogsFileZstdEncoder, asynchTraceLogsFile)
	traceLogsFileZstdEncoder.Close()
	compressFile.Close()
	asynchTraceLogsFile.Close()
	fatalIf(probe.NewError(e), "Unable to copy encoder for zstd `"+asynchTraceLogsFilePath+"`")
}

// Create trace log file
func createTraceLogsFile() *os.File {
	// Create and open file if it does not exist
	traceLogsFile, e := os.OpenFile(defineTraceLogsFile(), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0o644)
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to create trace log file `"+defineTraceLogsFile()+"`")
	}
	console.Println("Now logging to", traceLogsFilePath)
	return traceLogsFile
}

// Derive new trace log file name
func defineTraceLogsFile() string {
	// Identify trace log format.Time pattern within square brackets, to create a customizable and dynamic path
	pattern := regexp.MustCompile(`\[([^\[\]]+)\]`)
	traceLogsDirectory := pattern.ReplaceAllStringFunc(traceLogsPattern, func(match string) string {
		subMatches := pattern.FindStringSubmatch(match)
		return traceLogsStartTime.Format(subMatches[1])
	})
	traceLogsDirectoryName, traceLogsFileName := filepath.Split(traceLogsDirectory)
	// Default directory and file name if not provided
	if traceLogsDirectoryName == "" {
		defineTraceLogsDirectoryName, e := os.Getwd()
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to deterine current working directory")
		}
		traceLogsDirectoryName = defineTraceLogsDirectoryName
	}
	if traceLogsFileName == "" {
		// Default file name
		traceLogsFileName = "minio.trace." + strconv.Itoa(int(traceLogsStartTime.Unix())) + ".log"
	}
	// Create hierarchy of directories if they do not exist
	if e := os.MkdirAll(traceLogsDirectoryName, 0o755); e != nil {
		fatalIf(probe.NewError(e), "Unable to create trace log file directory `"+traceLogsDirectoryName+"`")
	}
	traceLogsFilePath = filepath.Join(traceLogsDirectoryName, traceLogsFileName)
	return traceLogsFilePath
}

// Calculate tracing options for command line flags
func tracingOpts(ctx *cli.Context, apis []string) (opts madmin.ServiceTraceOpts, e error) {
	opts.Threshold = ctx.Duration("response-duration")
	opts.OnlyErrors = ctx.Bool("errors")

	if ctx.Bool("all") {
		for _, fn := range traceCallTypes {
			fn(&opts)
		}
	}

	if len(apis) == 0 {
		// If api flag is not specified, then we will
		// trace only S3 requests by default.
		opts.S3 = true
		return
	}

	for _, api := range apis {
		fn, ok := traceCallTypes[api]
		if !ok {
			fn, ok = traceCallTypeAliases[api]
		}
		if !ok {
			return madmin.ServiceTraceOpts{}, fmt.Errorf("unknown call name: `%s`", api)
		}
		fn(&opts)
	}
	return
}

// mainAdminTrace - the entry function of trace command
func mainAdminTrace(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminTraceSyntax(ctx)

	verbose := ctx.Bool("verbose")
	aliasedURL := ctx.Args().Get(0)

	console.SetColor("Stat", color.New(color.FgYellow))

	console.SetColor("Request", color.New(color.FgCyan))
	console.SetColor("Method", color.New(color.Bold, color.FgWhite))
	console.SetColor("Host", color.New(color.Bold, color.FgGreen))
	console.SetColor("FuncName", color.New(color.Bold, color.FgGreen))

	console.SetColor("ReqHeaderKey", color.New(color.Bold, color.FgWhite))
	console.SetColor("RespHeaderKey", color.New(color.Bold, color.FgCyan))
	console.SetColor("HeaderValue", color.New(color.FgWhite))
	console.SetColor("RespStatus", color.New(color.Bold, color.FgYellow))
	console.SetColor("ErrStatus", color.New(color.Bold, color.FgRed))

	console.SetColor("Response", color.New(color.FgGreen))
	console.SetColor("Body", color.New(color.FgYellow))
	for _, c := range colors {
		console.SetColor(fmt.Sprintf("Node%d", c), color.New(c))
	}

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	opts, e := tracingOpts(ctx, ctx.StringSlice("call"))
	fatalIf(probe.NewError(e), "Unable to start tracing")

	mopts := matchingOpts(ctx)

	if ctx.String("rotate-trace-logs-size") != "" || ctx.String("rotate-trace-logs-frequency") != "" || ctx.String("rotate-trace-logs-pattern") != "" {
		console.SetColorOff()
		traceLogsEnabled = true
		traceLogsStartTime = time.Now()
		// If logging is based on frequency, ensure truncate of time to nearest interval
		if ctx.String("rotate-trace-logs-frequency") != "" {
			traceLogsStartTime = time.Now().Truncate(traceLogsIntervalType)
		}
		// Create new trace log file
		traceLogsFile = createTraceLogsFile()
	}

	// Start listening on all trace activity.
	traceCh := client.ServiceTrace(ctxt, opts)
	for traceInfo := range traceCh {
		if traceInfo.Err != nil {
			fatalIf(probe.NewError(traceInfo.Err), "Unable to listen to http trace")
		}
		if matchTrace(mopts, traceInfo) {
			printTrace(verbose, traceInfo)
		}
	}
	return nil
}

// Short trace record
type shortTraceMsg struct {
	Status     string        `json:"status"`
	Host       string        `json:"host"`
	Time       time.Time     `json:"time"`
	Client     string        `json:"client"`
	CallStats  *callStats    `json:"callStats,omitempty"`
	Duration   time.Duration `json:"duration"`
	FuncName   string        `json:"api"`
	Path       string        `json:"path"`
	Query      string        `json:"query"`
	StatusCode int           `json:"statusCode"`
	StatusMsg  string        `json:"statusMsg"`
	Type       string        `json:"type"`
	Error      string        `json:"error"`
	trcType    madmin.TraceType
}

type traceMessage struct {
	Status string `json:"status"`
	madmin.ServiceTraceInfo
}

type requestInfo struct {
	Time     time.Time         `json:"time"`
	Proto    string            `json:"proto"`
	Method   string            `json:"method"`
	Path     string            `json:"path,omitempty"`
	RawQuery string            `json:"rawQuery,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     string            `json:"body,omitempty"`
}

type responseInfo struct {
	Time       time.Time         `json:"time"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	StatusCode int               `json:"statusCode,omitempty"`
}

type callStats struct {
	Rx       int           `json:"rx"`
	Tx       int           `json:"tx"`
	Duration time.Duration `json:"duration"`
	Ttfb     time.Duration `json:"timeToFirstByte"`
}

type verboseTrace struct {
	Type string `json:"type"`

	NodeName     string                 `json:"host"`
	FuncName     string                 `json:"api"`
	Time         time.Time              `json:"time"`
	Duration     time.Duration          `json:"duration"`
	Path         string                 `json:"path"`
	Error        string                 `json:"error,omitempty"`
	Message      string                 `json:"message,omitempty"`
	RequestInfo  *requestInfo           `json:"request,omitempty"`
	ResponseInfo *responseInfo          `json:"response,omitempty"`
	CallStats    *callStats             `json:"callStats,omitempty"`
	HealResult   *madmin.HealResultItem `json:"healResult,omitempty"`

	trcType madmin.TraceType
}

// return a struct with minimal trace info.
func shortTrace(ti madmin.ServiceTraceInfo) shortTraceMsg {
	s := shortTraceMsg{}
	t := ti.Trace

	s.trcType = t.TraceType
	s.Type = t.TraceType.String()
	s.FuncName = t.FuncName
	s.Time = t.Time
	s.Path = t.Path
	s.Error = t.Error
	s.Host = t.NodeName
	s.Duration = t.Duration
	s.StatusMsg = t.Message

	switch t.TraceType {
	case madmin.TraceS3, madmin.TraceInternal:
		s.Query = t.HTTP.ReqInfo.RawQuery
		s.StatusCode = t.HTTP.RespInfo.StatusCode
		s.StatusMsg = http.StatusText(t.HTTP.RespInfo.StatusCode)
		s.Client = t.HTTP.ReqInfo.Client
		s.CallStats = &callStats{}
		s.CallStats.Duration = t.Duration
		s.CallStats.Rx = t.HTTP.CallStats.InputBytes
		s.CallStats.Tx = t.HTTP.CallStats.OutputBytes
	}
	return s
}

func (s shortTraceMsg) JSON() string {
	s.Status = "success"
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", " ")
	// Disable escaping special chars to display XML tags correctly
	enc.SetEscapeHTML(false)

	fatalIf(probe.NewError(enc.Encode(s)), "Unable to marshal into JSON.")
	return buf.String()
}

func (s shortTraceMsg) String() string {
	var hostStr string
	b := &strings.Builder{}

	if s.Host != "" {
		hostStr = colorizedNodeName(s.Host)
	}
	fmt.Fprintf(b, "%s ", s.Time.Local().Format(traceTimeFormat))

	switch s.trcType {
	case madmin.TraceS3, madmin.TraceInternal:
	case madmin.TraceBootstrap:
		fmt.Fprintf(b, "[%s] %s %s %s", console.Colorize("RespStatus", strings.ToUpper(s.trcType.String())), console.Colorize("FuncName", s.FuncName),
			hostStr,
			s.StatusMsg,
		)
		return b.String()
	default:
		if s.Error != "" {
			fmt.Fprintf(b, "[%s] %s %s %s err='%s' %2s", console.Colorize("RespStatus", strings.ToUpper(s.trcType.String())), console.Colorize("FuncName", s.FuncName),
				hostStr,
				s.Path,
				console.Colorize("ErrStatus", s.Error),
				console.Colorize("HeaderValue", s.Duration))
		} else {
			fmt.Fprintf(b, "[%s] %s %s %s %2s", console.Colorize("RespStatus", strings.ToUpper(s.trcType.String())), console.Colorize("FuncName", s.FuncName),
				hostStr,
				s.Path,
				console.Colorize("HeaderValue", s.Duration))
		}
		return b.String()
	}

	statusStr := fmt.Sprintf("%d %s", s.StatusCode, s.StatusMsg)
	if s.StatusCode >= http.StatusBadRequest {
		statusStr = console.Colorize("ErrStatus", statusStr)
	} else {
		statusStr = console.Colorize("RespStatus", statusStr)
	}

	fmt.Fprintf(b, "[%s] %s ", statusStr, console.Colorize("FuncName", s.FuncName))
	fmt.Fprintf(b, "%s%s", hostStr, s.Path)

	if s.Query != "" {
		fmt.Fprintf(b, "?%s ", s.Query)
	}
	fmt.Fprintf(b, " %s ", s.Client)

	spaces := 15 - len(s.Client)
	fmt.Fprintf(b, "%*s", spaces, " ")
	fmt.Fprint(b, console.Colorize("HeaderValue", fmt.Sprintf("  %2s", s.CallStats.Duration.Round(time.Microsecond))))
	spaces = 12 - len(fmt.Sprintf("%2s", s.CallStats.Duration.Round(time.Microsecond)))
	fmt.Fprintf(b, "%*s", spaces, " ")
	fmt.Fprint(b, console.Colorize("Stat", " ↑ "))
	fmt.Fprint(b, console.Colorize("HeaderValue", humanize.IBytes(uint64(s.CallStats.Rx))))
	fmt.Fprint(b, console.Colorize("Stat", " ↓ "))
	fmt.Fprint(b, console.Colorize("HeaderValue", humanize.IBytes(uint64(s.CallStats.Tx))))

	return b.String()
}

// colorize node name
func colorizedNodeName(nodeName string) string {
	nodeHash := fnv.New32a()
	nodeHash.Write([]byte(nodeName))
	nHashSum := nodeHash.Sum32()
	idx := nHashSum % uint32(len(colors))
	return console.Colorize(fmt.Sprintf("Node%d", colors[idx]), nodeName)
}

func (t traceMessage) JSON() string {
	trc := verboseTrace{
		trcType:    t.Trace.TraceType,
		Type:       t.Trace.TraceType.String(),
		NodeName:   t.Trace.NodeName,
		FuncName:   t.Trace.FuncName,
		Time:       t.Trace.Time,
		Duration:   t.Trace.Duration,
		Path:       t.Trace.Path,
		Error:      t.Trace.Error,
		HealResult: t.Trace.HealResult,
		Message:    t.Trace.Message,
	}

	if t.Trace.HTTP != nil {
		rq := t.Trace.HTTP.ReqInfo
		rs := t.Trace.HTTP.RespInfo

		var (
			rqHdrs  = make(map[string]string)
			rspHdrs = make(map[string]string)
		)
		for k, v := range rq.Headers {
			rqHdrs[k] = strings.Join(v, " ")
		}
		for k, v := range rs.Headers {
			rspHdrs[k] = strings.Join(v, " ")
		}

		trc.RequestInfo = &requestInfo{
			Time:     rq.Time,
			Proto:    rq.Proto,
			Method:   rq.Method,
			Path:     rq.Path,
			RawQuery: rq.RawQuery,
			Body:     string(rq.Body),
			Headers:  rqHdrs,
		}
		trc.ResponseInfo = &responseInfo{
			Time:       rs.Time,
			Body:       string(rs.Body),
			Headers:    rspHdrs,
			StatusCode: rs.StatusCode,
		}
		trc.CallStats = &callStats{
			Duration: t.Trace.Duration,
			Rx:       t.Trace.HTTP.CallStats.InputBytes,
			Tx:       t.Trace.HTTP.CallStats.OutputBytes,
			Ttfb:     t.Trace.HTTP.CallStats.TimeToFirstByte,
		}
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", " ")
	// Disable escaping special chars to display XML tags correctly
	enc.SetEscapeHTML(false)
	fatalIf(probe.NewError(enc.Encode(trc)), "Unable to marshal into JSON.")

	// strip off extra newline added by json encoder
	return strings.TrimSuffix(buf.String(), "\n")
}

func (t traceMessage) String() string {
	var nodeNameStr string
	b := &strings.Builder{}

	trc := t.Trace
	if trc.NodeName != "" {
		nodeNameStr = fmt.Sprintf("%s ", colorizedNodeName(trc.NodeName))
	}

	switch trc.TraceType {
	case madmin.TraceS3, madmin.TraceInternal:
		if trc.HTTP == nil {
			return ""
		}
	case madmin.TraceBootstrap:
		fmt.Fprintf(b, "%s %s [%s] %s", nodeNameStr, console.Colorize("Request", fmt.Sprintf("[%s %s]", strings.ToUpper(trc.TraceType.String()), trc.FuncName)), trc.Time.Local().Format(traceTimeFormat), trc.Message)
		return b.String()
	default:
		if trc.Error != "" {
			fmt.Fprintf(b, "%s %s [%s] %s err='%s' %s", nodeNameStr, console.Colorize("Request", fmt.Sprintf("[%s %s]", strings.ToUpper(trc.TraceType.String()), trc.FuncName)), trc.Time.Local().Format(traceTimeFormat), trc.Path, console.Colorize("ErrStatus", trc.Error), trc.Duration)
		} else {
			fmt.Fprintf(b, "%s %s [%s] %s %s", nodeNameStr, console.Colorize("Request", fmt.Sprintf("[%s %s]", strings.ToUpper(trc.TraceType.String()), trc.FuncName)), trc.Time.Local().Format(traceTimeFormat), trc.Path, trc.Duration)
		}
		return b.String()
	}

	ri := trc.HTTP.ReqInfo
	rs := trc.HTTP.RespInfo
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Request", fmt.Sprintf("[REQUEST %s] ", trc.FuncName)))
	fmt.Fprintf(b, "[%s] %s\n", ri.Time.Local().Format(traceTimeFormat), console.Colorize("Host", fmt.Sprintf("[Client IP: %s]", ri.Client)))
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Method", fmt.Sprintf("%s %s", ri.Method, ri.Path)))
	if ri.RawQuery != "" {
		fmt.Fprintf(b, "?%s", ri.RawQuery)
	}
	fmt.Fprint(b, "\n")
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Method", fmt.Sprintf("Proto: %s\n", ri.Proto)))
	host, ok := ri.Headers["Host"]
	if ok {
		delete(ri.Headers, "Host")
	}
	hostStr := strings.Join(host, "")
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Host", fmt.Sprintf("Host: %s\n", hostStr)))
	for k, v := range ri.Headers {
		fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("ReqHeaderKey",
			fmt.Sprintf("%s: ", k))+console.Colorize("HeaderValue", fmt.Sprintf("%s\n", strings.Join(v, ""))))
	}

	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Body", fmt.Sprintf("%s\n", string(ri.Body))))
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Response", "[RESPONSE] "))
	fmt.Fprintf(b, "[%s] ", rs.Time.Local().Format(traceTimeFormat))
	fmt.Fprint(b, console.Colorize("Stat", fmt.Sprintf("[ Duration %2s  ↑ %s  ↓ %s ]\n", trc.Duration.Round(time.Microsecond), humanize.IBytes(uint64(trc.HTTP.CallStats.InputBytes)), humanize.IBytes(uint64(trc.HTTP.CallStats.OutputBytes)))))

	statusStr := console.Colorize("RespStatus", fmt.Sprintf("%d %s", rs.StatusCode, http.StatusText(rs.StatusCode)))
	if rs.StatusCode != http.StatusOK {
		statusStr = console.Colorize("ErrStatus", fmt.Sprintf("%d %s", rs.StatusCode, http.StatusText(rs.StatusCode)))
	}
	fmt.Fprintf(b, "%s%s\n", nodeNameStr, statusStr)

	for k, v := range rs.Headers {
		fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("RespHeaderKey",
			fmt.Sprintf("%s: ", k))+console.Colorize("HeaderValue", fmt.Sprintf("%s\n", strings.Join(v, ","))))
	}
	fmt.Fprintf(b, "%s%s\n", nodeNameStr, console.Colorize("Body", string(rs.Body)))
	fmt.Fprint(b, nodeNameStr)
	return b.String()
}
