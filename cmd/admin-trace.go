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
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"path"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
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
		Usage: "trace only matching Call types (values: `s3`, `internal`, `storage`, `os`)",
	},
	cli.StringFlag{
		Name:  "response-threshold",
		Usage: "trace calls only with response duration greater than this threshold (e.g. `5ms`)",
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
	cli.BoolFlag{
		Name:  "errors, e",
		Usage: "trace only failed requests",
	},
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
EXAMPLES:
  1. Show verbose console trace for MinIO server
     {{.Prompt}} {{.HelpName}} -v -a myminio

  2. Show trace only for failed requests for MinIO server
    {{.Prompt}} {{.HelpName}} -v -e myminio

  3. Show verbose console trace for requests with '503' status code
    {{.Prompt}} {{.HelpName}} -v --status-code 503 myminio

  4. Show console trace for a specific path
    {{.Prompt}} {{.HelpName}} --path my-bucket/my-prefix/ myminio

  5. Show console trace for requests with '404' and '503' status code
    {{.Prompt}} {{.HelpName}} --status-code 404 --status-code 503 myminio
`,
}

const timeFormat = "2006-01-02T15:04:05:000"

var (
	colors = []color.Attribute{color.FgCyan, color.FgWhite, color.FgYellow, color.FgGreen}
)

func checkAdminTraceSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "trace", 1) // last argument is exit code
	}
}

func printTrace(verbose bool, traceInfo madmin.ServiceTraceInfo) {
	if verbose {
		printMsg(traceMessage{ServiceTraceInfo: traceInfo})
	} else {
		printMsg(shortTrace(traceInfo))
	}
}

func matchTrace(ctx *cli.Context, traceInfo madmin.ServiceTraceInfo) bool {
	statusCodes := ctx.IntSlice("status-code")
	methods := ctx.StringSlice("method")
	funcNames := ctx.StringSlice("funcname")
	apiPaths := ctx.StringSlice("path")

	if len(statusCodes) == 0 && len(methods) == 0 && len(funcNames) == 0 && len(apiPaths) == 0 {
		// no specific filtering found trace all the requests
		return true
	}

	// Filter request path if passed by the user
	if len(apiPaths) > 0 {
		matched := false
		for _, apiPath := range apiPaths {
			if pathMatch(path.Join("/", apiPath), traceInfo.Trace.ReqInfo.Path) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Filter response status codes if passed by the user
	if len(statusCodes) > 0 {
		matched := false
		for _, code := range statusCodes {
			if traceInfo.Trace.RespInfo.StatusCode == code {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}

	}

	// Filter request method if passed by the user
	if len(methods) > 0 {
		matched := false
		for _, method := range methods {
			if traceInfo.Trace.ReqInfo.Method == method {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}

	}

	if len(funcNames) > 0 {
		matched := false
		// Filter request function handler names if passed by the user.
		for _, funcName := range funcNames {
			if nameMatch(funcName, traceInfo.Trace.FuncName) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// Calculate tracing options for command line flags
func tracingOpts(ctx *cli.Context) (opts madmin.ServiceTraceOpts, e error) {

	if t := ctx.String("response-threshold"); t != "" {
		d, e := time.ParseDuration(t)
		if e != nil {
			return opts, fmt.Errorf("Unable to parse threshold argument: %w", e)
		}
		opts.Threshold = d
	}

	opts.OnlyErrors = ctx.Bool("errors")

	if ctx.Bool("all") {
		opts.All = true // Deprecated

		opts.S3 = true
		opts.Internal = true
		opts.Storage = true
		opts.OS = true
		return
	}

	apis := ctx.StringSlice("call")
	if len(apis) == 0 {
		// If api flag is not specified, then we will
		// trace only S3 requests by default.
		opts.S3 = true
		return
	}

	for _, api := range apis {
		switch api {
		case "storage":
			opts.Storage = true
		case "internal":
			opts.Internal = true
		case "s3":
			opts.S3 = true
		case "os":
			opts.OS = true
		}
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

	opts, e := tracingOpts(ctx)
	fatalIf(probe.NewError(e), "Unable to start tracing")

	// Start listening on all trace activity.
	traceCh := client.ServiceTrace(ctxt, opts)
	for traceInfo := range traceCh {
		if traceInfo.Err != nil {
			fatalIf(probe.NewError(traceInfo.Err), "Unable to listen to http trace")
		}
		if matchTrace(ctx, traceInfo) {
			printTrace(verbose, traceInfo)
		}
	}
	return nil
}

// Short trace record
type shortTraceMsg struct {
	Status     string    `json:"status"`
	Host       string    `json:"host"`
	Time       time.Time `json:"time"`
	Client     string    `json:"client"`
	CallStats  callStats `json:"callStats"`
	FuncName   string    `json:"api"`
	Path       string    `json:"path"`
	Query      string    `json:"query"`
	StatusCode int       `json:"statusCode"`
	StatusMsg  string    `json:"statusMsg"`

	StorageStats storageStats     `json:"storageStats"`
	OSStats      osStats          `json:"osStats"`
	Type         madmin.TraceType `json:"type"`
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

type osStats struct {
	Duration time.Duration `json:"duration"`
	Path     string        `json:"path"`
}

type storageStats struct {
	Duration time.Duration `json:"duration"`
	Path     string        `json:"path"`
}

type verboseTrace struct {
	Type madmin.TraceType `json:"type"`

	NodeName string    `json:"host"`
	FuncName string    `json:"api"`
	Time     time.Time `json:"time"`

	RequestInfo  requestInfo  `json:"request"`
	ResponseInfo responseInfo `json:"response"`
	CallStats    callStats    `json:"callStats"`

	StorageStats storageStats `json:"storageStats"`
	OSStats      osStats      `json:"osStats"`
}

// return a struct with minimal trace info.
func shortTrace(ti madmin.ServiceTraceInfo) shortTraceMsg {
	s := shortTraceMsg{}
	t := ti.Trace

	s.Type = t.TraceType
	s.FuncName = t.FuncName
	s.Time = t.Time

	switch t.TraceType {
	case madmin.TraceStorage:
		s.Path = t.StorageStats.Path
		s.Host = t.NodeName
		s.StorageStats.Duration = t.StorageStats.Duration
	case madmin.TraceOS:
		s.Path = t.OSStats.Path
		s.Host = t.NodeName
		s.OSStats.Duration = t.OSStats.Duration
	case madmin.TraceHTTP:
		s.Host = t.NodeName
		s.Path = t.ReqInfo.Path
		s.Query = t.ReqInfo.RawQuery
		s.StatusCode = t.RespInfo.StatusCode
		s.StatusMsg = http.StatusText(t.RespInfo.StatusCode)
		s.Client = t.ReqInfo.Client
		s.CallStats.Duration = t.CallStats.Latency
		s.CallStats.Rx = t.CallStats.InputBytes
		s.CallStats.Tx = t.CallStats.OutputBytes

		// Support older versions of MinIO server
		s.Time = t.ReqInfo.Time
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
	var b = &strings.Builder{}

	if s.Host != "" {
		hostStr = colorizedNodeName(s.Host)
	}
	fmt.Fprintf(b, "%s ", s.Time.Format(timeFormat))

	switch s.Type {
	case madmin.TraceStorage:
		fmt.Fprintf(b, "[%s] %s %s %s %2s", console.Colorize("RespStatus", "STORAGE"), console.Colorize("FuncName", s.FuncName),
			hostStr,
			s.Path,
			console.Colorize("HeaderValue", s.StorageStats.Duration))
		return b.String()
	case madmin.TraceOS:
		fmt.Fprintf(b, "[%s] %s %s %s %2s", console.Colorize("RespStatus", "OS"), console.Colorize("FuncName", s.FuncName),
			hostStr,
			s.Path,
			console.Colorize("HeaderValue", s.OSStats.Duration))
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
	idx := uint32(nHashSum) % uint32(len(colors))
	return console.Colorize(fmt.Sprintf("Node%d", colors[idx]), nodeName)
}

func (t traceMessage) JSON() string {
	t.Status = "success"
	rqHdrs := make(map[string]string)
	rspHdrs := make(map[string]string)
	rq := t.Trace.ReqInfo
	rs := t.Trace.RespInfo
	for k, v := range rq.Headers {
		rqHdrs[k] = strings.Join(v, " ")
	}
	for k, v := range rs.Headers {
		rspHdrs[k] = strings.Join(v, " ")
	}
	trc := verboseTrace{
		Type:     t.Trace.TraceType,
		NodeName: t.Trace.NodeName,
		FuncName: t.Trace.FuncName,
		Time:     t.Trace.Time,
		RequestInfo: requestInfo{
			Time:     rq.Time,
			Proto:    rq.Proto,
			Method:   rq.Method,
			Path:     rq.Path,
			RawQuery: rq.RawQuery,
			Body:     string(rq.Body),
			Headers:  rqHdrs,
		},
		ResponseInfo: responseInfo{
			Time:       rs.Time,
			Body:       string(rs.Body),
			Headers:    rspHdrs,
			StatusCode: rs.StatusCode,
		},
		CallStats: callStats{
			Duration: t.Trace.CallStats.Latency,
			Rx:       t.Trace.CallStats.InputBytes,
			Tx:       t.Trace.CallStats.OutputBytes,
			Ttfb:     t.Trace.CallStats.TimeToFirstByte,
		},
		OSStats: osStats{
			Duration: t.Trace.OSStats.Duration,
			Path:     t.Trace.OSStats.Path,
		},
		StorageStats: storageStats{
			Duration: t.Trace.StorageStats.Duration,
			Path:     t.Trace.StorageStats.Path,
		},
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
	var b = &strings.Builder{}

	trc := t.Trace
	if trc.NodeName != "" {
		nodeNameStr = fmt.Sprintf("%s ", colorizedNodeName(trc.NodeName))
	}

	switch trc.TraceType {
	case madmin.TraceStorage:
		fmt.Fprintf(b, "%s %s [%s] %s %s", nodeNameStr, console.Colorize("Request", fmt.Sprintf("[STORAGE %s]", trc.FuncName)), trc.Time.Format(timeFormat), trc.StorageStats.Path, trc.StorageStats.Duration)
		return b.String()
	case madmin.TraceOS:
		fmt.Fprintf(b, "%s %s [%s] %s %s", nodeNameStr, console.Colorize("Request", fmt.Sprintf("[OS %s]", trc.FuncName)), trc.Time.Format(timeFormat), trc.OSStats.Path, trc.OSStats.Duration)
		return b.String()
	}

	ri := trc.ReqInfo
	rs := trc.RespInfo
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Request", fmt.Sprintf("[REQUEST %s] ", trc.FuncName)))
	fmt.Fprintf(b, "[%s] %s\n", ri.Time.Format(timeFormat), console.Colorize("Host", fmt.Sprintf("[Client IP: %s]", ri.Client)))
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
	fmt.Fprintf(b, "[%s] ", rs.Time.Format(timeFormat))
	fmt.Fprint(b, console.Colorize("Stat", fmt.Sprintf("[ Duration %2s  ↑ %s  ↓ %s ]\n", trc.CallStats.Latency.Round(time.Microsecond), humanize.IBytes(uint64(trc.CallStats.InputBytes)), humanize.IBytes(uint64(trc.CallStats.OutputBytes)))))

	statusStr := console.Colorize("RespStatus", fmt.Sprintf("%d %s", rs.StatusCode, http.StatusText(rs.StatusCode)))
	if rs.StatusCode != http.StatusOK {
		statusStr = console.Colorize("ErrStatus", fmt.Sprintf("%d %s", rs.StatusCode, http.StatusText(rs.StatusCode)))
	}
	fmt.Fprintf(b, "%s%s\n", nodeNameStr, statusStr)

	for k, v := range rs.Headers {
		fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("RespHeaderKey",
			fmt.Sprintf("%s: ", k))+console.Colorize("HeaderValue", fmt.Sprintf("%s\n", strings.Join(v, ""))))
	}
	fmt.Fprintf(b, "%s%s\n", nodeNameStr, console.Colorize("Body", string(rs.Body)))
	fmt.Fprint(b, nodeNameStr)
	return b.String()
}
