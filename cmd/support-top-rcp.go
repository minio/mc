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
	"github.com/klauspost/compress/zstd"
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
			defer cancel()
			if _, e := ui.Run(); e != nil {
				fatalIf(probe.NewError(e), "Unable to fetch scanner metrics")
			}
		}()
		f, e := os.Open(inFile)
		fatalIf(probe.NewError(e), "Unable to open input")
		defer f.Close()
		in := io.Reader(f)
		if strings.HasSuffix(inFile, ".zst") {
			zr, e := zstd.NewReader(in)
			fatalIf(probe.NewError(e), "Unable to open input")
			defer zr.Close()
			in = zr
		}
		sc := bufio.NewReader(in)
		var lastTime time.Time
		for ctxt.Err() == nil {
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

const (
	rpcSortHostname = iota
	rpcSortReconnections
	rpcSortQueue
	rpcSortPing
)

type topRPCUI struct {
	spinner  spinner.Model
	offset   int
	quitting bool
	pageSz   int
	showTo   bool
	sortBy   uint8
	curr     madmin.RealtimeMetrics
	frozen   *madmin.RealtimeMetrics
}

func (m *topRPCUI) Init() tea.Cmd {
	m.showTo = true
	return m.spinner.Tick
}

func (m *topRPCUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up":
			m.offset--
		case "down":
			m.offset++
		case "pgdown":
			m.offset += m.pageSz - 1
		case "pgup":
			m.offset -= m.pageSz - 1
		case "t":
			m.showTo = true
		case "f":
			m.showTo = false
		case "r":
			m.sortBy = rpcSortReconnections
		case "q":
			m.sortBy = rpcSortQueue
		case "p":
			m.sortBy = rpcSortPing
		case tea.KeySpace.String():
			if m.frozen == nil {
				freeze := m.curr
				m.frozen = &freeze
			} else {
				m.frozen = nil
			}
		case "tab":
			m.showTo = !m.showTo
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
	byhost := m.curr.ByHost
	if m.frozen != nil {
		rpc = m.frozen.Aggregated.RPC
		byhost = m.frozen.ByHost
	}
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
	if len(byhost) > 0 {
		for k, v := range byhost {
			if v.RPC != nil {
				fromHost[k] = *v.RPC
			}
		}
	}
	sortBy := ""
	switch m.sortBy {
	case rpcSortReconnections:
		sortBy = " sorted by RECONNS"
		if m.showTo {
			sort.Slice(hosts, func(i, j int) bool {
				if intoHost[hosts[i]].ReconnectCount != intoHost[hosts[j]].ReconnectCount {
					return intoHost[hosts[i]].ReconnectCount > intoHost[hosts[j]].ReconnectCount
				}
				return hosts[i] < hosts[j]
			})
		} else {
			sort.Slice(hosts, func(i, j int) bool {
				if fromHost[hosts[i]].ReconnectCount != fromHost[hosts[j]].ReconnectCount {
					return fromHost[hosts[i]].ReconnectCount > fromHost[hosts[j]].ReconnectCount
				}
				return hosts[i] < hosts[j]
			})
		}
	case rpcSortQueue:
		sortBy = " sorted by Queue"
		if m.showTo {
			sort.Slice(hosts, func(i, j int) bool {
				if intoHost[hosts[i]].OutQueue != intoHost[hosts[j]].OutQueue {
					return intoHost[hosts[i]].OutQueue > intoHost[hosts[j]].OutQueue
				}
				return hosts[i] < hosts[j]
			})
		} else {
			sort.Slice(hosts, func(i, j int) bool {
				if fromHost[hosts[i]].OutQueue != fromHost[hosts[j]].OutQueue {
					return fromHost[hosts[i]].OutQueue > fromHost[hosts[j]].OutQueue
				}
				return hosts[i] < hosts[j]
			})
		}
	case rpcSortPing:
		sortBy = " sorted by Ping"
		if m.showTo {
			sort.Slice(hosts, func(i, j int) bool {
				if intoHost[hosts[i]].LastPingMS != intoHost[hosts[j]].LastPingMS {
					return intoHost[hosts[i]].LastPingMS > intoHost[hosts[j]].LastPingMS
				}
				return hosts[i] < hosts[j]
			})
		} else {
			sort.Slice(hosts, func(i, j int) bool {
				if fromHost[hosts[i]].LastPingMS != fromHost[hosts[j]].LastPingMS {
					return fromHost[hosts[i]].LastPingMS > fromHost[hosts[j]].LastPingMS
				}
				return hosts[i] < hosts[j]
			})
		}
	default:
		sortBy = " sorted by Host"
		sort.Strings(hosts)
	}
	allhosts := hosts
	maxHosts := max(3, globalTermHeight-4) // at least 3 hosts.
	m.pageSz = maxHosts
	truncate := len(hosts) > maxHosts && !m.quitting
	hostsShown := 0
	if m.offset >= len(hosts)-maxHosts {
		m.offset = len(hosts) - maxHosts
	}
	if m.offset < 0 {
		m.offset = 0
	}
	hosts = hosts[m.offset:]
	dataRender := make([][]string, 0, maxHosts)
	for _, host := range hosts {
		if hostsShown == maxHosts {
			truncate = true
			break
		}
		if m.showTo {
			if v, ok := intoHost[host]; ok {
				if m.sortBy == rpcSortReconnections && v.ReconnectCount == 0 {
					continue
				}
				dataRender = append(dataRender, []string{
					fmt.Sprintf("To %s", host),
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
				hostsShown++
			}
			continue
		}
		if v, ok := fromHost[host]; ok {
			if m.sortBy == rpcSortReconnections && v.ReconnectCount == 0 {
				continue
			}
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
			hostsShown++
		}
	}
	dir := "TO"
	if !m.showTo {
		dir = "FROM"
	}
	table.AppendBulk(dataRender)
	table.Render()
	pre := "\n"
	if m.frozen != nil {
		if time.Now().UnixMilli()&512 < 256 {
			pre = "\n[PAUSED] "
		} else {
			pre = "\n(PAUSED) "
		}
	}
	s.WriteString(pre)
	if truncate {
		s.WriteString(fmt.Sprintf("SHOWING %s Host %d to %d of %d%s. ↑ and ↓ available. <tab>=TO/FROM r=RECON q=Q p=PING.", dir, 1+m.offset, m.offset+hostsShown, len(allhosts), sortBy))
	} else {
		s.WriteString(fmt.Sprintf("SHOWING traffic %s hosts%s. <tab>=TO/FROM r=RECON q=Q p=PING.", dir, sortBy))
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
