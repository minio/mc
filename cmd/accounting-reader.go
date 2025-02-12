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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/minio/pkg/v3/console"

	"github.com/cheggaaa/pb"
	json "github.com/minio/colorjson"
	"github.com/minio/mc/pkg/probe"
)

// accounter keeps tabs of ongoing data transfer information.
type accounter struct {
	current int64

	total        int64
	startTime    time.Time
	startValue   int64
	refreshRate  time.Duration
	currentValue int64
	finishOnce   sync.Once
	isFinished   chan struct{}
}

// Instantiate a new accounter.
func newAccounter(total int64) *accounter {
	acct := &accounter{
		total:        total,
		startTime:    time.Now(),
		startValue:   0,
		refreshRate:  time.Millisecond * 200,
		isFinished:   make(chan struct{}),
		currentValue: -1,
	}
	go acct.writer()
	return acct
}

// write calculate the final speed.
func (a *accounter) write(current int64) (float64, time.Duration) {
	fromStart := time.Since(a.startTime)
	currentFromStart := current - a.startValue
	if currentFromStart > 0 {
		speed := float64(currentFromStart) / (float64(fromStart) / float64(time.Second))
		return speed, fromStart
	}
	return 0.0, 0
}

// writer update new accounting data for a specified refreshRate.
func (a *accounter) writer() {
	a.Update()
	for {
		select {
		case <-a.isFinished:
			return
		case <-time.After(a.refreshRate):
			a.Update()
		}
	}
}

// accountStat cantainer for current stats captured.
type accountStat struct {
	Status      string        `json:"status"`
	Total       int64         `json:"total"`
	Transferred int64         `json:"transferred"`
	Duration    time.Duration `json:"duration"`
	Speed       float64       `json:"speed"`
}

func (c accountStat) JSON() string {
	c.Status = "success"
	accountMessageBytes, e := json.MarshalIndent(c, "", " ")
	fatalIf(probe.NewError(e), "Unable to marshal into JSON.")

	return string(accountMessageBytes)
}

func (c accountStat) String() string {
	dspOrder := []col{colGreen} // Header
	dspOrder = append(dspOrder, colGrey)
	var printColors []*color.Color
	for _, c := range dspOrder {
		printColors = append(printColors, getPrintCol(c))
	}

	tbl := console.NewTable(printColors, []bool{false, false, false, false}, 0)

	var builder strings.Builder
	cellText := make([][]string, 0, 2)
	cellText = append(cellText, []string{
		"Total",
		"Transferred",
		"Duration",
		"Speed",
	})

	speedBox := pb.Format(int64(c.Speed)).To(pb.U_BYTES).String()
	if speedBox == "" {
		speedBox = "0 MB"
	} else {
		speedBox = speedBox + "/s"
	}

	cellText = append(cellText, []string{
		pb.Format(c.Total).To(pb.U_BYTES).String(),
		pb.Format(c.Transferred).To(pb.U_BYTES).String(),
		pb.Format(int64(c.Duration)).To(pb.U_DURATION).String(),
		speedBox,
	})

	e := tbl.PopulateTable(&builder, cellText)
	fatalIf(probe.NewError(e), "unable to populate the table")

	return builder.String()
}

// Stat provides current stats captured.
func (a *accounter) Stat() accountStat {
	var acntStat accountStat
	a.finishOnce.Do(func() {
		close(a.isFinished)
		acntStat.Total = a.total
		acntStat.Transferred = atomic.LoadInt64(&a.current)
		acntStat.Speed, acntStat.Duration = a.write(atomic.LoadInt64(&a.current))
	})
	return acntStat
}

// Update update with new values loaded atomically.
func (a *accounter) Update() {
	c := atomic.LoadInt64(&a.current)
	if c != a.currentValue {
		a.write(c)
		a.currentValue = c
	}
}

// Set sets the current value atomically.
func (a *accounter) Set(n int64) *accounter {
	atomic.StoreInt64(&a.current, n)
	return a
}

// Get gets current value atomically
func (a *accounter) Get() int64 {
	return atomic.LoadInt64(&a.current)
}

func (a *accounter) SetTotal(n int64) {
	atomic.StoreInt64(&a.total, n)
}

// Add add to current value atomically.
func (a *accounter) Add(n int64) int64 {
	return atomic.AddInt64(&a.current, n)
}

// Read implements Reader which internally updates current value.
func (a *accounter) Read(p []byte) (n int, err error) {
	defer func() {
		// Upload retry can read one object twice; Avoid read to be greater than Total
		if n, t := a.Get(), atomic.LoadInt64(&a.total); t > 0 && n > t {
			a.Set(t)
		}
	}()

	n = len(p)
	a.Add(int64(n))
	return
}
