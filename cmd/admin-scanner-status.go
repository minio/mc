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
	"bufio"
	"bytes"
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
	"github.com/fatih/color"
	"github.com/minio/cli"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/v3/console"
	"github.com/olekukonko/tablewriter"
)

var adminScannerInfoFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "nodes",
		Usage: "show only on matching servers, comma separate multiple",
	},
	cli.IntFlag{
		Name:  "n",
		Usage: "number of requests to run before exiting. 0 for endless",
		Value: 0,
	},
	cli.IntFlag{
		Name:  "interval",
		Usage: "interval between requests in seconds",
		Value: 3,
	},
	cli.IntFlag{
		Name:  "max-paths",
		Usage: "maximum number of active paths to show. -1 for unlimited",
		Value: -1,
	},
	cli.StringFlag{
		Name:   "in",
		Hidden: true,
		Usage:  "read previously saved json from file and replay",
	},
}

var adminScannerInfo = cli.Command{
	Name:            "status",
	Aliases:         []string{"info"},
	HiddenAliases:   true,
	Usage:           "summarize scanner events on MinIO server in real-time",
	Action:          mainAdminScannerInfo,
	OnUsageError:    onUsageError,
	Before:          setGlobalsFromContext,
	Flags:           append(adminScannerInfoFlags, globalFlags...),
	HideHelpCommand: true,
	CustomHelpTemplate: `NAME:
  {{.HelpName}} - {{.Usage}}

USAGE:
  {{.HelpName}} [FLAGS] TARGET

FLAGS:
  {{range .VisibleFlags}}{{.}}
  {{end}}
EXAMPLES:
   1. Display current in-progress all scanner operations.
      {{.Prompt}} {{.HelpName}} myminio/
`,
}

// checkAdminTopAPISyntax - validate all the passed arguments
func checkAdminScannerInfoSyntax(ctx *cli.Context) {
	if ctx.String("in") != "" {
		return
	}
	if len(ctx.Args()) == 0 || len(ctx.Args()) > 1 {
		showCommandHelpAndExit(ctx, 1) // last argument is exit code
	}
}

func mainAdminScannerInfo(ctx *cli.Context) error {
	checkAdminScannerInfoSyntax(ctx)

	aliasedURL := ctx.Args().Get(0)

	ui := tea.NewProgram(initScannerMetricsUI(ctx.Int("max-paths")))
	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	// Replay from file
	if inFile := ctx.String("in"); inFile != "" {
		go func() {
			if _, e := ui.Run(); e != nil {
				cancel()
				fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch scanner metrics")
			}
		}()
		f, e := os.Open(inFile)
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to open input")
		sc := bufio.NewReader(f)
		var lastTime time.Time
		for {
			b, e := sc.ReadBytes('\n')
			if e == io.EOF {
				break
			}
			var metrics madmin.RealtimeMetrics
			e = json.Unmarshal(b, &metrics)
			if e != nil || metrics.Aggregated.Scanner == nil {
				continue
			}
			delay := metrics.Aggregated.Scanner.CollectedAt.Sub(lastTime)
			if !lastTime.IsZero() && delay > 0 {
				if delay > 3*time.Second {
					delay = 3 * time.Second
				}
				time.Sleep(delay)
			}
			ui.Send(metrics)
			lastTime = metrics.Aggregated.Scanner.CollectedAt
		}
		os.Exit(0)
	}

	// Create a new MinIO Admin Client
	client, err := newAdminClient(aliasedURL)
	fatalIf(err.Trace(aliasedURL), "Unable to initialize admin client.")

	opts := madmin.MetricsOptions{
		Type:     madmin.MetricsScanner,
		N:        ctx.Int("n"),
		Interval: time.Duration(ctx.Int("interval")) * time.Second,
		Hosts:    strings.Split(ctx.String("nodes"), ","),
		ByHost:   false,
	}
	if globalJSON {
		e := client.Metrics(ctxt, opts, func(metrics madmin.RealtimeMetrics) {
			printMsg(metricsMessage{RealtimeMetrics: metrics})
		})

		if e != nil && !errors.Is(e, context.Canceled) {
			fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch scanner metrics")
		}
		return nil
	}

	go func() {
		e := client.Metrics(ctxt, opts, func(metrics madmin.RealtimeMetrics) {
			ui.Send(metrics)
		})

		if e != nil && !errors.Is(e, context.Canceled) {
			fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch scanner metrics")
		}
	}()

	if _, e := ui.Run(); e != nil {
		cancel()
		fatalIf(probe.NewError(e).Trace(aliasedURL), "Unable to fetch scanner metrics")
	}

	return nil
}

type metricsMessage struct {
	madmin.RealtimeMetrics
}

func (s metricsMessage) JSON() string {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetIndent("", " ")
	enc.SetEscapeHTML(false)

	fatalIf(probe.NewError(enc.Encode(s)), "Unable to marshal into JSON.")
	return buf.String()
}

