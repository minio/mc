// Copyright (c) 2022 MinIO, Inc.
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
	"strconv"

	humanize "github.com/dustin/go-humanize"
	"github.com/gdamore/tcell/v2"
	"github.com/minio/madmin-go"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
)

func quitOnKeys(app *tview.Application) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		var stop bool
		switch event.Key() {
		case tcell.KeyESC, tcell.KeyEnter:
			stop = true
		case tcell.KeyRune:
			if r := event.Rune(); r == 'q' || r == ' ' {
				stop = true
			}
		}
		if stop {
			app.Stop()
		}
		return event
	}
}

func (ts tierInfos) TableUI() *tview.Table {
	table := tview.NewTable().
		SetBorders(true)
	columnHdrs := []string{"Name", "API", "Type", "Usage", "Objects", "Versions"}
	for i, colHdr := range columnHdrs {
		table.SetCell(0, i,
			tview.NewTableCell(colHdr).
				SetTextColor(tcell.ColorYellow).
				SetAlign(tview.AlignCenter))
	}

	tierType := func(tierName string) string {
		if tierName == "STANDARD" {
			return "hot"
		}
		return "warm"
	}
	for i, tInfo := range ts {
		table.SetCell(i+1, 0,
			tview.NewTableCell(tInfo.Name).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
		table.SetCell(i+1, 1,
			tview.NewTableCell(tInfo.Type).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
		table.SetCell(i+1, 2,
			tview.NewTableCell(tierType(tInfo.Name)).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
		table.SetCell(i+1, 3,
			tview.NewTableCell(humanize.IBytes(uint64(tInfo.Stats.TotalSize))).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
		table.SetCell(i+1, 4,
			tview.NewTableCell(strconv.Itoa(tInfo.Stats.NumObjects)).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))
		table.SetCell(i+1, 5,
			tview.NewTableCell(strconv.Itoa(tInfo.Stats.NumVersions)).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter))

	}
	return table
}

func (ts tierInfos) Barcharts(tier string) (objects *tvxwidgets.BarChart, versions *tvxwidgets.BarChart) {
	var maxObj int
	var maxVer int
	for _, t := range ts {
		if maxObj < t.Stats.NumObjects {
			maxObj = t.Stats.NumObjects
		}
		if maxVer < t.Stats.NumVersions {
			maxVer = t.Stats.NumVersions
		}
	}

	objects = tvxwidgets.NewBarChart()
	objects.SetBorder(true)
	objects.SetTitle("Tier stats - Objects")
	objects.SetMaxValue(maxObj)

	versions = tvxwidgets.NewBarChart()
	versions.SetBorder(true)
	versions.SetTitle("Tier stats - Versions")
	versions.SetMaxValue(maxVer)

	var tInfo madmin.TierInfo
	for _, t := range ts {
		if t.Name == tier {
			tInfo = t
			break
		}
	}
	if tInfo.Name == "" {
		return nil, nil
	}

	dailyStats := tInfo.DailyStats
	lastIdx := dailyStats.UpdatedAt.Hour()
	hrs := 23
	for i := 1; i <= 24; i++ {
		if hrs == 0 {
			objects.AddBar(fmt.Sprintf("now"), dailyStats.Bins[(lastIdx+i)%24].NumObjects, tcell.ColorRed)
			versions.AddBar(fmt.Sprintf("now"), dailyStats.Bins[(lastIdx+i)%24].NumVersions, tcell.ColorBlue)

		} else {
			objects.AddBar(fmt.Sprintf("-%d", hrs), dailyStats.Bins[(lastIdx+i)%24].NumObjects, tcell.ColorRed)
			versions.AddBar(fmt.Sprintf("-%d", hrs), dailyStats.Bins[(lastIdx+i)%24].NumVersions, tcell.ColorBlue)
		}
		hrs--
	}

	return objects, versions
}
