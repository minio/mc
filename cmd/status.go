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
	"io"
	"sync/atomic"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/pkg/console"
)

// Status implements a interface that can be used in quit mode or with progressbar.
type Status interface {
	Println(data ...interface{})
	AddCounts(int64)
	SetCounts(int64)
	GetCounts() int64

	Add(int64) Status
	Get() int64
	Start()
	Finish()

	PrintMsg(msg message)

	Update()
	Total() int64
	SetTotal(int64) Status
	SetCaption(string)

	Read(p []byte) (n int, err error)

	errorIf(err *probe.Error, msg string)
	fatalIf(err *probe.Error, msg string)
}

// NewQuietStatus returns a quiet status object
func NewQuietStatus(hook io.Reader) Status {
	return &QuietStatus{
		accounter: newAccounter(0),
		hook:      hook,
	}
}

// QuietStatus will only show the progress and summary
type QuietStatus struct {
	// Keep this as first element of struct because it guarantees 64bit
	// alignment on 32 bit machines. atomic.* functions crash if operand is not
	// aligned at 64bit. See https://github.com/golang/go/issues/599
	counts int64
	*accounter
	hook io.Reader
}

// Read implements the io.Reader interface
func (qs *QuietStatus) Read(p []byte) (n int, err error) {
	qs.hook.Read(p)
	return qs.accounter.Read(p)
}

// SetCounts sets number of files uploaded
func (qs *QuietStatus) SetCounts(v int64) {
	atomic.StoreInt64(&qs.counts, v)
}

// GetCounts returns number of files uploaded
func (qs *QuietStatus) GetCounts() int64 {
	return atomic.LoadInt64(&qs.counts)
}

// AddCounts adds 'v' number of files uploaded.
func (qs *QuietStatus) AddCounts(v int64) {
	atomic.AddInt64(&qs.counts, v)
}

// SetTotal sets the total of the progressbar, ignored for quietstatus
func (qs *QuietStatus) SetTotal(v int64) Status {
	qs.accounter.Set(v)
	return qs
}

// SetCaption sets the caption of the progressbar, ignored for quietstatus
func (qs *QuietStatus) SetCaption(s string) {
}

// Get returns the current number of bytes
func (qs *QuietStatus) Get() int64 {
	return qs.accounter.Get()
}

// Total returns the total number of bytes
func (qs *QuietStatus) Total() int64 {
	return qs.accounter.Get()
}

// Add bytes to current number of bytes
func (qs *QuietStatus) Add(v int64) Status {
	qs.accounter.Add(v)
	return qs
}

// Println prints line, ignored for quietstatus
func (qs *QuietStatus) Println(data ...interface{}) {
}

// PrintMsg prints message
func (qs *QuietStatus) PrintMsg(msg message) {
	printMsg(msg)
}

// Start is ignored for quietstatus
func (qs *QuietStatus) Start() {
}

// Finish displays the accounting summary
func (qs *QuietStatus) Finish() {
	printMsg(qs.accounter.Stat())
}

// Update is ignored for quietstatus
func (qs *QuietStatus) Update() {
}

func (qs *QuietStatus) errorIf(err *probe.Error, msg string) {
	errorIf(err, msg)
}

func (qs *QuietStatus) fatalIf(err *probe.Error, msg string) {
	fatalIf(err, msg)
}

// NewProgressStatus returns a progress status object
func NewProgressStatus(hook io.Reader) Status {
	return &ProgressStatus{
		progressBar: newProgressBar(0),
		hook:        hook,
	}
}

// ProgressStatus shows a progressbar
type ProgressStatus struct {
	// Keep this as first element of struct because it guarantees 64bit
	// alignment on 32 bit machines. atomic.* functions crash if operand is not
	// aligned at 64bit. See https://github.com/golang/go/issues/599
	counts int64
	*progressBar
	hook io.Reader
}

// Read implements the io.Reader interface
func (ps *ProgressStatus) Read(p []byte) (n int, err error) {
	ps.hook.Read(p)
	return ps.progressBar.Read(p)
}

// SetCaption sets the caption of the progressbar
func (ps *ProgressStatus) SetCaption(s string) {
	ps.progressBar.SetCaption(s)
}

// SetCounts sets number of files uploaded
func (ps *ProgressStatus) SetCounts(v int64) {
	atomic.StoreInt64(&ps.counts, v)
}

// GetCounts returns number of files uploaded
func (ps *ProgressStatus) GetCounts() int64 {
	return atomic.LoadInt64(&ps.counts)
}

// AddCounts adds 'v' number of files uploaded.
func (ps *ProgressStatus) AddCounts(v int64) {
	atomic.AddInt64(&ps.counts, v)
}

// Get returns the current number of bytes
func (ps *ProgressStatus) Get() int64 {
	return ps.progressBar.Get()
}

// Total returns the total number of bytes
func (ps *ProgressStatus) Total() int64 {
	return ps.progressBar.Get()
}

// SetTotal sets the total of the progressbar
func (ps *ProgressStatus) SetTotal(v int64) Status {
	ps.progressBar.SetTotal(v)
	return ps
}

// Add bytes to current number of bytes
func (ps *ProgressStatus) Add(v int64) Status {
	ps.progressBar.Add64(v)
	return ps
}

// Println prints line, ignored for quietstatus
func (ps *ProgressStatus) Println(data ...interface{}) {
	console.Eraseline()
	console.Println(data...)
}

// PrintMsg prints message
func (ps *ProgressStatus) PrintMsg(msg message) {
}

// Start is ignored for quietstatus
func (ps *ProgressStatus) Start() {
	ps.progressBar.Start()
}

// Finish displays the accounting summary
func (ps *ProgressStatus) Finish() {
	ps.progressBar.Finish()
}

// Update is ignored for quietstatus
func (ps *ProgressStatus) Update() {
	ps.progressBar.Update()
}

func (ps *ProgressStatus) errorIf(err *probe.Error, msg string) {
	// remove progressbar
	console.Eraseline()
	errorIf(err, msg)

	ps.progressBar.Update()
}

func (ps *ProgressStatus) fatalIf(err *probe.Error, msg string) {
	// remove progressbar
	console.Eraseline()
	fatalIf(err, msg)

	ps.progressBar.Update()
}
