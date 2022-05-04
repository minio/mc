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
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize/english"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

const (
	livelinessEndPoint = "/minio/health/live"
	pingInterval       = 3 * time.Second
)

var pingFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "count, c",
		Usage: "will return liveliness for that count number of times and return.",
		Value: 4,
	},
}

// return latency and liveness probe.
var pingCmd = cli.Command{
	Name:            "ping",
	Usage:           "will return latency and liveness probe",
	Action:          mainPing,
	Before:          setGlobalsFromContext,
	OnUsageError:    onUsageError,
	Flags:           append(pingFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET [TARGET...]
{{if .VisibleFlags}}
FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}{{end}}
EXAMPLES:
  1. Return Latency and liveness probe.
     {{.Prompt}} {{.HelpName}} myminio

  2. Return Latency and liveness probe 5 number of times.
     {{.Prompt}} {{.HelpName}} --count 5 myminio
`,
}

// Validate command line arguments.
func checkPingSyntax(cliCtx *cli.Context) {
	if !cliCtx.Args().Present() {
		cli.ShowCommandHelpAndExit(cliCtx, "ping", 1) // last argument is exit code
	}
}

// PingResult is a slice of Ping
type PingResult struct {
	Pings []Ping `json:"ping,omitempty"`
}

// Ping wraps  status, error and latency
type Ping struct {
	Server  string `json:"server"`
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency,omitempty"`
}

func computeLatency(start time.Time) string {
	diff := time.Since(start)
	hours := int(diff.Hours())
	minutes := int(diff.Minutes()) % 60
	seconds := int(diff.Seconds()) % 60
	milliSeconds := int(diff.Milliseconds())
	microSeconds := int(diff.Microseconds())
	nanoSeconds := int(diff.Nanoseconds())
	switch {
	case hours > 0:
		return fmt.Sprintf("%s hour", strconv.Itoa(hours))
	case minutes > 0:
		return fmt.Sprintf("%s min", strconv.Itoa(minutes))
	case seconds > 0:
		return fmt.Sprintf("%s sec", strconv.Itoa(seconds))
	case milliSeconds > 0:
		return fmt.Sprintf("%s ms", strconv.Itoa(milliSeconds))
	case microSeconds > 0:
		return fmt.Sprintf("%s μs", strconv.Itoa(microSeconds))
	default:
		return english.Plural(nanoSeconds, " ns", "")
	}
}

func getServer(admInfo madmin.InfoMessage, endPoint string) string {
	var domain string
	server := endPoint
	if len(admInfo.Domain) > 0 {
		domain = admInfo.Domain[0]
	}
	if domain != "" {
		if len(strings.Split(endPoint, ":")) > 1 {
			server = domain + ":" + strings.Split(endPoint, ":")[1]
		} else {
			server = domain
		}
	}
	return server
}

// JSON jsonified ping result message.
func (pr PingResult) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(pr, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// String colorized ping result message.
func (pr PingResult) String() (msg string) {
	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	console.SetColor("InfoFail", color.New(color.FgRed, color.Bold))
	for _, ping := range pr.Pings {
		if ping.Error == "" {
			coloredDot := console.Colorize("Info", dot)
			// Print server title
			msg += fmt.Sprintf("%s  %s", coloredDot, console.Colorize("PrintB", ping.Server))
			msg += fmt.Sprintf(" => %s\n", ping.Latency)
		} else {
			coloredDot := console.Colorize("InfoFail", dot)
			msg += fmt.Sprintf("%s  %s", coloredDot, console.Colorize("PrintB", ping.Server))
			msg += fmt.Sprintf(" => Error: %s\n", console.Colorize("InfoFail", ping.Error))
		}
	}
	return msg
}

func fetchAdminInfo(alias, url string) madmin.InfoMessage {
	// Create a new MinIO Admin Client
	client, err := newAdminClient(alias)
	fatalIf(err, "Unable to initialize admin connection.")

	// Fetch info of all servers (cluster or single server)
	admInfo, er := client.ServerInfo(globalContext)
	// Keeps on retring until the server is up
	for er != nil {
		var ping Ping
		var pings []Ping
		ping.Server = url
		ping.Error = er.Error()
		pings = append(pings, ping)
		time.Sleep(pingInterval)
		printMsg(PingResult{Pings: pings})
		admInfo, er = client.ServerInfo(globalContext)
	}
	return admInfo
}

// mainPing is entry point for ping command.
func mainPing(ctx *cli.Context) error {
	// check 'ping' cli arguments.
	checkPingSyntax(ctx)

	// Get the alias parameter from cli
	args := ctx.Args()
	alias := cleanAlias(args.Get(0))
	hostConfig := mustGetHostConfig(alias)
	if hostConfig == nil {
		fatalIf(errInvalidAliasedURL(alias), "No such alias `"+alias+"` found.")
		return nil
	}

	count := ctx.Int("count")
	if count < 4 {
		fatalIf(errInvalidArgument().Trace(ctx.Args()...), "please set count value more than 4")
	}

	httpClient := httpClient(10 * time.Second)

	u, e := url.Parse(hostConfig.URL)
	fatalIf(probe.NewError(e), "Unable to parse url")
	admInfo := fetchAdminInfo(alias, hostConfig.URL)
	for i := 0; i < count; i++ {
		var pings []Ping
		for _, server := range admInfo.Servers {
			var ping Ping
			req, e := http.NewRequest(http.MethodGet, u.Scheme+"://"+server.Endpoint+livelinessEndPoint, nil)
			if e != nil {
				return e
			}
			start := time.Now()
			resp, e := httpClient.Do(req)
			latency := computeLatency(start)
			ping.Server = getServer(admInfo, server.Endpoint)
			if e != nil {
				ping.Error = e.Error()
			}

			if resp != nil {
				switch resp.StatusCode {
				case http.StatusOK:
					ping.Latency = latency
					ping.Error = ""
					defer resp.Body.Close()
				default:
					ping.Error = resp.Status
				}
			}
			pings = append(pings, ping)
		}
		time.Sleep(pingInterval)
		printMsg(PingResult{Pings: pings})
	}
	return nil
}
