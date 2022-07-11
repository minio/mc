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
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
	"github.com/olekukonko/tablewriter"
)

const (
	pingInterval = time.Second // keep it similar to unix ping interval
)

var pingFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "count, c",
		Usage: "perform liveliness check for count number of times",
		Value: 4,
	},
	cli.DurationFlag{
		Name:  "interval, i",
		Usage: "Wait interval between each request",
		Value: pingInterval,
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
     {{.Prompt}} {{.HelpName}} --interval 30s myminio
`,
}

// Validate command line arguments.
func checkPingSyntax(cliCtx *cli.Context) {
	if !cliCtx.Args().Present() {
		cli.ShowCommandHelpAndExit(cliCtx, "ping", 1) // last argument is exit code
	}
}

// PingResult is result for each ping
type PingResult struct {
	madmin.AliveResult
}

// PingResults is result for each ping for all hosts
type PingResults struct {
	Results map[string][]PingResult
	Final   bool
}

// JSON jsonified ping result message.
func (pr PingResult) JSON() string {
	statusJSONBytes, e := json.MarshalIndent(pr, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(statusJSONBytes)
}

// String colorized ping result message.
func (pr PingResult) String() (msg string) {
	if pr.Error == nil {
		coloredDot := console.Colorize("Info", dot)
		// Print server title
		msg += fmt.Sprintf("%s %s:", coloredDot, console.Colorize("PrintB", pr.Endpoint.String()))
		msg += fmt.Sprintf(" time=%s\n", pr.ResponseTime)
		return
	}
	coloredDot := console.Colorize("InfoFail", dot)
	msg += fmt.Sprintf("%s %s:", coloredDot, console.Colorize("PrintB", pr.Endpoint.String()))
	msg += fmt.Sprintf(" time=%s, error=%s\n", pr.ResponseTime, console.Colorize("InfoFail", pr.Error.Error()))

	return msg
}

type pingUI struct {
	spinner  spinner.Model
	quitting bool
	results  PingResults
}

func initPingUI() *pingUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &pingUI{
		spinner: s,
	}
}

func (m *pingUI) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *pingUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}
	case PingResults:
		m.results = msg
		if msg.Final {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}

func (m *pingUI) View() string {
	var s strings.Builder

	// Set table header
	table := tablewriter.NewWriter(&s)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)

	res := m.results

	if len(res.Results) > 0 {
		s.WriteString("\n")
	}

	trailerIfGreaterThan := func(in string, max int) string {
		if len(in) < max {
			return in
		}
		return in[:max] + "..."
	}

	table.SetHeader([]string{"Node", "Avg-Latency", "Count", ""})
	data := make([][]string, 0, len(res.Results))

	if len(res.Results) == 0 {
		data = append(data, []string{
			"...",
			whiteStyle.Render("-- ms"),
			whiteStyle.Render("--"),
			"",
		})
	} else {
		for k, results := range res.Results {
			data = append(data, []string{
				trailerIfGreaterThan(k, 64),
				getAvgLatency(results...).String(),
				strconv.Itoa(len(results)),
				"",
			})
		}

		sort.Slice(data, func(i, j int) bool {
			return data[i][0] < data[j][0]
		})

		table.AppendBulk(data)
		table.Render()
	}
	if !m.quitting {
		s.WriteString(fmt.Sprintf("\nPinging: %s", m.spinner.View()))
	} else {
		s.WriteString("\n")
	}
	return s.String()
}

func getAvgLatency(results ...PingResult) (avg time.Duration) {
	if len(results) == 0 {
		return avg
	}
	var totalDurationNS uint64
	for _, result := range results {
		totalDurationNS += uint64(result.ResponseTime.Nanoseconds())
	}
	return time.Duration(totalDurationNS / uint64(len(results)))
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

// mainPing is entry point for ping command.
func mainPing(cliCtx *cli.Context) error {
	// check 'ping' cli arguments.
	checkPingSyntax(cliCtx)

	console.SetColor("Info", color.New(color.FgGreen, color.Bold))
	console.SetColor("InfoFail", color.New(color.FgRed, color.Bold))

	ctx, cancel := context.WithCancel(globalContext)
	defer cancel()

	aliasedURL := cliCtx.Args().Get(0)

	count := cliCtx.Int("count")
	if count < 1 {
		fatalIf(errInvalidArgument().Trace(cliCtx.Args()...), "ping count cannot be less than 1")
	}

	interval := cliCtx.Duration("interval")

	admClient, err := newAdminClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client for `"+aliasedURL+"`.")

	anonClient, err := newAnonymousClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize anonymous client for `"+aliasedURL+"`.")

	done := make(chan struct{})

	ui := tea.NewProgram(initPingUI())
	if !globalJSON {
		go func() {
			if e := ui.Start(); e != nil {
				cancel()
				os.Exit(1)
			}
			close(done)
		}()
	}

	admInfo, e := fetchAdminInfo(admClient)
	fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to get server info")

	pingResults := PingResults{
		Results: make(map[string][]PingResult),
	}
	for i := 0; i < count; i++ {
		for result := range anonClient.Alive(ctx, madmin.AliveOpts{}, admInfo.Servers...) {
			if globalJSON {
				printMsg(PingResult{result})
			} else {
				hostResults, ok := pingResults.Results[result.Endpoint.Host]
				if !ok {
					pingResults.Results[result.Endpoint.Host] = []PingResult{{result}}
				} else {
					hostResults = append(hostResults, PingResult{result})
					pingResults.Results[result.Endpoint.Host] = hostResults
				}
				ui.Send(pingResults)
			}
		}
		time.Sleep(interval)
	}
	if !globalJSON {
		pingResults.Final = true
		ui.Send(pingResults)

		<-done
	}
	return nil
}
