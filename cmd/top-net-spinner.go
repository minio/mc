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
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/minio/madmin-go/v3"
	"github.com/olekukonko/tablewriter"
	"github.com/prometheus/procfs"
)

type topNetUI struct {
	spinner  spinner.Model
	quitting bool

	sortAsc bool

	prevTopMap map[string]topNetResult
	currTopMap map[string]topNetResult
}

type topNetResult struct {
	final    bool
	endPoint string
	error    string
	stats    madmin.NetMetrics
}

func (t topNetResult) GetTotalBytes() uint64 {
	return t.stats.NetStats.RxBytes + t.stats.NetStats.TxBytes
}

func (m *topNetUI) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *topNetUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	case topNetResult:
		m.prevTopMap[msg.endPoint] = m.currTopMap[msg.endPoint]
		m.currTopMap[msg.endPoint] = msg
		if msg.final {
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

func (m *topNetUI) calculationRate(prev, curr uint64, dur time.Duration) uint64 {
	if curr < prev {
		return uint64(float64(math.MaxUint64-prev+curr) / dur.Seconds())
	}
	return uint64(float64(curr-prev) / dur.Seconds())
}

func (m *topNetUI) View() string {
	var s strings.Builder
	// Set table header
	table := tablewriter.NewWriter(&s)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)
	table.SetHeader([]string{"SERVER", "INTERFACE", "RECEIVE", "TRANSMIT", ""})

	data := make([]topNetResult, 0, len(m.currTopMap))

	for endPoint, curr := range m.currTopMap {
		if prev, ok := m.prevTopMap[endPoint]; ok {
			data = append(data, topNetResult{
				final:    curr.final,
				endPoint: curr.endPoint,
				error:    curr.error,
				stats: madmin.NetMetrics{
					CollectedAt:   curr.stats.CollectedAt,
					InterfaceName: curr.stats.InterfaceName,
					NetStats: procfs.NetDevLine{
						RxBytes: m.calculationRate(prev.stats.NetStats.RxBytes, curr.stats.NetStats.RxBytes, curr.stats.CollectedAt.Sub(prev.stats.CollectedAt)),
						TxBytes: m.calculationRate(prev.stats.NetStats.TxBytes, curr.stats.NetStats.TxBytes, curr.stats.CollectedAt.Sub(prev.stats.CollectedAt)),
					},
				},
			})
		}
	}

	sort.Slice(data, func(i, j int) bool {
		if m.sortAsc {
			return data[i].GetTotalBytes() < data[j].GetTotalBytes()
		}
		return data[i].GetTotalBytes() >= data[j].GetTotalBytes()
	})

	dataRender := make([][]string, 0, len(data))
	for _, d := range data {
		if d.error == "" {
			dataRender = append(dataRender, []string{
				d.endPoint,
				whiteStyle.Render(d.stats.InterfaceName),
				whiteStyle.Render(fmt.Sprintf("%s/s", humanize.IBytes(d.stats.NetStats.RxBytes))),
				whiteStyle.Render(fmt.Sprintf("%s/s", humanize.IBytes(d.stats.NetStats.TxBytes))),
				"",
			})
		} else {
			dataRender = append(dataRender, []string{
				d.endPoint,
				whiteStyle.Render(d.stats.NetStats.Name),
				crossTickCell,
				crossTickCell,
				d.error,
			})
		}
	}

	table.AppendBulk(dataRender)
	table.Render()
	return s.String()
}

func initTopNetUI() *topNetUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &topNetUI{
		spinner:    s,
		currTopMap: make(map[string]topNetResult),
		prevTopMap: make(map[string]topNetResult),
	}
}
