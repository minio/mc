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
	"github.com/minio/madmin-go/v3"
	"github.com/olekukonko/tablewriter"
)

var whiteStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#ffffff"))

type speedTestUI struct {
	spinner  spinner.Model
	quitting bool
	result   PerfTestResult
}

// PerfTestType - The type of performance test (net/drive/object)
type PerfTestType byte

// Constants for performance test type
const (
	NetPerfTest PerfTestType = 1 << iota
	DrivePerfTest
	ObjectPerfTest
)

// Name - returns name of the performance test
func (p PerfTestType) Name() string {
	switch p {
	case NetPerfTest:
		return "NetPerf"
	case DrivePerfTest:
		return "DrivePerf"
	case ObjectPerfTest:
		return "ObjectPerf"
	}
	return "<unknown>"
}

// PerfTestResult - stores the result of a performance test
type PerfTestResult struct {
	Type         PerfTestType                  `json:"type"`
	ObjectResult *madmin.SpeedTestResult       `json:"object,omitempty"`
	NetResult    *madmin.NetperfResult         `json:"network,omitempty"`
	DriveResult  []madmin.DriveSpeedTestResult `json:"drive,omitempty"`
	Err          string                        `json:"err,omitempty"`
	Final        bool                          `json:"final,omitempty"`
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
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}
	case PerfTestResult:
		m.result = msg
		if msg.Final {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *speedTestUI) View() string {
	// Quit when there is an error
	if m.result.Err != "" {
		return fmt.Sprintf("\n%s: %s (Err: %s)\n", m.result.Type.Name(), crossTickCell, m.result.Err)
	}

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

	ores := m.result.ObjectResult
	nres := m.result.NetResult
	dres := m.result.DriveResult

	trailerIfGreaterThan := func(in string, max int) string {
		if len(in) < max {
			return in
		}
		return in[:max] + "..."
	}

	// Print the spinner
	if !m.quitting {
		s.WriteString(fmt.Sprintf("\n%s: %s\n\n", m.result.Type.Name(), m.spinner.View()))
	} else {
		s.WriteString(fmt.Sprintf("\n%s: %s\n\n", m.result.Type.Name(), m.spinner.Style.Render(tickCell)))
	}

	if ores != nil {
		table.SetHeader([]string{"", "Throughput", "IOPS"})
		data := make([][]string, 2)

		if ores.Version == "" {
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
				whiteStyle.Render(humanize.IBytes(ores.PUTStats.ThroughputPerSec) + "/s"),
				whiteStyle.Render(humanize.Comma(int64(ores.PUTStats.ObjectsPerSec)) + " objs/s"),
			}
			data[1] = []string{
				"GET",
				whiteStyle.Render(humanize.IBytes(ores.GETStats.ThroughputPerSec) + "/s"),
				whiteStyle.Render(humanize.Comma(int64(ores.GETStats.ObjectsPerSec)) + " objs/s"),
			}
		}
		table.AppendBulk(data)
		table.Render()

		if m.quitting {
			s.WriteString("\n" + objectTestShortResult(ores))
			if globalPerfTestVerbose {
				s.WriteString("\n\n")
				s.WriteString(objectTestVerboseResult(ores))
			}
			s.WriteString("\n")
		}
	} else if nres != nil {
		table.SetHeader([]string{"Node", "RX", "TX", ""})
		data := make([][]string, 0, len(nres.NodeResults))

		if len(nres.NodeResults) == 0 {
			data = append(data, []string{
				"...",
				whiteStyle.Render("-- MiB/s"),
				whiteStyle.Render("-- MiB/s"),
				"",
			})
		} else {
			for _, nodeResult := range nres.NodeResults {
				if nodeResult.Error != "" {
					data = append(data, []string{
						trailerIfGreaterThan(nodeResult.Endpoint, 64),
						crossTickCell,
						crossTickCell,
						"Err: " + nodeResult.Error,
					})
				} else {
					data = append(data, []string{
						trailerIfGreaterThan(nodeResult.Endpoint, 64),
						whiteStyle.Render(humanize.IBytes(uint64(nodeResult.RX))) + "/s",
						whiteStyle.Render(humanize.IBytes(uint64(nodeResult.TX))) + "/s",
						"",
					})
				}
			}
		}

		sort.Slice(data, func(i, j int) bool {
			return data[i][0] < data[j][0]
		})

		table.AppendBulk(data)
		table.Render()
	} else if dres != nil {
		table.SetHeader([]string{"Node", "Path", "Read", "Write", ""})
		data := make([][]string, 0, len(dres))

		if len(dres) == 0 {
			data = append(data, []string{
				"...",
				"...",
				whiteStyle.Render("-- KiB/s"),
				whiteStyle.Render("-- KiB/s"),
				"",
			})
		} else {
			for _, driveResult := range dres {
				for _, result := range driveResult.DrivePerf {
					if result.Error != "" {
						data = append(data, []string{
							trailerIfGreaterThan(driveResult.Endpoint, 64),
							result.Path,
							crossTickCell,
							crossTickCell,
							"Err: " + result.Error,
						})
					} else {
						data = append(data, []string{
							trailerIfGreaterThan(driveResult.Endpoint, 64),
							result.Path,
							whiteStyle.Render(humanize.IBytes(result.ReadThroughput)) + "/s",
							whiteStyle.Render(humanize.IBytes(result.WriteThroughput)) + "/s",
							"",
						})
					}
				}
			}
		}
		table.AppendBulk(data)
		table.Render()
	}

	return s.String()
}
