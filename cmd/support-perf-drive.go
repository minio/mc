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
	"context"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

func mainAdminSpeedTestDrive(ctx *cli.Context, aliasedURL string, outCh chan<- PerfTestResult) error {
	client, perr := newAdminClient(aliasedURL)
	if perr != nil {
		fatalIf(perr.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	blocksize, e := humanize.ParseBytes(ctx.String("blocksize"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse blocksize")
		return nil
	}
	if blocksize <= 0 {
		fatalIf(errInvalidArgument(), "blocksize cannot be <= 0")
		return nil
	}

	filesize, e := humanize.ParseBytes(ctx.String("filesize"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse filesize")
		return nil
	}
	if filesize <= 0 {
		fatalIf(errInvalidArgument(), "filesize cannot be <= 0")
		return nil
	}

	serial := ctx.Bool("serial")

	resultCh, e := client.DriveSpeedtest(ctxt, madmin.DriveSpeedTestOpts{
		Serial:    serial,
		BlockSize: blocksize,
		FileSize:  filesize,
	})

	if globalJSON {
		if e != nil {
			printMsg(convertPerfResult(PerfTestResult{
				Type:  DrivePerfTest,
				Err:   e.Error(),
				Final: true,
			}))

			return nil
		}

		var results []madmin.DriveSpeedTestResult
		for result := range resultCh {
			if result.Version != "" {
				results = append(results, result)
			}
		}
		printMsg(convertPerfResult(PerfTestResult{
			Type:        DrivePerfTest,
			DriveResult: results,
			Final:       true,
		}))

		return nil
	}

	done := make(chan struct{})

	p := tea.NewProgram(initSpeedTestUI())
	go func() {
		if _, e := p.Run(); e != nil {
			os.Exit(1)
		}
		close(done)
	}()

	go func() {
		if e != nil {
			r := PerfTestResult{
				Type:  DrivePerfTest,
				Err:   e.Error(),
				Final: true,
			}
			p.Send(r)
			if outCh != nil {
				outCh <- r
			}
			return
		}

		var results []madmin.DriveSpeedTestResult
		for result := range resultCh {
			if result.Version != "" {
				results = append(results, result)
			} else {
				p.Send(PerfTestResult{
					Type:        DrivePerfTest,
					DriveResult: []madmin.DriveSpeedTestResult{},
				})
			}
		}
		r := PerfTestResult{
			Type:        DrivePerfTest,
			DriveResult: results,
			Final:       true,
		}
		p.Send(r)
		if outCh != nil {
			outCh <- r
		}
	}()

	<-done

	return nil
}
