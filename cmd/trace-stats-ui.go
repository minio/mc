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
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/pkg/v3/console"
	"github.com/muesli/reflow/truncate"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
)

type traceStatsUI struct {
	current    *statTrace
	started    time.Time
	meter      spinner.Model
	quitting   bool
	maxEntries int
	offset     int
	allFlag    bool
}

func (m *traceStatsUI) Init() tea.Cmd {
	return m.meter.Tick
}

func (m *traceStatsUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "r":
			// Reset min/max
			m.current.mu.Lock()
			defer m.current.mu.Unlock()
			for k, si := range m.current.Calls {
				si.MaxDur, si.MinDur = 0, 0
				si.MaxTTFB = 0
				m.current.Calls[k] = si
			}
			return m, nil
		case "down":
			m.offset++
		case "up":
			m.offset--
		case "home":
			m.offset = 0
		case "end":
			m.offset = len(m.current.Calls)
		default:
			return m, nil
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.meter, cmd = m.meter.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *traceStatsUI) View() string {
	var s strings.Builder

	dur := m.current.Latest.Sub(m.current.Oldest)
	s.WriteString(fmt.Sprintf("%s %s\n",
		console.Colorize("metrics-top-title", "Duration: "+dur.Round(time.Second).String()), m.meter.View()))

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
	table.SetTablePadding("  ") // pad with tabs
	table.SetNoWhiteSpace(true)
	var entries []statItem

	m.current.mu.Lock()
	var (
		totalCnt = 0
		totalRX  = 0
		totalTX  = 0
	)
	for _, v := range m.current.Calls {
		totalCnt += v.Count
		totalRX += v.CallStats.Rx
		totalTX += v.CallStats.Tx
		entries = append(entries, v)
	}
	m.current.mu.Unlock()
	if len(entries) == 0 {
		s.WriteString("(waiting for data)")
		return s.String()
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Count > entries[j].Count
	})

	frontTrunc := false
	endTrunc := false
	m.offset = min(max(0, m.offset), len(entries))
	offset := m.offset
	if m.maxEntries > 0 && len(entries) > m.maxEntries {
		entLeft := len(entries) - offset
		if entLeft > m.maxEntries {
			// Truncate both ends
			entries = entries[offset : m.maxEntries+offset]
			frontTrunc = offset > 0
			endTrunc = true
		} else {
			entries = entries[len(entries)-m.maxEntries:]
			frontTrunc = true
		}
	}
	hasTTFB := false
	for _, e := range entries {
		if e.TTFB > 0 {
			hasTTFB = true
			break
		}
	}

	if !m.allFlag {
		if totalRX > 0 {
			s.WriteString(console.Colorize("metrics-top-title", fmt.Sprintf("RX Rate:↑ %s/m\n",
				humanize.IBytes(uint64(float64(totalRX)/dur.Minutes())))))
		}
		if totalTX > 0 {
			s.WriteString(console.Colorize("metrics-top-title", fmt.Sprintf("TX Rate:↓ %s/m\n",
				humanize.IBytes(uint64(float64(totalTX)/dur.Minutes())))))
		}
	}
	s.WriteString(console.Colorize("metrics-top-title", fmt.Sprintf("RPM    :  %0.1f\n", float64(totalCnt)/dur.Minutes())))
	s.WriteString("-------------\n")

	preCall := ""
	if frontTrunc {
		preCall = console.Colorize("metrics-error", "↑ ")
	}

	t := []string{
		preCall + console.Colorize("metrics-top-title", "Call"),
		console.Colorize("metrics-top-title", "Count"),
		console.Colorize("metrics-top-title", "RPM"),
		console.Colorize("metrics-top-title", "Avg Time"),
		console.Colorize("metrics-top-title", "Min Time"),
		console.Colorize("metrics-top-title", "Max Time"),
	}
	if hasTTFB {
		t = append(t,
			console.Colorize("metrics-top-title", "Avg TTFB"),
			console.Colorize("metrics-top-title", "Max TTFB"),
		)
	}
	t = append(t,
		console.Colorize("metrics-top-title", "Avg Size"),
		console.Colorize("metrics-top-title", "Rate /min"),
		console.Colorize("metrics-top-title", "Errors"),
	)

	table.Append(t)

	for i, v := range entries {
		if v.Count <= 0 {
			continue
		}
		errs := "0"
		if v.Errors > 0 {
			errs = console.Colorize("metrics-error", strconv.Itoa(v.Errors))
		}
		avg := v.Duration / time.Duration(v.Count)
		avgTTFB := v.TTFB / time.Duration(v.Count)
		avgColor := "metrics-dur"
		if avg > 10*time.Second {
			avgColor = "metrics-dur-high"
		} else if avg > 2*time.Second {
			avgColor = "metrics-dur-med"
		}

		minColor := "metrics-dur"
		if v.MinDur > 10*time.Second {
			minColor = "metrics-dur-high"
		} else if v.MinDur > 2*time.Second {
			minColor = "metrics-dur-med"
		}

		maxColor := "metrics-dur"
		if v.MaxDur > 10*time.Second {
			maxColor = "metrics-dur-high"
		} else if v.MaxDur > time.Second {
			maxColor = "metrics-dur-med"
		}

		sz := "-"
		rate := "-"
		if v.Size > 0 && v.Count > 0 {
			sz = ibytesShort(uint64(v.Size) / uint64(v.Count))
			rate = ibytesShort(uint64(float64(v.Size) / dur.Minutes()))
		}
		if v.CallStatsCount > 0 {
			var s, r []string
			if v.CallStats.Rx > 0 {
				s = append(s, fmt.Sprintf("↑%s", ibytesShort(uint64(v.CallStats.Rx/v.CallStatsCount))))
				r = append(r, fmt.Sprintf("↑%s", ibytesShort(uint64(float64(v.CallStats.Rx)/dur.Minutes()))))
			}
			if v.CallStats.Tx > 0 {
				s = append(s, fmt.Sprintf("↓%s", ibytesShort(uint64(v.CallStats.Tx/v.CallStatsCount))))
				r = append(r, fmt.Sprintf("↓%s", ibytesShort(uint64(float64(v.CallStats.Tx)/dur.Minutes()))))
			}
			if len(s)+len(r) > 0 {
				sz = strings.Join(s, " ")
			}
			if len(r) > 0 {
				rate = strings.Join(r, " ")
			}
		}
		if sz != "-" {
			sz = console.Colorize("metrics-size", sz)
			rate = console.Colorize("metrics-size", rate)
		}

		preCall = ""
		if endTrunc && i == len(entries)-1 {
			preCall = console.Colorize("metrics-error", "↓ ")
		}

		t := []string{
			preCall + console.Colorize("metrics-title", metricsTitle(v.Name)),
			console.Colorize("metrics-number", fmt.Sprintf("%d ", v.Count)) +
				console.Colorize("metrics-number-secondary", fmt.Sprintf("(%0.1f%%)", float64(v.Count)/float64(totalCnt)*100)),
			console.Colorize("metrics-number", fmt.Sprintf("%0.1f", float64(v.Count)/dur.Minutes())),
			console.Colorize(avgColor, fmt.Sprintf("%v", roundDur(avg))),
			console.Colorize(minColor, roundDur(v.MinDur)),
			console.Colorize(maxColor, roundDur(v.MaxDur)),
		}
		if hasTTFB {
			if v.TTFB > 0 {
				t = append(t,
					console.Colorize(avgColor, fmt.Sprintf("%v", roundDur(avgTTFB))),
					console.Colorize(maxColor, roundDur(v.MaxTTFB)))
			} else {
				t = append(t, "-", "-")
			}
		}
		t = append(t, sz, rate, errs)
		table.Append(t)
	}

	table.Render()
	if globalTermWidth <= 10 {
		return s.String()
	}
	w := globalTermWidth
	if nw, _, e := term.GetSize(int(os.Stdout.Fd())); e == nil {
		w = nw
	}
	split := strings.Split(s.String(), "\n")
	for i, line := range split {
		split[i] = truncate.StringWithTail(line, uint(w), "»")
	}
	return strings.Join(split, "\n")
}

