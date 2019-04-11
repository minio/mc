/*
 * MinIO Client, (C) 2016 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"io"

	"github.com/minio/mc/pkg/console"
	"github.com/minio/mc/pkg/probe"
)

// Status implements a interface that can be used in quit mode or with progressbar.
type Status interface {
	Println(data ...interface{})
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

// NewDummyStatus returns a dummy status object
func NewDummyStatus(hook io.Reader) Status {
	return &DummyStatus{hook}
}

// DummyStatus will not show anything.
type DummyStatus struct {
	hook io.Reader
}

// Read implements the io.Reader interface
func (ds *DummyStatus) Read(p []byte) (n int, err error) {
	ds.hook.Read(p)
	return len(p), nil
}

// Get implements Progress interface
func (ds *DummyStatus) Get() int64 {
	return 0
}

// SetTotal sets the total of the progressbar, ignored for quietstatus
func (ds *DummyStatus) SetTotal(v int64) Status {
	return ds
}

// SetCaption sets the caption of the progressbar, ignored for quietstatus
func (ds *DummyStatus) SetCaption(s string) {}

// Total returns the total number of bytes
func (ds *DummyStatus) Total() int64 {
	return 0
}

// Add bytes to current number of bytes
func (ds *DummyStatus) Add(v int64) Status {
	return ds
}

// Println prints line, ignored for quietstatus
func (ds *DummyStatus) Println(data ...interface{}) {}

// PrintMsg prints message
func (ds *DummyStatus) PrintMsg(msg message) {
	if !globalJSON {
		console.Println(msg.String())
	} else {
		console.Println(msg.JSON())
	}
}

// Start is ignored for quietstatus
func (ds *DummyStatus) Start() {}

// Finish displays the accounting summary
func (ds *DummyStatus) Finish() {}

// Update is ignored for quietstatus
func (ds *DummyStatus) Update() {}

func (ds *DummyStatus) errorIf(err *probe.Error, msg string) {
	errorIf(err, msg)
}

func (ds *DummyStatus) fatalIf(err *probe.Error, msg string) {
	fatalIf(err, msg)
}

// NewQuietStatus returns a quiet status object
func NewQuietStatus(hook io.Reader) Status {
	return &QuietStatus{
		newAccounter(0),
		hook,
	}
}

// QuietStatus will only show the progress and summary
type QuietStatus struct {
	*accounter
	hook io.Reader
}

// Read implements the io.Reader interface
func (qs *QuietStatus) Read(p []byte) (n int, err error) {
	qs.hook.Read(p)
	return qs.accounter.Read(p)
}

// SetTotal sets the total of the progressbar, ignored for quietstatus
func (qs *QuietStatus) SetTotal(v int64) Status {
	qs.accounter.Total = v
	return qs
}

// SetCaption sets the caption of the progressbar, ignored for quietstatus
func (qs *QuietStatus) SetCaption(s string) {
}

// Total returns the total number of bytes
func (qs *QuietStatus) Total() int64 {
	return qs.accounter.Total
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
	if !globalJSON {
		console.Println(msg.String())
	} else {
		console.Println(msg.JSON())
	}
}

// Start is ignored for quietstatus
func (qs *QuietStatus) Start() {
}

// Finish displays the accounting summary
func (qs *QuietStatus) Finish() {
	console.Println(console.Colorize("Mirror", qs.accounter.Stat().String()))
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
		newProgressBar(0),
		hook,
	}
}

// ProgressStatus shows a progressbar
type ProgressStatus struct {
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

// Total returns the total number of bytes
func (ps *ProgressStatus) Total() int64 {
	return ps.progressBar.Total
}

// SetTotal sets the total of the progressbar
func (ps *ProgressStatus) SetTotal(v int64) Status {
	ps.progressBar.Total = v
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
