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
	"cmp"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/minio/madmin-go/v3"
	"github.com/olekukonko/tablewriter"
)

type topDriveUI struct {
	spinner  spinner.Model
	quitting bool

	sortBy        drivesSorter
	sortAsc       bool
	count         int
	pool, maxPool int

	drivesInfo map[string]madmin.Disk

	prevTopMap map[string]madmin.DiskIOStats
	currTopMap map[string]madmin.DiskIOStats
}

type topDriveResult struct {
	final    bool
	diskName string
	stats    madmin.DiskIOStats
}

func initTopDriveUI(disks []madmin.Disk, count int) *topDriveUI {
	maxPool := 0
	drivesInfo := make(map[string]madmin.Disk)
	for i := range disks {
		drivesInfo[disks[i].Endpoint] = disks[i]
		if disks[i].PoolIndex > maxPool {
			maxPool = disks[i].PoolIndex
		}
	}

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &topDriveUI{
		count:      count,
		sortBy:     sortByName,
		pool:       0,
		maxPool:    maxPool,
		drivesInfo: drivesInfo,
		spinner:    s,
		prevTopMap: make(map[string]madmin.DiskIOStats),
		currTopMap: make(map[string]madmin.DiskIOStats),
	}
}

func (m *topDriveUI) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *topDriveUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "right":
			m.pool++
			if m.pool >= m.maxPool {
				m.pool = m.maxPool
			}
		case "left":
			m.pool--
			if m.pool < 0 {
				m.pool = 0
			}
		case "u":
			m.sortBy = sortByUsed
		case "t":
			m.sortBy = sortByTps
		case "r":
			m.sortBy = sortByRead
		case "w":
			m.sortBy = sortByWrite
		case "d":
			m.sortBy = sortByDiscard
		case "a":
			m.sortBy = sortByAwait
		case "U":
			m.sortBy = sortByUtil
		case "o", "O":
			m.sortAsc = !m.sortAsc
		}

		return m, nil
	case topDriveResult:
		m.prevTopMap[msg.diskName] = m.currTopMap[msg.diskName]
		m.currTopMap[msg.diskName] = msg.stats
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

type driveIOStat struct {
	endpoint   string
	util       float64
	await      float64
	readMBs    float64
	writeMBs   float64
	discardMBs float64
	tps        uint64
	used       uint64
}

func generateDriveStat(disk madmin.Disk, curr, prev madmin.DiskIOStats, interval uint64) (d driveIOStat) {
	if disk.TotalSpace == 0 {
		return d
	}
	d.endpoint = disk.Endpoint
	d.used = 100 * disk.UsedSpace / disk.TotalSpace
	d.util = 100 * float64(curr.TotalTicks-prev.TotalTicks) / float64(interval)
	currTotalIOs := curr.ReadIOs + curr.WriteIOs + curr.DiscardIOs
	prevTotalIOs := prev.ReadIOs + prev.WriteIOs + prev.DiscardIOs
	totalTicksDiff := curr.ReadTicks - prev.ReadTicks + curr.WriteTicks - prev.WriteTicks + curr.DiscardTicks - prev.DiscardTicks
	if currTotalIOs > prevTotalIOs {
		d.tps = currTotalIOs - prevTotalIOs
		d.await = float64(totalTicksDiff) / float64(currTotalIOs-prevTotalIOs)
	}
	intervalInSec := float64(interval / 1000)
	d.readMBs = float64(curr.ReadSectors-prev.ReadSectors) / (2048 * intervalInSec)
	d.writeMBs = float64(curr.WriteSectors-prev.WriteSectors) / (2048 * intervalInSec)
	d.discardMBs = float64(curr.DiscardSectors-prev.DiscardSectors) / (2048 * intervalInSec)
	return d
}

type drivesSorter int

const (
	sortByName drivesSorter = iota
	sortByUsed
	sortByAwait
	sortByUtil
	sortByRead
	sortByWrite
	sortByDiscard
	sortByTps
)

func (s drivesSorter) String() string {
	switch s {
	case sortByName:
		return "name"
	case sortByUsed:
		return "used"
	case sortByAwait:
		return "await"
	case sortByUtil:
		return "util"
	case sortByRead:
		return "read"
	case sortByWrite:
		return "write"
	case sortByDiscard:
		return "discard"
	case sortByTps:
		return "tps"
	}
	return "unknown"
}

func sortDriveIOStat(sortBy drivesSorter, asc bool, data []driveIOStat) {
	sort.SliceStable(data, func(i, j int) bool {
		c := 0
		switch sortBy {
		case sortByName:
			c = cmp.Compare(data[i].endpoint, data[j].endpoint)
		case sortByUsed:
			c = cmp.Compare(data[i].used, data[j].used)
		case sortByAwait:
			c = cmp.Compare(data[i].await, data[j].await)
		case sortByUtil:
			c = cmp.Compare(data[i].util, data[j].util)
		case sortByRead:
			c = cmp.Compare(data[i].readMBs, data[j].readMBs)
		case sortByWrite:
			c = cmp.Compare(data[i].writeMBs, data[j].writeMBs)
		case sortByDiscard:
			c = cmp.Compare(data[i].discardMBs, data[j].discardMBs)
		case sortByTps:
			c = cmp.Compare(data[i].tps, data[j].tps)
		}

		less := c < 0
		if sortBy != sortByName && !asc {
			less = !less
		}
		return less
	})
}

func (m *topDriveUI) View() string {
	var s strings.Builder
	s.WriteString("\n")

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

	table.SetHeader([]string{"Drive", "used", "tps", "read", "write", "discard", "await", "util"})

	var data []driveIOStat

	for disk := range m.currTopMap {
		currDisk, ok := m.drivesInfo[disk]
		if !ok || currDisk.PoolIndex != m.pool {
			continue
		}
		data = append(data, generateDriveStat(m.drivesInfo[disk], m.currTopMap[disk], m.prevTopMap[disk], 1000))
	}

	sortDriveIOStat(m.sortBy, m.sortAsc, data)

	if len(data) > m.count {
		data = data[:m.count]
	}

	dataRender := make([][]string, 0, len(data))
	for _, d := range data {
		endpoint := d.endpoint
		diskInfo := m.drivesInfo[endpoint]
		if diskInfo.Healing {
			endpoint += "!"
		}
		if diskInfo.Scanning {
			endpoint += "*"
		}
		if diskInfo.TotalSpace == 0 {
			endpoint += crossTickCell
		}

		dataRender = append(dataRender, []string{
			endpoint,
			whiteStyle.Render(fmt.Sprintf("%d%%", d.used)),
			whiteStyle.Render(fmt.Sprintf("%v", d.tps)),
			whiteStyle.Render(fmt.Sprintf("%.2f MiB/s", d.readMBs)),
			whiteStyle.Render(fmt.Sprintf("%.2f MiB/s", d.writeMBs)),
			whiteStyle.Render(fmt.Sprintf("%.2f MiB/s", d.discardMBs)),
			whiteStyle.Render(fmt.Sprintf("%.1f ms", d.await)),
			whiteStyle.Render(fmt.Sprintf("%.1f%%", d.util)),
		})
	}

	table.AppendBulk(dataRender)
	table.Render()

	if !m.quitting {
		order := "DESC"
		if m.sortAsc {
			order = "ASC"
		}

		s.WriteString(fmt.Sprintf("\n%s \u25C0 Pool %d \u25B6 | Sort By: %s (u,t,r,w,d,a,U) | (O)rder: %s ", m.spinner.View(), m.pool+1, m.sortBy, order))
	}
	return s.String() + "\n"
}
