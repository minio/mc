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
	"fmt"
	"hash/fnv"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/mc/pkg/colorjson"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
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
}

var adminTraceCmd = cli.Command{
	Name:            "trace",
	Usage:           "show http trace for minio server",
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
  1. Show console trace for a Minio server with alias 'play'
     $ {{.HelpName}} play -v -a
 `,
}

const timeFormat = "15:04:05.000000000"

var (
	funcNameRegex = regexp.MustCompile(`^.*?\\.([^\\-]*?)Handler\\-.*?$`)
	colors        = []color.Attribute{color.FgCyan, color.FgWhite, color.FgYellow, color.FgGreen}
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
	aliasedURL := ctx.Args().Get(0)
	console.SetColor("Request", color.New(color.FgCyan))
	console.SetColor("Method", color.New(color.Bold, color.FgWhite))
	console.SetColor("Host", color.New(color.Bold, color.FgGreen))
	console.SetColor("FuncName", color.New(color.Bold, color.FgGreen))

	console.SetColor("ReqHeaderKey", color.New(color.Bold, color.FgWhite))
	console.SetColor("RespHeaderKey", color.New(color.Bold, color.FgCyan))
	console.SetColor("HeaderValue", color.New(color.FgWhite))
	console.SetColor("ResponseStatus", color.New(color.Bold, color.FgYellow))

	console.SetColor("Response", color.New(color.FgGreen))
	console.SetColor("Body", color.New(color.FgYellow))
	for _, c := range colors {
		console.SetColor(fmt.Sprintf("Node%d", c), color.New(c))
	}
	// Create a new Minio Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Cannot initialize admin client.")
		return nil
	}
	doneCh := make(chan struct{})
	defer close(doneCh)

	// Start listening on all trace activity.
	traceCh := client.Trace(all, doneCh)
	for traceInfo := range traceCh {
		if traceInfo.Err != nil {
			fatalIf(probe.NewError(traceInfo.Err), "Cannot listen to http trace")
		}
		if verbose {
			printMsg(traceMessage{traceInfo})
			continue
		}
		printMsg(shortTrace(traceInfo))
	}
	return nil
}

// Short trace record
type shortTraceMsg struct {
	NodeName   string
	Time       time.Time
	FuncName   string
	Host       string
	Path       string
	Query      string
	StatusCode int
	StatusMsg  string
}

type traceMessage struct {
	madmin.TraceInfo
}

type requestInfo struct {
	Time     time.Time         `json:"time"`
	Method   string            `json:"method"`
	Path     string            `json:"path,omitempty"`
	RawQuery string            `json:"rawquery,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     string            `json:"body,omitempty"`
}

type responseInfo struct {
	Time       time.Time         `json:"time"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body,omitempty"`
	StatusCode int               `json:"statuscode,omitempty"`
}

type trace struct {
	NodeName     string       `json:"nodename"`
	FuncName     string       `json:"funcname"`
	RequestInfo  requestInfo  `json:"request"`
	ResponseInfo responseInfo `json:"response"`
}

// parse Operation name from function name
func getOpName(fname string) (op string) {
	res := funcNameRegex.FindStringSubmatch(fname)
	op = fname
	if len(res) == 2 {
		op = res[1]
	}
	return
}

// return a struct with minimal trace info.
func shortTrace(ti madmin.TraceInfo) shortTraceMsg {
	s := shortTraceMsg{}
	t := ti.Trace
	s.NodeName = t.NodeName
	s.Time = t.ReqInfo.Time
	if host, ok := t.ReqInfo.Headers["Host"]; ok {
		s.Host = strings.Join(host, "")
	}
	s.Path = t.ReqInfo.Path
	s.Query = t.ReqInfo.RawQuery
	s.FuncName = getOpName(t.FuncName)
	s.StatusCode = t.RespInfo.StatusCode
	s.StatusMsg = http.StatusText(t.RespInfo.StatusCode)

	return s
}

func (s shortTraceMsg) JSON() string {
	traceJSONBytes, e := json.MarshalIndent(s, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")
	return string(traceJSONBytes)
}

func (s shortTraceMsg) String() string {
	var hostStr string
	var b = &strings.Builder{}

	if s.Host != "" {
		hostStr = colorizedNodeName(s.Host)
	}
	fmt.Fprintf(b, "%s %s ", s.Time.Format(timeFormat), console.Colorize("FuncName", s.FuncName))
	fmt.Fprintf(b, "%s%s", hostStr, s.Path)

	if s.Query != "" {
		fmt.Fprintf(b, "?%s", s.Query)
	}
	fmt.Fprintf(b, " %s", console.Colorize("ResponseStatus",
		fmt.Sprintf("\t%s %s", strconv.Itoa(s.StatusCode), s.StatusMsg)))
	return b.String()
}

// colorize node name
func colorizedNodeName(nodeName string) string {
	nodeHash := fnv.New32a()
	nodeHash.Write([]byte(nodeName))
	nHashSum := nodeHash.Sum32()
	idx := int(nHashSum) % len(colors)
	return console.Colorize(fmt.Sprintf("Node%d", colors[idx]), nodeName)
}

func (t traceMessage) JSON() string {
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
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", " ")
	// Disable escaping special chars to display XML tags correctly
	enc.SetEscapeHTML(false)
	enc.Encode(trc)
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

	fmt.Fprintf(b, "[%s]\n", ri.Time.Format(timeFormat))
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Method", fmt.Sprintf("%s %s", ri.Method, ri.Path)))
	if ri.RawQuery != "" {
		fmt.Fprintf(b, "?%s", ri.RawQuery)
	}
	fmt.Fprint(b, "\n")
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
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("Response", fmt.Sprintf("[RESPONSE] ")))
	fmt.Fprintf(b, "[%s]\n", rs.Time.Format(timeFormat))
	fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("ResponseStatus",
		fmt.Sprintf("%d %s\n", rs.StatusCode, http.StatusText(rs.StatusCode))))
	for k, v := range rs.Headers {
		fmt.Fprintf(b, "%s%s", nodeNameStr, console.Colorize("RespHeaderKey",
			fmt.Sprintf("%s: ", k))+console.Colorize("HeaderValue", fmt.Sprintf("%s\n", strings.Join(v, ""))))
	}
	fmt.Fprintf(b, "%s%s\n", nodeNameStr, console.Colorize("Body", string(rs.Body)))
	fmt.Fprint(b, nodeNameStr)
	return b.String()
}
