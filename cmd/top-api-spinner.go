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
	"math"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/minio/madmin-go/v3"
	"github.com/olekukonko/tablewriter"
)

type topAPIStats struct {
	TotalCalls         uint64
	TotalBytesRX       uint64
	TotalBytesTX       uint64
	TotalErrors        uint64
	TotalDurationNanos uint64
	MaxDurationNanos   uint64
	MinDurationNanos   uint64
}

func (s *topAPIStats) addAPICall(n int) {
	atomic.AddUint64(&s.TotalCalls, uint64(n))
}

func (s *topAPIStats) addAPIBytesRX(n int) {
	atomic.AddUint64(&s.TotalBytesRX, uint64(n))
}

func (s *topAPIStats) addAPIBytesTX(n int) {
	atomic.AddUint64(&s.TotalBytesTX, uint64(n))
}

func (s *topAPIStats) addAPIErrors(n int) {
	atomic.AddUint64(&s.TotalErrors, uint64(n))
}

func (s *topAPIStats) addAPIDurationNanos(n int64) {
	atomic.AddUint64(&s.TotalDurationNanos, uint64(n))
	if s.MinDurationNanos == 0 {
		s.MinDurationNanos = uint64(n)
	}
	s.MinDurationNanos = uint64(math.Min(float64(n), float64(s.MinDurationNanos)))
	s.MaxDurationNanos = uint64(math.Max(float64(n), float64(s.MaxDurationNanos)))
}

func (s *topAPIStats) loadAPICall() uint64 {
	return atomic.LoadUint64(&s.TotalCalls)
}

func (s *topAPIStats) loadAPIBytesRX() uint64 {
	return atomic.LoadUint64(&s.TotalBytesRX)
}

func (s *topAPIStats) loadAPIBytesTX() uint64 {
	return atomic.LoadUint64(&s.TotalBytesTX)
}

func (s *topAPIStats) loadAPIErrors() uint64 {
	return atomic.LoadUint64(&s.TotalErrors)
}

func (s *topAPIStats) loadAPIDurationNanos() uint64 {
	return atomic.LoadUint64(&s.TotalDurationNanos)
}

func (s *topAPIStats) loadAPIMinDurationNanos() uint64 {
	return atomic.LoadUint64(&s.MinDurationNanos)
}

func (s *topAPIStats) loadAPIMaxDurationNanos() uint64 {
	return atomic.LoadUint64(&s.MaxDurationNanos)
}

type traceUI struct {
	spinner     spinner.Model
	quitting    bool
	startTime   time.Time
	result      topAPIResult
	lastResult  topAPIResult
	apiStatsMap map[string]*topAPIStats
}

type topAPIResult struct {
	final       bool
	apiCallInfo madmin.ServiceTraceInfo
}

func initTraceUI() *traceUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &traceUI{
		spinner:     s,
		apiStatsMap: make(map[string]*topAPIStats),
	}
}

func (m *traceUI) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *traceUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}
	case topAPIResult:
		m.result = msg
		if m.result.apiCallInfo.Trace.FuncName != "" {
			m.lastResult = m.result
		}
		if msg.final {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *traceUI) View() string {
	var s strings.Builder
	s.WriteString("\n")

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

	res := m.result.apiCallInfo
	if m.startTime.IsZero() && !res.Trace.Time.IsZero() {
		m.startTime = res.Trace.Time
	}
	if res.Trace.FuncName != "" && res.Trace.FuncName != "errorResponseHandler" {
		traceSt, ok := m.apiStatsMap[res.Trace.FuncName]
		if !ok {
			traceSt = &topAPIStats{}
		}
		traceSt.addAPICall(1)
		if res.Trace.HTTP != nil {
			traceSt.addAPIBytesRX(res.Trace.HTTP.CallStats.InputBytes)
			traceSt.addAPIBytesTX(res.Trace.HTTP.CallStats.OutputBytes)
			traceSt.addAPIDurationNanos(res.Trace.Duration.Nanoseconds())
		}
		if res.Trace.HTTP.RespInfo.StatusCode >= 499 {
			traceSt.addAPIErrors(1)
		}
		m.apiStatsMap[res.Trace.FuncName] = traceSt
	}

	table.SetHeader([]string{"API", "RX", "TX", "CALLS", "ERRORS", "MinRT", "MaxRT", "AvgRT", "AvgTP"})
	data := make([][]string, 0, len(m.apiStatsMap))

	for k, stats := range m.apiStatsMap {
		secs := time.Duration(stats.loadAPIDurationNanos()).Seconds()
		bytes := float64(stats.loadAPIBytesRX() + stats.loadAPIBytesTX())
		data = append(data, []string{
			k,
			whiteStyle.Render(humanize.IBytes(stats.loadAPIBytesRX())),
			whiteStyle.Render(humanize.IBytes(stats.loadAPIBytesTX())),
			whiteStyle.Render(fmt.Sprintf("%d", stats.loadAPICall())),
			whiteStyle.Render(fmt.Sprintf("%d", stats.loadAPIErrors())),
			whiteStyle.Render(fmt.Sprintf("%.03f s", time.Duration(stats.loadAPIMinDurationNanos()).Seconds())),
			whiteStyle.Render(fmt.Sprintf("%.03f s", time.Duration(stats.loadAPIMaxDurationNanos()).Seconds())),
			whiteStyle.Render(fmt.Sprintf("%.03f s", time.Duration(stats.loadAPIDurationNanos()/stats.loadAPICall()).Seconds())),
			whiteStyle.Render(fmt.Sprintf("%s/s", humanize.IBytes(uint64(bytes/secs)))),
		})
	}
	sort.Slice(data, func(i, j int) bool {
		return data[i][0] < data[j][0]
	})

	table.AppendBulk(data)
	table.Render()

	if !m.quitting {
		s.WriteString(fmt.Sprintf("\nTopAPI: %s", m.spinner.View()))
	} else {
		var totalTX, totalRX, totalCalls uint64
		lastReqTime := m.lastResult.apiCallInfo.Trace.Time
		if m.lastResult.apiCallInfo.Trace.Time.IsZero() {
			lastReqTime = time.Now()
		}
		for _, stats := range m.apiStatsMap {
			totalRX += stats.loadAPIBytesRX()
			totalTX += stats.loadAPIBytesTX()
			totalCalls += stats.loadAPICall()
		}

		msg := fmt.Sprintf("\nSummary:\n\nTotal: %d CALLS, %s RX, %s TX",
			totalCalls,
			humanize.IBytes(totalRX),
			humanize.IBytes(totalTX),
		)
		if !m.startTime.IsZero() {
			msg += fmt.Sprintf(" - in %.02fs", lastReqTime.Sub(m.startTime).Seconds())
		}

		s.WriteString(msg)
		s.WriteString("\n")
	}
	return s.String()
}
