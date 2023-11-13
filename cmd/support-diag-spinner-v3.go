// Copyright (c) 2015-2023 MinIO, Inc.
//
// # This file is part of MinIO Object Storage stack
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
	"io"
	"sync"
	"time"

	json "github.com/minio/colorjson"
	"github.com/minio/madmin-go/v3"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

func receiveHealthInfo(decoder *json.Decoder) (info madmin.HealthInfo, e error) {
	var wg sync.WaitGroup
	pg := mpb.New(mpb.WithWaitGroup(&wg), mpb.WithWidth(16))

	spinnerStyle := mpb.SpinnerStyle().Meta(func(s string) string {
		return infoText(s)
	}).PositionLeft()

	type progressSpinner struct {
		name    string
		spinner *mpb.Bar
		cond    func(madmin.HealthInfo) bool
	}
	spinners := []progressSpinner{}

	createSpinner := func(name string, cond func(madmin.HealthInfo) bool) {
		caption := fmt.Sprintf("%s %s ...", dot, name)
		wip := mpb.PrependDecorators(decor.Name(greenText(caption)))
		done := mpb.BarFillerOnComplete(greenText(check))
		spinner := pg.New(1, spinnerStyle, wip, done)
		spinners = append(spinners, progressSpinner{
			name:    name,
			cond:    cond,
			spinner: spinner,
		})
	}

	createSpinner("CPU Info", func(info madmin.HealthInfo) bool { return len(info.Sys.CPUInfo) > 0 })
	createSpinner("Disk Info", func(info madmin.HealthInfo) bool { return len(info.Sys.Partitions) > 0 })
	createSpinner("Net Info", func(info madmin.HealthInfo) bool { return len(info.Sys.NetInfo) > 0 })
	createSpinner("OS Info", func(info madmin.HealthInfo) bool { return len(info.Sys.OSInfo) > 0 })
	createSpinner("Mem Info", func(info madmin.HealthInfo) bool { return len(info.Sys.MemInfo) > 0 })
	createSpinner("Process Info", func(info madmin.HealthInfo) bool { return len(info.Sys.ProcInfo) > 0 })
	createSpinner("Server Config", func(info madmin.HealthInfo) bool { return info.Minio.Config.Config != nil })
	createSpinner("System Errors", func(info madmin.HealthInfo) bool { return len(info.Sys.SysErrs) > 0 })
	createSpinner("System Services", func(info madmin.HealthInfo) bool { return len(info.Sys.SysServices) > 0 })
	createSpinner("System Config", func(info madmin.HealthInfo) bool { return len(info.Sys.SysConfig) > 0 })
	createSpinner("Admin Info", func(info madmin.HealthInfo) bool { return len(info.Minio.Info.Servers) > 0 })

	wg.Add(len(spinners))
	start := time.Now()

	markDone := func(bar *mpb.Bar) {
		if bar.Current() == 0 {
			bar.EwmaIncrement(time.Since(start))
			wg.Done()
		}
	}

	receivedLast := false
	progress := func(info madmin.HealthInfo) {
		receivedLast = len(info.Minio.Info.Servers) > 0

		for _, bar := range spinners {
			if receivedLast || bar.cond(info) {
				markDone(bar.spinner)
			}
		}
	}

	go func() {
		for {
			if e = decoder.Decode(&info); e != nil {
				if errors.Is(e, io.EOF) {
					e = nil
				}

				break
			}

			progress(info)
		}
	}()
	pg.Wait()
	return
}
