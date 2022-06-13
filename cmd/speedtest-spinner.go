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
	humanize "github.com/dustin/go-humanize"
	"github.com/minio/madmin-go"
	"github.com/olekukonko/tablewriter"
)

var whiteStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#ffffff"))

type speedTestUI struct {
	spinner  spinner.Model
	quitting bool
	result   speedTestResult
}

type speedTestResult struct {
	final   bool
	result  *madmin.SpeedTestResult
	nresult *madmin.NetperfResult
}

func initSpeedTestUI() *speedTestUI {
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &speedTestUI{
		spinner: s,
	}
}

func (m *speedTestUI) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *speedTestUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}
	case speedTestResult:
		m.result = msg
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

func (m *speedTestUI) View() string {
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

	res := m.result.result
	nres := m.result.nresult

	if res != nil {
		table.SetHeader([]string{"", "Throughput", "IOPS"})
		data := make([][]string, 2)

		if res.Version == "" {
			data[0] = []string{
				"PUT",
				whiteStyle.Render("-- KiB/sec"),
				whiteStyle.Render("-- objs/sec"),
			}
			data[1] = []string{
				"GET",
				whiteStyle.Render("-- KiB/sec"),
				whiteStyle.Render("-- objs/sec"),
			}
		} else {
			data[0] = []string{
				"PUT",
				whiteStyle.Render(humanize.IBytes(res.PUTStats.ThroughputPerSec) + "/s"),
				whiteStyle.Render(humanize.Comma(int64(res.PUTStats.ObjectsPerSec)) + " objs/s"),
			}
			data[1] = []string{
				"GET",
				whiteStyle.Render(humanize.IBytes(res.GETStats.ThroughputPerSec) + "/s"),
				whiteStyle.Render(humanize.Comma(int64(res.GETStats.ObjectsPerSec)) + " objs/s"),
			}
		}
		table.AppendBulk(data)
		table.Render()

		if m.quitting {
			s.WriteString(fmt.Sprintf("\nSpeedtest: %s", m.result.String()))
			if vstr := m.result.StringVerbose(); vstr != "" {
				s.WriteString(vstr)
			} else {
				s.WriteString("\n")
			}
		}
	} else if nres != nil {
		table.SetHeader([]string{"Node", "RX", "TX", ""})
		data := make([][]string, 0, len(nres.NodeResults))

		for _, nodeResult := range nres.NodeResults {
			if nodeResult.Error != "" {
				data = append(data, []string{
					nodeResult.Endpoint,
					"✗",
					"✗",
					"Err: " + nodeResult.Error,
				})
			} else {
				data = append(data, []string{
					nodeResult.Endpoint,
					humanize.IBytes(uint64(nodeResult.RX)),
					humanize.IBytes(uint64(nodeResult.TX)),
					"✔",
				})
			}
		}

		sort.Slice(data, func(i, j int) bool {
			return data[i][0] < data[j][0]
		})

		table.AppendBulk(data)
		table.Render()

		if m.quitting {
			s.WriteString("\nNetperf: ✔\n")
		}
	}
	if !m.quitting {
		if nres != nil {
			s.WriteString(fmt.Sprintf("\nNetperf: %s", m.spinner.View()))
		} else if res != nil {
			s.WriteString(fmt.Sprintf("\nSpeedtest: %s", m.spinner.View()))
		}
	}
	return s.String()
}
