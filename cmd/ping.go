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
	"context"
	"fmt"
	"math"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

const (
	pingInterval = time.Second // keep it similar to unix ping interval
)

var pingFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "count, c",
		Usage: "perform liveliness check for count number of times",
		// Value: 4,
	},
	cli.IntFlag{
		Name:  "error-count, e",
		Usage: "exit if errors more than consecutive error count",
		Value: 50,
	},
	cli.IntFlag{
		Name:  "interval, i",
		Usage: "wait interval between each request in seconds",
		Value: 1,
	},
	cli.BoolFlag{
		Name:  "distributed, a",
		Usage: "ping all the servers in the cluster",
	},
}

// return latency and liveness probe.
var pingCmd = cli.Command{
	Name:            "ping",
	Usage:           "perform liveness check",
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

  3. Return Latency and liveness with wait interval set to 30 seconds.
     {{.Prompt}} {{.HelpName}} --interval 30 myminio

  4. Stop pinging when error count > 20.
  	 {{.Prompt}} {{.HelpName}} --error-count 20 myminio	 
`,
}

var stop bool

// Validate command line arguments.
func checkPingSyntax(cliCtx *cli.Context) {
	if !cliCtx.Args().Present() {
		cli.ShowCommandHelpAndExit(cliCtx, "ping", 1) // last argument is exit code
	}
}

// JSON jsonified ping result message.
func (pr PingResult) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(pr, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

func (pr PingResult) String() (msg string) {
	var s strings.Builder
	w := tabwriter.NewWriter(&s, 0, 8, 1, '\t', tabwriter.AlignRight)
	var ep string
	for _, p := range pr.Endpoints {
		if p.Endpoint.Port == "" {
			ep = p.Endpoint.Scheme + "://" + p.Endpoint.Host
		} else {
			ep = p.Endpoint.Scheme + "://" + p.Endpoint.Host + ":" + p.Endpoint.Port
		}
		if p.Error == "" {
			fmt.Fprintf(&s, "%d:  %s\t", pr.Counter, ep)
			fmt.Fprintf(&s, "   min=%s\t   max=%s\t   average=%s\t   errors=%d\t    roundtrip=%s\t\n", p.Min, p.Max, p.Average, p.CountErr, p.Roundtrip)
		} else {
			fmt.Fprintf(&s, console.Colorize("InfoFail", fmt.Sprintf("%d:  %s\t   min=%s\t   max=%s\t   average=%s\t   errors=%d\t    roundtrip=%s\t\n",
				pr.Counter, ep, p.Min, p.Max, p.Average, p.CountErr, p.Roundtrip)))
		}
	}
	w.Flush()
	return s.String()
}

// Endpoint - container to hold server info
type Endpoint struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Port   string `json:"port"`
}

// EndPointStats - container to hold server ping stats
type EndPointStats struct {
	Endpoint  Endpoint      `json:"endpoint"`
	Min       time.Duration `json:"min"`
	Max       time.Duration `json:"max"`
	Average   time.Duration `json:"average"`
	CountErr  int           `json:"error-count,omitempty"`
	Error     string        `json:"error,omitempty"`
	Roundtrip time.Duration `json:"roundtrip"`
}

// PingResult contains ping output
type PingResult struct {
	Status    string          `json:"status"`
	Counter   int             `json:"counter"`
	Endpoints []EndPointStats `json:"servers"`
}

type serverStats struct {
	min        uint64
	max        uint64
	sum        uint64
	avg        uint64
	errorCount int // used to keep a track of consecutive errors
	err        string
	counter    int // used to find the average, acts as denominator
}

func fetchAdminInfo(admClnt *madmin.AdminClient) (madmin.InfoMessage, error) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-globalContext.Done():
			return madmin.InfoMessage{}, globalContext.Err()
		case <-timer.C:
			ctx, cancel := context.WithTimeout(globalContext, 3*time.Second)
			// Fetch the service status of the specified MinIO server
			info, e := admClnt.ServerInfo(ctx)
			cancel()
			if e == nil {
				return info, nil
			}
			timer.Reset(time.Second)
		}
	}
}

func ping(ctx context.Context, cliCtx *cli.Context, anonClient *madmin.AnonymousClient, admInfo madmin.InfoMessage, endPointMap map[string]serverStats, index int) {
	var endPointStats []EndPointStats
	var servers []madmin.ServerProperties
	if cliCtx.Bool("distributed") {
		servers = admInfo.Servers
	} else {
		servers = append(servers, admInfo.Servers[0])
	}

	for result := range anonClient.Alive(ctx, madmin.AliveOpts{}, servers...) {
		endPoint := getEndPoint(result)
		stat := populateData(cliCtx, result, endPointMap)
		endPointStat := EndPointStats{
			Endpoint:  endPoint,
			Min:       time.Duration(stat.min).Round(time.Microsecond),
			Max:       time.Duration(stat.max).Round(time.Microsecond),
			Average:   time.Duration(stat.avg).Round(time.Microsecond),
			CountErr:  stat.errorCount,
			Error:     stat.err,
			Roundtrip: result.ResponseTime.Round(time.Microsecond),
		}
		endPointStats = append(endPointStats, endPointStat)
		endPointMap[result.Endpoint.Host] = stat

	}
	printMsg(PingResult{
		Status:    "success",
		Counter:   index,
		Endpoints: endPointStats,
	})

	time.Sleep(time.Duration(cliCtx.Int("interval")) * pingInterval)
}

func populateData(cliCtx *cli.Context, result madmin.AliveResult, serverMap map[string]serverStats) serverStats {
	ec := cliCtx.Int("error-count")
	var errorString string
	var sum, avg uint64
	min := uint64(math.MaxUint64)
	var max uint64
	var counter, errorCount int

	if result.Error != nil {
		errorString = result.Error.Error()
		if stat, ok := serverMap[result.Endpoint.Host]; ok {
			min = stat.min
			max = stat.max
			sum = stat.sum
			counter = stat.counter
			avg = stat.avg
			errorCount = stat.errorCount + 1

		} else {
			min = 0
			errorCount = 1
		}
		if errorCount >= ec {
			stop = true
		}
	} else {
		// reset consecutive error count
		errorCount = 0
		if stat, ok := serverMap[result.Endpoint.Host]; ok {
			var minVal uint64
			if stat.min == 0 {
				minVal = uint64(result.ResponseTime)
			} else {
				minVal = stat.min
			}
			min = uint64(math.Min(float64(minVal), float64(uint64(result.ResponseTime))))
			max = uint64(math.Max(float64(stat.max), float64(uint64(result.ResponseTime))))
			sum = stat.sum + uint64(result.ResponseTime.Nanoseconds())
			counter = stat.counter + 1

		} else {
			min = uint64(math.Min(float64(min), float64(uint64(result.ResponseTime))))
			max = uint64(math.Max(float64(max), float64(uint64(result.ResponseTime))))
			sum = uint64(result.ResponseTime)
			counter = 1
		}
		avg = sum / uint64(counter)
	}
	return serverStats{min, max, sum, avg, errorCount, errorString, counter}
}

func getEndPoint(result madmin.AliveResult) Endpoint {
	address := strings.Split(result.Endpoint.Host, ":")
	port := ""
	if len(address) > 1 {
		port = address[1]
	}
	return Endpoint{Scheme: result.Endpoint.Scheme, Host: address[0], Port: port}
}

// mainPing is entry point for ping command.
func mainPing(cliCtx *cli.Context) error {
	// check 'ping' cli arguments.
	checkPingSyntax(cliCtx)

	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	console.SetColor("InfoFail", color.New(color.FgRed, color.Bold))

	ctx, cancel := context.WithCancel(globalContext)
	defer cancel()

	aliasedURL := cliCtx.Args().Get(0)
	admClient, err := newAdminClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client for `"+aliasedURL+"`.")

	anonClient, err := newAnonymousClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize anonymous client for `"+aliasedURL+"`.")

	admInfo, e := fetchAdminInfo(admClient)
	fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to get server info")

	// map to contain server stats for all the servers
	serverMap := make(map[string]serverStats)

	index := 1
	if cliCtx.IsSet("count") {
		count := cliCtx.Int("count")
		if count < 1 {
			fatalIf(errInvalidArgument().Trace(cliCtx.Args()...), "ping count cannot be less than 1")
		}
		for index <= count {
			// return if consecutive error count more then specified value
			if stop {
				return nil
			}
			ping(ctx, cliCtx, anonClient, admInfo, serverMap, index)
			index++
		}
	} else {
		for {
			select {
			case <-globalContext.Done():
				return globalContext.Err()
			default:
				// return if consecutive error count more then specified value
				if stop {
					return nil
				}
				ping(ctx, cliCtx, anonClient, admInfo, serverMap, index)
				index++
			}
		}
	}
	return nil
}