func (s metricsMessage) String() string {
	return s.JSON()
}

func initScannerMetricsUI(maxPaths int) *scannerMetricsUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	console.SetColor("metrics-duration", color.New(color.FgHiWhite))
	console.SetColor("metrics-path", color.New(color.FgGreen))
	console.SetColor("metrics-error", color.New(color.FgHiRed))
	console.SetColor("metrics-title", color.New(color.FgCyan))
	console.SetColor("metrics-top-title", color.New(color.FgHiCyan))
	console.SetColor("metrics-number", color.New(color.FgHiWhite))
	console.SetColor("metrics-zero", color.New(color.FgHiWhite))
	console.SetColor("metrics-date", color.New(color.FgHiWhite))
	return &scannerMetricsUI{
		spinner:  s,
		maxPaths: maxPaths,
	}
}

type scannerMetricsUI struct {
	current  madmin.RealtimeMetrics
	spinner  spinner.Model
	quitting bool
	maxPaths int
}

func (m *scannerMetricsUI) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *scannerMetricsUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}
	case madmin.RealtimeMetrics:
		m.current = msg
		if msg.Final {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *scannerMetricsUI) View() string {
	var s strings.Builder

	if !m.quitting {
		s.WriteString(fmt.Sprintf("%s %s\n", console.Colorize("metrics-top-title", "Scanner Activity:"), m.spinner.View()))
	}

	// Set table header - akin to k8s style
	// https://github.com/olekukonko/tablewriter#example-10---set-nowhitespace-and-tablepadding-option
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

	writtenRows := 0
	addRow := func(s string) {
		table.Append([]string{s})
		writtenRows++
	}
	_ = addRow
	addRowF := func(format string, vals ...interface{}) {
		s := fmt.Sprintf(format, vals...)
		table.Append([]string{s})
		writtenRows++
	}

	sc := m.current.Aggregated.Scanner
	if sc == nil {
		s.WriteString("(waiting for data)")
		return s.String()
	}

	title := metricsTitle
	ui := metricsUint64
	const wantCycles = 16
	addRow("")
	if len(sc.CyclesCompletedAt) < 2 {
		addRow("Last full scan time:             Unknown (not enough data)")
	} else {
		addRow("Overall Statistics")
		addRow("------------------")
		sort.Slice(sc.CyclesCompletedAt, func(i, j int) bool {
			return sc.CyclesCompletedAt[i].After(sc.CyclesCompletedAt[j])
		})
		if len(sc.CyclesCompletedAt) >= wantCycles {
			sinceLast := sc.CyclesCompletedAt[0].Sub(sc.CyclesCompletedAt[wantCycles-1])
			perMonth := float64(30*24*time.Hour) / float64(sinceLast)
			cycleTime := console.Colorize("metrics-number", fmt.Sprintf("%dd%dh%dm", int(sinceLast.Hours()/24), int(sinceLast.Hours())%24, int(sinceLast.Minutes())%60))
			perms := console.Colorize("metrics-number", fmt.Sprintf("%.02f", perMonth))
			addRowF(title("Last full scan time:")+"   %s; Estimated %s/month", cycleTime, perms)
		} else {
			sinceLast := sc.CyclesCompletedAt[0].Sub(sc.CyclesCompletedAt[1]) * time.Duration(wantCycles)
			perMonth := float64(30*24*time.Hour) / float64(sinceLast)
			cycleTime := console.Colorize("metrics-number", fmt.Sprintf("%dd%dh%dm", int(sinceLast.Hours()/24), int(sinceLast.Hours())%24, int(sinceLast.Minutes())%60))
			perms := console.Colorize("metrics-number", fmt.Sprintf("%.02f", perMonth))
			addRowF(title("Est. full scan time:")+"   %s; Estimated %s/month", cycleTime, perms)
		}
	}
	if sc.CurrentCycle > 0 {
		addRowF(title("Current cycle:")+"         %s; Started: %v", ui(sc.CurrentCycle), console.Colorize("metrics-date", sc.CurrentStarted))
		addRowF(title("Active drives:")+"         %s", ui(uint64(len(sc.ActivePaths))))
	} else {
		addRowF(title("Current cycle:") + "         (between cycles)")
		addRowF(title("Active drives:")+"         %s", ui(uint64(len(sc.ActivePaths))))
	}
	getRate := func(x madmin.TimedAction) string {
		if x.AccTime > 0 {
			return fmt.Sprintf("; Rate: %v/day", ui(uint64(float64(24*time.Hour)/(float64(time.Minute)/float64(x.Count)))))
		}
		return ""
	}
	addRow("")
	addRow("Last Minute Statistics")
	addRow("----------------------")
	objs := uint64(0)
	x := sc.LastMinute.Actions["ScanObject"]
	{
		avg := x.Avg()
		addRowF(title("Objects Scanned:")+"       %s objects; Avg: %v%s", ui(x.Count), metricsDuration(avg), getRate(x))
		objs = x.Count
	}
	x = sc.LastMinute.Actions["ApplyVersion"]
	{
		avg := x.Avg()
		addRowF(title("Versions Scanned:")+"      %s versions; Avg: %v%s", ui(x.Count), metricsDuration(avg), getRate(x))
	}
	x = sc.LastMinute.Actions["HealCheck"]
	{
		avg := x.Avg()
		rate := ""
		if x.AccTime > 0 {
			rate = fmt.Sprintf("; Rate: %s/day", ui(uint64(float64(24*time.Hour)/(float64(time.Minute)/float64(x.Count)))))
		}
		addRowF(title("Versions Heal Checked:")+" %s versions; Avg: %v%s", ui(x.Count), metricsDuration(avg), rate)
	}
	x = sc.LastMinute.Actions["ReadMetadata"]
	addRowF(title("Read Metadata:")+"         %s objects; Avg: %v, Size: %v bytes/obj", ui(x.Count), metricsDuration(x.Avg()), ui(x.AvgBytes()))
	x = sc.LastMinute.Actions["ILM"]
	addRowF(title("ILM checks:")+"            %s versions; Avg: %v", ui(x.Count), metricsDuration(x.Avg()))
	x = sc.LastMinute.Actions["CheckReplication"]
	addRowF(title("Check Replication:")+"     %s versions; Avg: %v", ui(x.Count), metricsDuration(x.Avg()))
	x = sc.LastMinute.Actions["TierObjSweep"]
	if x.Count > 0 {
		addRowF(title("Sweep Tiered:")+"        %s versions; Avg: %v", ui(x.Count), metricsDuration(x.Avg()))
	}
	x = sc.LastMinute.Actions["CheckMissing"]
	addRowF(title("Verify Deleted:")+"        %s folders; Avg: %v", ui(x.Count), metricsDuration(x.Avg()))
	x = sc.LastMinute.Actions["HealAbandonedObject"]
	if x.Count > 0 {
		addRowF(title(" Missing Objects:")+"      %s objects healed; Avg: %v%s", ui(x.Count), metricsDuration(x.Avg()), getRate(x))
	}
	x = sc.LastMinute.Actions["HealAbandonedVersion"]
	if x.Count > 0 {
		addRowF(title(" Missing Versions:")+"     %s versions healed; Avg: %v%s; %v bytes/v", ui(x.Count), metricsDuration(x.Avg()), getRate(x), ui(x.AvgBytes()))
	}

	for k, x := range sc.LastMinute.ILM {
		const length = 17
		k += ":"
		if len(k) < length {
			k += strings.Repeat(" ", length-len(k))
		}
		addRowF(title("ILM, %s")+" %s actions; Avg: %v.", k, ui(x.Count), metricsDuration(x.Avg()))
	}
	x = sc.LastMinute.Actions["Yield"]
	{
		avg := fmt.Sprintf("%v", metricsDuration(x.Avg()))
		if objs > 0 {
			avg = console.Colorize("metrics-duration", fmt.Sprintf("%v/obj", metricsDuration(time.Duration(x.AccTime/objs))))
		}
		addRowF(title("Yield:")+"                 %v total; Avg: %s", metricsDuration(time.Duration(x.AccTime)), avg)
	}
	if errs := m.current.Errors; len(errs) > 0 {
		addRow("------------------------------------------- Errors --------------------------------------------------")
		for _, s := range errs {
			addRow(console.Colorize("metrics-error", s))
		}
	}

	if m.maxPaths != 0 && len(sc.ActivePaths) > 0 {
		addRow("------------------------------------- Currently Scanning Paths --------------------------------------")
		length := 100
		if globalTermWidth > 5 {
			length = globalTermWidth
		}
		for i, s := range sc.ActivePaths {
			if i == m.maxPaths {
				break
			}
			if globalTermHeight > 5 && writtenRows >= globalTermHeight-5 {
				addRow(console.Colorize("metrics-path", fmt.Sprintf("( ... hiding %d more disk(s) .. )", len(sc.ActivePaths)-i)))
				break
			}
			if len(s) > length {
				s = s[:length-3] + "..."
			}
			s = strings.ReplaceAll(s, "\\", "/")
			addRow(console.Colorize("metrics-path", s))
		}
	}
	table.Render()
	return s.String()
}

func metricsDuration(d time.Duration) string {
	if d == 0 {
		return console.Colorize("metrics-zero", "0ms")
	}
	if d > time.Millisecond {
		d = d.Round(time.Microsecond)
	}
	if d > time.Second {
		d = d.Round(time.Millisecond)
	}
	if d > time.Minute {
		d = d.Round(time.Second / 10)
	}
	return console.Colorize("metrics-duration", d)
}

func metricsUint64(v uint64) string {
	if v == 0 {
		return console.Colorize("metrics-zero", v)
	}
	return console.Colorize("metrics-number", v)
}

func metricsTitle(s string) string {
	return console.Colorize("metrics-title", s)
}
