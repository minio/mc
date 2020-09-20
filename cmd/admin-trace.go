/*
 * MinIO Client (C) 2019 MinIO, Inc.
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
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/console"
	"github.com/minio/minio/pkg/madmin"
)

var adminTraceFlags = []cli.Flag{
	cli.BoolFlag{
		Name:  "verbose, v",
		Usage: "print verbose trace",
	},
	cli.BoolFlag{
		Name:  "all, a",
		Usage: "trace all traffic (including internode traffic between MinIO servers)",
	},
	cli.BoolFlag{
		Name:  "errors, e",
		Usage: "trace failed requests only",
	},
}

var adminTraceCmd = cli.Command{
	Name:            "trace",
	Usage:           "show http trace for MinIO server",
	Action:          mainAdminTrace,
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
  1. Show console trace for a MinIO server with alias 'play'
     {{.Prompt}} {{.HelpName}} -v -a play

  2. Show trace only for failed requests for a MinIO server with alias 'myminio'
    {{.Prompt}} {{.HelpName}} -v -e myminio
`,
}

const timeFormat = "15:04:05.000"

var (
	colors = []color.Attribute{color.FgCyan, color.FgWhite, color.FgYellow, color.FgGreen}
)

func checkAdminTraceSyntax(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		cli.ShowCommandHelpAndExit(ctx, "trace", 1) // last argument is exit code
	}
}

// mainAdminTrace - the entry function of trace command
func mainAdminTrace(ctx *cli.Context) error {
	// Check for command syntax
	checkAdminTraceSyntax(ctx)
	verbose := ctx.Bool("verbose")
	all := ctx.Bool("all")
	errfltr := ctx.Bool("errors")
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

	// Start listening on all trace activity.
	traceCh := client.ServiceTrace(ctxt, all, errfltr)
	for traceInfo := range traceCh {
		if traceInfo.Err != nil {
			fatalIf(probe.NewError(traceInfo.Err), "Unable to listen to http trace")
		}
		if verbose {
			printMsg(traceMessage{ServiceTraceInfo: traceInfo})
			continue
		}
		printMsg(shortTrace(traceInfo))
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

type trace struct {
	NodeName     string       `json:"host"`
	FuncName     string       `json:"api"`
	RequestInfo  requestInfo  `json:"request"`
	ResponseInfo responseInfo `json:"response"`
	CallStats    callStats    `json:"callStats"`
}

// return a struct with minimal trace info.
func shortTrace(ti madmin.ServiceTraceInfo) shortTraceMsg {
	s := shortTraceMsg{}
	t := ti.Trace
	s.Time = t.ReqInfo.Time
	if host, ok := t.ReqInfo.Headers["Host"]; ok {
		s.Host = strings.Join(host, "")
	}
	s.Path = t.ReqInfo.Path
	s.Query = t.ReqInfo.RawQuery
	s.FuncName = t.FuncName
	s.StatusCode = t.RespInfo.StatusCode
	s.StatusMsg = http.StatusText(t.RespInfo.StatusCode)
	cSlice := strings.Split(t.ReqInfo.Client, ":")
	s.Client = cSlice[0]
	s.CallStats.Duration = t.CallStats.Latency
	s.CallStats.Rx = t.CallStats.InputBytes
	s.CallStats.Tx = t.CallStats.OutputBytes
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
	statusStr := console.Colorize("RespStatus", fmt.Sprintf("%d %s", s.StatusCode, s.StatusMsg))
	if s.StatusCode >= http.StatusBadRequest {
		statusStr = console.Colorize("ErrStatus", fmt.Sprintf("%d %s", s.StatusCode, s.StatusMsg))
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
	trc := trace{
		NodeName: t.Trace.NodeName,
		FuncName: t.Trace.FuncName,
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

	ri := trc.ReqInfo
	rs := trc.RespInfo
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Request", fmt.Sprintf("[REQUEST %s] ", trc.FuncName)))
	fmt.Fprintf(b, "%s\n", ri.Time.Format(timeFormat))
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
