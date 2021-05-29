// Copyright (c) 2015-2021 MinIO, Inc.
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
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/fatih/color"
	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go"
	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

const (
	lineWidth = 80
)

var (
	hColOrder = []col{colRed, colYellow, colGreen}
	hColTable = map[int][]int{
		1: {0, -1, 1},
		2: {0, 1, 2},
		3: {1, 2, 3},
		4: {1, 2, 4},
		5: {1, 3, 5},
		6: {2, 4, 6},
		7: {2, 4, 7},
		8: {2, 5, 8},
	}
)

func getHColCode(surplusShards, parityShards int) (c col, err error) {
	if parityShards < 1 || parityShards > 8 || surplusShards > parityShards {
		return c, fmt.Errorf("Invalid parity shard count/surplus shard count given")
	}
	if surplusShards < 0 {
		return colGrey, err
	}
	colRow := hColTable[parityShards]
	for index, val := range colRow {
		if val != -1 && surplusShards <= val {
			return hColOrder[index], err
		}
	}
	return c, fmt.Errorf("cannot get a heal color code")
}

type uiData struct {
	Bucket, Prefix string
	Client         *madmin.AdminClient
	ClientToken    string
	ForceStart     bool
	HealOpts       *madmin.HealOpts
	LastItem       *hri

	// Total time since heal start
	HealDuration time.Duration

	// Accumulated statistics of heal result records
	BytesScanned int64

	// Counter for objects, and another counter for all kinds of
	// items
	ObjectsScanned, ItemsScanned int64

	// Counters for healed objects and all kinds of healed items
	ObjectsHealed, ItemsHealed int64

	// Map from online drives to number of objects with that many
	// online drives.
	ObjectsByOnlineDrives map[int]int64
	// Map of health color code to number of objects with that
	// health color code.
	HealthCols map[col]int64

	// channel to receive a prompt string to indicate activity on
	// the terminal
	CurChan (<-chan string)
}

func (ui *uiData) updateStats(i madmin.HealResultItem) error {
	if i.Type == madmin.HealItemObject {
		// Objects whose size could not be found have -1 size
		// returned.
		if i.ObjectSize >= 0 {
			ui.BytesScanned += i.ObjectSize
		}

		ui.ObjectsScanned++
	}
	ui.ItemsScanned++

	beforeUp, afterUp := i.GetOnlineCounts()
	if afterUp > beforeUp {
		if i.Type == madmin.HealItemObject {
			ui.ObjectsHealed++
		}
		ui.ItemsHealed++
	}
	ui.ObjectsByOnlineDrives[afterUp]++

	// Update health color stats:

	// Fetch health color after heal:
	var err error
	var afterCol col
	h := newHRI(&i)
	switch h.Type {
	case madmin.HealItemMetadata, madmin.HealItemBucket:
		_, afterCol, err = h.getReplicatedFileHCCChange()
	default:
		_, afterCol, err = h.getObjectHCCChange()
	}
	if err != nil {
		return err
	}

	ui.HealthCols[afterCol]++
	return nil
}

func (ui *uiData) updateDuration(s *madmin.HealTaskStatus) {
	ui.HealDuration = UTCNow().Sub(s.StartTime)
}

func (ui *uiData) getProgress() (oCount, objSize, duration string) {
	oCount = humanize.Comma(ui.ObjectsScanned)

	duration = ui.HealDuration.Round(time.Second).String()

	bytesScanned := float64(ui.BytesScanned)

	// Compute unit for object size
	magnitudes := []float64{1 << 10, 1 << 20, 1 << 30, 1 << 40, 1 << 50, 1 << 60}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	var i int
	for i = 0; i < len(magnitudes); i++ {
		if bytesScanned <= magnitudes[i] {
			break
		}
	}
	numUnits := int(bytesScanned * (1 << 10) / magnitudes[i])
	objSize = fmt.Sprintf("%d %s", numUnits, units[i])
	return
}

func (ui *uiData) getPercentsNBars() (p map[col]float64, b map[col]string) {
	// barChar, emptyBarChar := "█", "░"
	barChar, emptyBarChar := "█", " "
	barLen := 12
	sum := float64(ui.ItemsScanned)
	cols := []col{colGrey, colRed, colYellow, colGreen}

	p = make(map[col]float64, len(cols))
	b = make(map[col]string, len(cols))
	var filledLen int
	for _, col := range cols {
		v := float64(ui.HealthCols[col])
		if sum == 0 {
			p[col] = 0
			filledLen = 0
		} else {
			p[col] = v * 100 / sum
			// round up the filled part
			filledLen = int(math.Ceil(float64(barLen) * v / sum))
		}
		b[col] = strings.Repeat(barChar, filledLen) +
			strings.Repeat(emptyBarChar, barLen-filledLen)
	}
	return
}

