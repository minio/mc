package main

import (
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
)

// Implements a simple status interface that can be used in quit mode or with progressbar.

type Status interface {
	Println(data ...interface{})
	Add(int64) Status
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

func NewQuitStatus() Status {
	return &QuitStatus{
		newAccounter(0),
	}
}

type QuitStatus struct {
	*accounter
}

func (s *QuitStatus) Read(p []byte) (n int, err error) {
	return s.accounter.Read(p)
}

func (s *ProgressStatus) Read(p []byte) (n int, err error) {
	return s.progressBar.Read(p)
}

func (s *QuitStatus) SetTotal(v int64) Status {
	s.accounter.Total = v
	return s
}

func (qs *QuitStatus) SetCaption(s string) {
}

func (s *QuitStatus) Total() int64 {
	return s.accounter.Total
}

func (ps *ProgressStatus) SetCaption(s string) {
	ps.progressBar.SetCaption(s)
}

func (s *ProgressStatus) Total() int64 {
	return s.progressBar.Total
}

func (s *ProgressStatus) SetTotal(v int64) Status {
	s.progressBar.Total = v
	return s
}

func (s *QuitStatus) Add(v int64) Status {
	s.accounter.Add(v)
	return s
}

func (s *ProgressStatus) Add(v int64) Status {
	s.progressBar.Add64(v)
	return s
}

func (s *QuitStatus) Println(data ...interface{}) {
}

func (s *QuitStatus) PrintMsg(msg message) {
	if !globalJSON {
		console.Println(msg.String())
	} else {
		console.Println(msg.JSON())
	}
}

func (s *QuitStatus) Start() {
}

func (s *QuitStatus) Finish() {
	accntStat := s.accounter.Stat()
	cpStatMessage := mirrorStatMessage{
		Total:       accntStat.Total,
		Transferred: accntStat.Transferred,
		Speed:       accntStat.Speed,
	}

	console.Println(console.Colorize("Mirror", cpStatMessage.String()))
}

func (s *QuitStatus) Update() {
}

func (s *QuitStatus) errorIf(err *probe.Error, msg string) {
	errorIf(err, msg)
}

func (s *QuitStatus) fatalIf(err *probe.Error, msg string) {
	fatalIf(err, msg)
}

func NewProgressStatus() Status {
	return &ProgressStatus{
		newProgressBar(0),
	}
}

type ProgressStatus struct {
	*progressBar
}

func (s *ProgressStatus) Println(data ...interface{}) {
	console.Eraseline()
	console.Println(data...)
}

func (s *ProgressStatus) PrintMsg(msg message) {
}

func (s *ProgressStatus) Start() {
	s.progressBar.Start()
}

func (s *ProgressStatus) Finish() {
	s.progressBar.Finish()
}

func (s *ProgressStatus) Update() {
	s.progressBar.Update()
}

func (s *ProgressStatus) errorIf(err *probe.Error, msg string) {
	// remove progressbar
	console.Eraseline()
	errorIf(err, msg)

	s.progressBar.Update()
}

func (s *ProgressStatus) fatalIf(err *probe.Error, msg string) {
	// remove progressbar
	console.Eraseline()
	fatalIf(err, msg)

	s.progressBar.Update()
}
