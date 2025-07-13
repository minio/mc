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
	"errors"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"time"
)

type profiler interface {
	Start() error
	Stop() error
}

type cpuProfiler struct {
	*os.File
}

func newCPUProfiler(f *os.File) *cpuProfiler {
	return &cpuProfiler{File: f}
}

func (p *cpuProfiler) Start() error {
	return pprof.StartCPUProfile(p.File)
}

func (p *cpuProfiler) Stop() error {
	pprof.StopCPUProfile()
	return p.Close()
}

type memProfiler struct {
	*os.File
}

func newMemProfiler(f *os.File) *memProfiler {
	return &memProfiler{File: f}
}

func (p *memProfiler) Start() error {
	return nil
}

func (p *memProfiler) Stop() error {
	runtime.GC()
	if e := pprof.Lookup("heap").WriteTo(p.File, 0); e != nil {
		return e
	}
	return p.Close()
}

type blockProfiler struct {
	*os.File
}

func newBlockProfiler(f *os.File) *blockProfiler {
	return &blockProfiler{File: f}
}

func (p *blockProfiler) Start() error {
	runtime.SetBlockProfileRate(100)
	return nil
}

func (p *blockProfiler) Stop() error {
	if e := pprof.Lookup("block").WriteTo(p.File, 0); e != nil {
		return e
	}
	runtime.SetBlockProfileRate(0)
	return p.Close()
}

type goroutineProfiler struct {
	*os.File
}

func newGoroutineProfiler(f *os.File) *goroutineProfiler {
	return &goroutineProfiler{File: f}
}

func (p *goroutineProfiler) Start() error {
	return nil
}

func (p *goroutineProfiler) Stop() error {
	if e := pprof.Lookup("goroutine").WriteTo(p.File, 1); e != nil {
		return e
	}
	return p.Close()
}

var globalProfilers []profiler

// Enable profiling supported modes are [cpu, mem, block, goroutine].
func enableProfilers(outputFolder string, profilers []string) error {
	now := time.Now().Format("2006-01-02T15-04-05")

	if _, e := os.Stat(outputFolder); e != nil {
		if e := os.MkdirAll(outputFolder, 0o700); e != nil {
			return e
		}
	}

	for _, profilerName := range profilers {
		outputFile := path.Join(outputFolder, profilerName+"."+now)
		f, e := os.Create(outputFile)
		if e != nil {
			return e
		}

		var p profiler
		switch profilerName {
		case "cpu":
			p = newCPUProfiler(f)
		case "mem":
			p = newMemProfiler(f)
		case "block":
			p = newBlockProfiler(f)
		case "goroutine":
			p = newGoroutineProfiler(f)
		default:
			return errors.New("unknown profiler name")
		}

		if e := p.Start(); e != nil {
			return e
		}

		// Keep the profiler in a list to stop it later
		globalProfilers = append(globalProfilers, p)
	}

	return nil
}

func stopProfiling() error {
	for _, p := range globalProfilers {
		if e := p.Stop(); e != nil {
			return e
		}
	}
	return nil
}
