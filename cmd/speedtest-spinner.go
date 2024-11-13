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
	"time"

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
	SiteReplicationPerfTest
	ClientPerfTest
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
	case SiteReplicationPerfTest:
		return "SiteReplication"
	case ClientPerfTest:
		return "Client"
	}
	return "<unknown>"
}

// PerfTestResult - stores the result of a performance test
type PerfTestResult struct {
	Type                  PerfTestType                  `json:"type"`
	ObjectResult          *madmin.SpeedTestResult       `json:"object,omitempty"`
	NetResult             *madmin.NetperfResult         `json:"network,omitempty"`
	SiteReplicationResult *madmin.SiteNetPerfResult     `json:"siteReplication,omitempty"`
	ClientResult          *madmin.ClientPerfResult      `json:"client,omitempty"`
	DriveResult           []madmin.DriveSpeedTestResult `json:"drive,omitempty"`
	Err                   string                        `json:"err,omitempty"`
	Final                 bool                          `json:"final,omitempty"`
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
	sres := m.result.SiteReplicationResult
	dres := m.result.DriveResult
	cres := m.result.ClientResult

	trailerIfGreaterThan := func(in string, maxIdx int) string {
		if len(in) < maxIdx {
			return in
		}
		return in[:maxIdx] + "..."
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
				nodeErr := ""
				if nodeResult.Error != "" {
					nodeErr = "Err: " + nodeResult.Error
				}
				data = append(data, []string{
					trailerIfGreaterThan(nodeResult.Endpoint, 64),
					whiteStyle.Render(humanize.IBytes(uint64(nodeResult.RX))) + "/s",
					whiteStyle.Render(humanize.IBytes(uint64(nodeResult.TX))) + "/s",
					nodeErr,
				})
			}
		}

		sort.Slice(data, func(i, j int) bool {
			return data[i][0] < data[j][0]
		})

		table.AppendBulk(data)
		table.Render()
	} else if sres != nil {
		table.SetHeader([]string{"Endpoint", "RX", "TX", ""})
		data := make([][]string, 0, len(sres.NodeResults))
		if len(sres.NodeResults) == 0 {
			data = append(data, []string{
				"...",
				whiteStyle.Render("-- MiB"),
				whiteStyle.Render("-- MiB"),
				"",
			})
		} else {
			for _, nodeResult := range sres.NodeResults {
				if nodeResult.Error != "" {
					data = append(data, []string{
						trailerIfGreaterThan(nodeResult.Endpoint, 64),
						crossTickCell,
						crossTickCell,
						"Err: " + nodeResult.Error,
					})
				} else {
					dataItem := []string{}
					dataError := ""
					// show endpoint
					dataItem = append(dataItem, trailerIfGreaterThan(nodeResult.Endpoint, 64))
					// show RX
					if uint64(nodeResult.RXTotalDuration.Seconds()) == 0 {
						dataError += "- RXTotalDuration are zero "
						dataItem = append(dataItem, crossTickCell)
					} else {
						dataItem = append(dataItem, whiteStyle.Render(humanize.IBytes(nodeResult.RX/uint64(nodeResult.RXTotalDuration.Seconds())))+"/s")
					}
					// show TX
					if uint64(nodeResult.TXTotalDuration.Seconds()) == 0 {
						dataError += "- TXTotalDuration are zero"
						dataItem = append(dataItem, crossTickCell)
					} else {
						dataItem = append(dataItem, whiteStyle.Render(humanize.IBytes(nodeResult.TX/uint64(nodeResult.TXTotalDuration.Seconds())))+"/s")
					}
					// show message
					dataItem = append(dataItem, dataError)
					data = append(data, dataItem)
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
	} else if cres != nil {
		table.SetHeader([]string{"Endpoint", "Tx"})
		data := make([][]string, 0, 2)
		tx := uint64(0)
		if cres.TimeSpent > 0 {
			tx = uint64(float64(cres.BytesSend) / time.Duration(cres.TimeSpent).Seconds())
		}
		if tx == 0 {
			data = append(data, []string{
				"...",
				whiteStyle.Render("-- KiB/s"),
				"",
			})
		} else {
			data = append(data, []string{
				cres.Endpoint,
				whiteStyle.Render(humanize.IBytes(tx)) + "/s",
				cres.Error,
			})
		}
		table.AppendBulk(data)
		table.Render()
	}

	return s.String()
}
