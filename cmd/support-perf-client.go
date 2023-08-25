// Copyright (c) 2023 MinIO, Inc.
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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/minio/cli"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/mc/pkg/probe"
)

func mainAdminSpeedTestClientPerf(ctx *cli.Context, aliasedURL string, outCh chan<- PerfTestResult) error {
	client, perr := newAdminClient(aliasedURL)
	if perr != nil {
		fatalIf(perr.Trace(aliasedURL), "Unable to initialize admin client.")
		return nil
	}

	ctxt, cancel := context.WithCancel(globalContext)
	defer cancel()

	duration, e := time.ParseDuration(ctx.String("duration"))
	if e != nil {
		fatalIf(probe.NewError(e), "Unable to parse duration")
		return nil
	}
	if duration <= 0 {
		fatalIf(errInvalidArgument(), "duration cannot be 0 or negative")
		return nil
	}

	resultCh := make(chan madmin.ClientPerfResult)
	errorCh := make(chan error)
	go func() {
		defer close(resultCh)
		defer close(errorCh)
		result, e := client.ClientPerf(ctxt, duration)
		if e != nil {
			errorCh <- e
		}
		resultCh <- result
	}()
	if globalJSON {
		var rsl PerfTestOutput
		select {
		case e := <-errorCh:
			rsl = convertPerfResult(PerfTestResult{
				Type:  ClientPerfTest,
				Err:   e.Error(),
				Final: true,
			})
		case result := <-resultCh:
			rsl = convertPerfResult(PerfTestResult{
				Type:         ClientPerfTest,
				ClientResult: &result,
				Final:        true,
			})
		}
		printMsg(rsl)
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
		for {
			select {
			case e := <-errorCh:
				r := PerfTestResult{
					Type:  ClientPerfTest,
					Err:   e.Error(),
					Final: true,
				}
				p.Send(r)
				if outCh != nil {
					outCh <- r
				}
				return
			case result := <-resultCh:
				r := PerfTestResult{
					Type:         ClientPerfTest,
					ClientResult: &result,
					Final:        true,
				}
				p.Send(r)
				if outCh != nil {
					outCh <- r
				}
				return
			default:
				p.Send(PerfTestResult{
					Type:         ClientPerfTest,
					ClientResult: &madmin.ClientPerfResult{},
				})
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	<-done

	return nil
}