func (ui *uiData) printItemsQuietly(s *madmin.HealTaskStatus) (err error) {
	lpad := func(s col) string {
		return fmt.Sprintf("%-6s", string(s))
	}
	rpad := func(s col) string {
		return fmt.Sprintf("%6s", string(s))
	}
	printColStr := func(before, after col) {
		console.PrintC("[" + lpad(before) + " -> " + rpad(after) + "] ")
	}

	var b, a col
	for _, item := range s.Items {
		h := newHRI(&item)
		switch h.Type {
		case madmin.HealItemMetadata, madmin.HealItemBucket:
			b, a, err = h.getReplicatedFileHCCChange()
		default:
			b, a, err = h.getObjectHCCChange()
		}
		if err != nil {
			return err
		}
		printColStr(b, a)
		hrStr := h.getHealResultStr()
		switch h.Type {
		case madmin.HealItemMetadata, madmin.HealItemBucketMetadata:
			console.PrintC(fmt.Sprintln("**", hrStr, "**"))
		default:
			console.PrintC(hrStr, "\n")
		}
	}
	return nil
}

func (ui *uiData) printStatsQuietly(s *madmin.HealTaskStatus) {
	totalObjects, totalSize, totalTime := ui.getProgress()

	healedStr := fmt.Sprintf("Healed:\t%s/%s objects; %s in %s\n",
		humanize.Comma(ui.ObjectsHealed), totalObjects,
		totalSize, totalTime)

	console.PrintC(healedStr)
}

func (ui *uiData) printItemsJSON(s *madmin.HealTaskStatus) (err error) {
	type healRec struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
		Type   string `json:"type"`
		Name   string `json:"name"`
		Before struct {
			Color     string                 `json:"color"`
			Offline   int                    `json:"offline"`
			Online    int                    `json:"online"`
			Missing   int                    `json:"missing"`
			Corrupted int                    `json:"corrupted"`
			Drives    []madmin.HealDriveInfo `json:"drives"`
		} `json:"before"`
		After struct {
			Color     string                 `json:"color"`
			Offline   int                    `json:"offline"`
			Online    int                    `json:"online"`
			Missing   int                    `json:"missing"`
			Corrupted int                    `json:"corrupted"`
			Drives    []madmin.HealDriveInfo `json:"drives"`
		} `json:"after"`
		Size int64 `json:"size"`
	}
	makeHR := func(h *hri) (r healRec, err error) {
		r.Status = "success"
		r.Type, r.Name = h.getHRTypeAndName()

		var b, a col
		switch h.Type {
		case madmin.HealItemMetadata, madmin.HealItemBucket:
			b, a, err = h.getReplicatedFileHCCChange()
		default:
			if h.Type == madmin.HealItemObject {
				r.Size = h.ObjectSize
			}
			b, a, err = h.getObjectHCCChange()
		}
		if err != nil {
			return r, err
		}
		r.Before.Color = strings.ToLower(string(b))
		r.After.Color = strings.ToLower(string(a))
		r.Before.Online, r.After.Online = h.GetOnlineCounts()
		r.Before.Missing, r.After.Missing = h.GetMissingCounts()
		r.Before.Corrupted, r.After.Corrupted = h.GetCorruptedCounts()
		r.Before.Offline, r.After.Offline = h.GetOfflineCounts()
		r.Before.Drives = h.Before.Drives
		r.After.Drives = h.After.Drives
		return r, nil
	}

	for _, item := range s.Items {
		h := newHRI(&item)
		r, err := makeHR(h)
		if err != nil {
			return err
		}
		jsonBytes, err := json.MarshalIndent(r, "", " ")
		fatalIf(probe.NewError(err), "Unable to marshal to JSON.")
		console.Println(string(jsonBytes))
	}
	return nil
}