// ibytesShort returns a short un-padded version of the value from humanize.IBytes.
func ibytesShort(v uint64) string {
	return strings.ReplaceAll(strings.TrimSuffix(humanize.IBytes(v), "iB"), " ", "")
}

// roundDur will round the duration to a nice, printable number, with "reasonable" precision.
func roundDur(d time.Duration) time.Duration {
	if d > time.Minute {
		return d.Round(time.Second)
	}
	if d > time.Second {
		return d.Round(time.Millisecond)
	}
	if d > time.Millisecond {
		return d.Round(time.Millisecond / 10)
	}
	return d.Round(time.Microsecond)
}

func initTraceStatsUI(allFlag bool, maxEntries int, traces <-chan madmin.ServiceTraceInfo) *traceStatsUI {
	meter := spinner.New()
	meter.Spinner = spinner.Meter
	meter.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	// Use half the default fps to reduce flickering
	meter.Spinner.FPS = time.Second / 3
	console.SetColor("metrics-duration", color.New(color.FgWhite))
	console.SetColor("metrics-size", color.New(color.FgGreen))
	console.SetColor("metrics-dur", color.New(color.FgGreen))
	console.SetColor("metrics-dur-med", color.New(color.FgYellow))
	console.SetColor("metrics-dur-high", color.New(color.FgRed))
	console.SetColor("metrics-error", color.New(color.FgYellow))
	console.SetColor("metrics-title", color.New(color.FgCyan))
	console.SetColor("metrics-top-title", color.New(color.FgHiCyan))
	console.SetColor("metrics-number", color.New(color.FgWhite))
	console.SetColor("metrics-number-secondary", color.New(color.FgBlue))
	console.SetColor("metrics-zero", color.New(color.FgWhite))
	stats := &statTrace{Calls: make(map[string]statItem, 20)}
	go func() {
		for t := range traces {
			stats.add(t)
		}
	}()
	return &traceStatsUI{
		started:    time.Now(),
		meter:      meter,
		maxEntries: maxEntries,
		current:    stats,
		allFlag:    allFlag,
	}
}
