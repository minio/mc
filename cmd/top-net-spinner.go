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
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/minio/madmin-go/v3"
	"github.com/olekukonko/tablewriter"
)

type topNetUI struct {
	spinner  spinner.Model
	quitting bool

	sortAsc   bool
	count     int
	endPoints []string

	prevTopMap map[string]*madmin.NetMetrics
	currTopMap map[string]*madmin.NetMetrics
}

type topNetResult struct {
	final    bool
	endPoint string
	stats    madmin.NetMetrics
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
		m.currTopMap[msg.endPoint] = &msg.stats
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
	table.SetHeader([]string{"Host", "EndPoint", "Face", "RECEIVE", "TRANSMIT"})

	var data []*madmin.NetMetrics

	for _, metric := range m.currTopMap {
		data = append(data, metric)
	}

	sort.Slice(data, func(i, j int) bool {
		if m.sortAsc {
			return data[i].NetStats.RxBytes < data[j].NetStats.RxBytes
		}
		return data[i].NetStats.RxBytes >= data[j].NetStats.RxBytes
	})

	if len(data) > m.count {
		data = data[:m.count]
	}

	dataRender := make([][]string, 0, len(data))
	for _, d := range data {
		if d.Host == "" {
			d.Host = "---"
		}
		dataRender = append(dataRender, []string{
			d.Host,
			d.EndPoint,
			whiteStyle.Render(d.InterfaceName),
			whiteStyle.Render(fmt.Sprintf("%s/s", humanize.IBytes(uint64(d.NetStats.RxBytes)))),
			whiteStyle.Render(fmt.Sprintf("%s/s", humanize.IBytes(uint64(d.NetStats.TxBytes)))),
		})
	}
	table.AppendBulk(dataRender)
	table.Render()
	return s.String()
}

func initTopNetUI(endpoint []string, count int) *topNetUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &topNetUI{
		count:      count,
		endPoints:  endpoint,
		spinner:    s,
		prevTopMap: make(map[string]*madmin.NetMetrics),
		currTopMap: make(map[string]*madmin.NetMetrics),
	}
}