func (ui *uiData) printStatsJSON(s *madmin.HealTaskStatus) {
	var summary struct {
		Status         string `json:"status"`
		Error          string `json:"error,omitempty"`
		Type           string `json:"type"`
		ObjectsScanned int64  `json:"objects_scanned"`
		ObjectsHealed  int64  `json:"objects_healed"`
		ItemsScanned   int64  `json:"items_scanned"`
		ItemsHealed    int64  `json:"items_healed"`
		Size           int64  `json:"size"`
		ElapsedTime    int64  `json:"duration"`
	}

	summary.Status = "success"
	summary.Type = "summary"

	summary.ObjectsScanned = ui.ObjectsScanned
	summary.ObjectsHealed = ui.ObjectsHealed
	summary.ItemsScanned = ui.ItemsScanned
	summary.ItemsHealed = ui.ItemsHealed
	summary.Size = ui.BytesScanned
	summary.ElapsedTime = int64(ui.HealDuration.Round(time.Second).Seconds())

	jBytes, err := json.MarshalIndent(summary, "", " ")
	fatalIf(probe.NewError(err), "Unable to marshal to JSON.")
	console.Println(string(jBytes))
}

func (ui *uiData) updateUI(s *madmin.HealTaskStatus) (err error) {
	itemCount := len(s.Items)
	h := ui.LastItem
	if itemCount > 0 {
		item := s.Items[itemCount-1]
		h = newHRI(&item)
		ui.LastItem = h
	}
	scannedStr := "** waiting for status from server **"
	if h != nil {
		scannedStr = lineTrunc(h.makeHealEntityString(), lineWidth-len("Scanned: "))
	}

	totalObjects, totalSize, totalTime := ui.getProgress()
	healedStr := fmt.Sprintf("%s/%s objects; %s in %s",
		humanize.Comma(ui.ObjectsHealed), totalObjects,
		totalSize, totalTime)

	console.Print(console.Colorize("HealUpdateUI", fmt.Sprintf(" %s", <-ui.CurChan)))
	console.PrintC(fmt.Sprintf("  %s\n", scannedStr))
	console.PrintC(fmt.Sprintf("    %s\n", healedStr))

	dspOrder := []col{colGreen, colYellow, colRed, colGrey}
	printColors := []*color.Color{}
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}
	t := console.NewTable(printColors, []bool{false, true, true}, 4)

	percentMap, barMap := ui.getPercentsNBars()
	cellText := make([][]string, len(dspOrder))
	for i := range cellText {
		cellText[i] = []string{
			string(dspOrder[i]),
			fmt.Sprint(humanize.Comma(ui.HealthCols[dspOrder[i]])),
			fmt.Sprintf("%5.1f%% %s", percentMap[dspOrder[i]], barMap[dspOrder[i]]),
		}
	}

	t.DisplayTable(cellText)
	return nil
}

func (ui *uiData) UpdateDisplay(s *madmin.HealTaskStatus) (err error) {
	// Update state
	ui.updateDuration(s)
	for _, i := range s.Items {
		ui.updateStats(i)
	}

	// Update display
	switch {
	case globalJSON:
		err = ui.printItemsJSON(s)
	case globalQuiet:
		err = ui.printItemsQuietly(s)
	default:
		err = ui.updateUI(s)
	}
	return
}

func (ui *uiData) healResumeMsg(aliasedURL string) string {
	var flags string
	if ui.HealOpts.Recursive {
		flags += "--recursive "
	}
	if ui.HealOpts.DryRun {
		flags += "--dry-run "
	}
	return fmt.Sprintf("Healing is backgrounded, to resume watching use `mc admin heal %s %s`", flags, aliasedURL)
}

func (ui *uiData) DisplayAndFollowHealStatus(aliasedURL string) (res madmin.HealTaskStatus, err error) {
	quitMsg := ui.healResumeMsg(aliasedURL)

	firstIter := true
	for {
		select {
		case <-globalContext.Done():
			return res, errors.New(quitMsg)
		default:
			_, res, err = ui.Client.Heal(globalContext, ui.Bucket, ui.Prefix, *ui.HealOpts,
				ui.ClientToken, ui.ForceStart, false)
			if err != nil {
				return res, err
			}
			if firstIter {
				firstIter = false
			} else {
				if !globalQuiet && !globalJSON {
					console.RewindLines(8)
				}
			}
			err = ui.UpdateDisplay(&res)
			if err != nil {
				return res, err
			}

			if res.Summary == "finished" {
				if globalJSON {
					ui.printStatsJSON(&res)
				} else if globalQuiet {
					ui.printStatsQuietly(&res)
				}
				return res, nil
			}

			if res.Summary == "stopped" {
				return res, fmt.Errorf("Heal had an error - %s", res.FailureDetail)
			}

			time.Sleep(time.Second)
		}
	}
}
