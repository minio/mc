// Copyright (c) 2015-2024 MinIO, Inc.
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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/olekukonko/tablewriter"
)

var supportTopRPCFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "nodes",
		Usage: "collect only metrics from matching servers, comma separate multiple",
	},
	cli.IntFlag{
		Name:  "interval",
		Usage: "interval between requests in seconds",
		Value: 1,
	},
	cli.IntFlag{
		Name:  "n",
		Usage: "number of requests to run before exiting. 0 for endless (default)",
		Value: 0,
	},
	cli.StringFlag{
		Name:  "in",
		Usage: "read previously saved json from file and replay",
	},
}

var supportTopRPCCmd = cli.Command{
	Name:            "rpc",
	HiddenAliases:   true,
	Usage:           "show real-time rpc metrics (grid only)",
	Action:          mainSupportTopRPC,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(supportTopRPCFlags, supportGlobalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Display net metrics
      {{.Prompt}} {{.HelpName}} myminio/
`,
}

// checkSupportTopNetSyntax - validate all the passed arguments
func checkSupportTopRPCSyntax(ctx *cli.Context) {
	if ctx.String("in") != "" {
		return
	}
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainSupportTopRPC(ctx *cli.Context) error {
	checkSupportTopRPCSyntax(ctx)

	ui := tea.NewProgram(initTopRPCUI())
	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	// Replay from file.
	if inFile := ctx.String("in"); inFile != "" {
		go func() {
			if _, e := ui.Run(); e != nil {
				cancel()
				fatalIf(probe.NewError(e), "Unable to fetch scanner metrics")
			}
		}()
		f, e := os.Open(inFile)
		fatalIf(probe.NewError(e), "Unable to open input")
		sc := bufio.NewReader(f)
		var lastTime time.Time
		for {
			b, e := sc.ReadBytes('\n')
			if e == io.EOF {
				break
			}
			var metrics madmin.RealtimeMetrics
			e = json.Unmarshal(b, &metrics)
			if e != nil || metrics.Aggregated.RPC == nil {
				continue
			}
			delay := metrics.Aggregated.RPC.CollectedAt.Sub(lastTime)
			if !lastTime.IsZero() && delay > 0 {
				if delay > 3*time.Second {
					delay = 3 * time.Second
				}
				time.Sleep(delay)
			}
			ui.Send(metrics)
			lastTime = metrics.Aggregated.RPC.CollectedAt
		}
		os.Exit(0)
	}

	aliasedURL := ctx.Args().Get(0)
	alias, _ := url2Alias(aliasedURL)
	validateClusterRegistered(alias, false)

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	if err != nil {
		fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	// MetricsOptions are options provided to Metrics call.
	opts := madmin.MetricsOptions{
		Type:     madmin.MetricsRPC,
		Interval: time.Duration(ctx.Int("interval")) * time.Second,
		N:        ctx.Int("n"),
		Hosts:    strings.Split(ctx.String("nodes"), ","),
		ByHost:   true,
	}
	if globalJSON {
		e := client.Metrics(ctxt, opts, func(metrics madmin.RealtimeMetrics) {
			printMsg(metricsMessage{RealtimeMetrics: metrics})
		})
		if e != nil && !errors.Is(e, context.Canceled) {
			fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch net metrics")
		}
		return nil
	}
	go func() {
		out := func(m madmin.RealtimeMetrics) {
			ui.Send(m)
		}

		e := client.Metrics(ctxt, opts, out)
		if e != nil {
			fatalIf(probe.NewError(e), "Unable to fetch top net events")
		}
		ui.Quit()
	}()

	if _, e := ui.Run(); e != nil {
		cancel()
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch top net events")
	}

	return nil
}

type topRPCUI struct {
	spinner  spinner.Model
	offset   int
	quitting bool
	curr     madmin.RealtimeMetrics
}

func (m *topRPCUI) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *topRPCUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up":
			m.offset--
		case "down":
			m.offset++
		}
		return m, nil
	case madmin.RealtimeMetrics:
		m.curr = msg
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

func (m *topRPCUI) View() string {
	var s strings.Builder
	// Set table header
	table := tablewriter.NewWriter(&s)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)
	table.SetHeader([]string{"SERVER", "CONCTD", "PING", "PONG", "OUT.Q", "RECONNS", "STR.IN", "STR.OUT", "MSG.IN", "MSG.OUT"})

	rpc := m.curr.Aggregated.RPC
	if rpc == nil || len(rpc.ByDestination) == 0 {
		table.Render()
		s.WriteString("\n(no rpc connections)\n")
		return s.String()
	}
	hosts := make([]string, 0, len(rpc.ByDestination))
	intoHost := make(map[string]madmin.RPCMetrics, len(rpc.ByDestination))
	fromHost := make(map[string]madmin.RPCMetrics, len(rpc.ByDestination))
	for k, v := range rpc.ByDestination {
		k = strings.TrimPrefix(k, "http://")
		k = strings.TrimPrefix(k, "https://")
		hosts = append(hosts, k)
		intoHost[k] = v
	}
	byhost := m.curr.ByHost
	if len(byhost) > 0 {
		for k, v := range byhost {
			if v.RPC != nil {
				fromHost[k] = *v.RPC
			}
		}
	}
	sort.Strings(hosts)
	allhosts := hosts
	maxHosts := max(3, (globalTermHeight-4)/2) // 2 lines of output per host, or at least 3 hosts.

	truncate := len(hosts) > maxHosts && !m.quitting
	if truncate {
		if m.offset < 0 {
			m.offset = 0
		}
		if m.offset >= len(hosts)-maxHosts {
			m.offset = len(hosts) - maxHosts
		}
		hosts = hosts[m.offset:]
		if len(hosts) > maxHosts {
			hosts = hosts[:maxHosts]
		}
	}
	dataRender := make([][]string, 0, len(hosts)*2)
	for _, host := range hosts {
		if v, ok := intoHost[host]; ok {
			dataRender = append(dataRender, []string{
				fmt.Sprintf(" To  %s", host),
				fmt.Sprintf("%d", v.Connected),
				fmt.Sprintf("%0.1fms", v.LastPingMS),
				fmt.Sprintf("%ds ago", v.CollectedAt.Sub(v.LastPongTime)/time.Second),
				fmt.Sprintf("%d", v.OutQueue),
				fmt.Sprintf("%d", v.ReconnectCount),
				fmt.Sprintf("->%d", v.IncomingStreams),
				fmt.Sprintf("%d->", v.OutgoingStreams),
				fmt.Sprintf("%d", v.IncomingMessages),
				fmt.Sprintf("%d", v.OutgoingMessages),
			})
		}
		if v, ok := fromHost[host]; ok {
			dataRender = append(dataRender, []string{
				fmt.Sprintf("From %s", host),
				fmt.Sprintf("%d", v.Connected),
				fmt.Sprintf("%0.1fms", v.LastPingMS),
				fmt.Sprintf("%ds ago", v.CollectedAt.Sub(v.LastPongTime)/time.Second),
				fmt.Sprintf("%d", v.OutQueue),
				fmt.Sprintf("%d", v.ReconnectCount),
				fmt.Sprintf("->%d", v.IncomingStreams),
				fmt.Sprintf("%d->", v.OutgoingStreams),
				fmt.Sprintf("%d", v.IncomingMessages),
				fmt.Sprintf("%d", v.OutgoingMessages),
			})
		}
	}

	table.AppendBulk(dataRender)
	table.Render()
	if truncate {
		s.WriteString(fmt.Sprintf("\nSHOWING Host %d to %d of %d. Use ↑ and ↓ to see different hosts", 1+m.offset, m.offset+len(hosts), len(allhosts)))
	}
	return s.String()
}

func initTopRPCUI() *topRPCUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &topRPCUI{
		spinner: s,
	}
}
