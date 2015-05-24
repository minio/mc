/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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

package main

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/minio/mc/pkg/console"
)

type cpBarCmd int

const (
	cpBarCmdExtend cpBarCmd = iota
	cpBarCmdProgress
	cpBarCmdFinish
)

type copyReader struct {
	io.Reader
	bar *barSend
}

func (r *copyReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.bar.Progress(int64(n))
	return
}

type barMsg struct {
	Cmd cpBarCmd
	Arg interface{}
}

type barSend struct {
	cmdCh    chan<- barMsg
	finishCh <-chan bool
}

func (b barSend) Extend(total int64) {
	b.cmdCh <- barMsg{Cmd: cpBarCmdExtend, Arg: total}
}

func (b barSend) Progress(progress int64) {
	b.cmdCh <- barMsg{Cmd: cpBarCmdProgress, Arg: progress}
}

func (b *barSend) NewProxyReader(r io.Reader) *copyReader {
	return &copyReader{r, b}
}

func (b barSend) Finish() {
	defer close(b.cmdCh)
	b.cmdCh <- barMsg{Cmd: cpBarCmdFinish}
	<-b.finishCh
}

// newCpBar - instantiate a cpBar.
func newCpBar() barSend {
	cmdCh := make(chan barMsg)
	finishCh := make(chan bool)
	go func(cmdCh <-chan barMsg, finishCh chan<- bool) {
		started := false
		bar := pb.New64(0)
		bar.SetUnits(pb.U_BYTES)
		bar.SetRefreshRate(time.Millisecond * 10)
		bar.NotPrint = true
		bar.ShowSpeed = true
		bar.Callback = func(s string) {
			// Colorize
			console.Bar("\r" + s)
		}
		// Feels like wget
		bar.Format("[=> ]")
		for msg := range cmdCh {
			switch msg.Cmd {
			case cpBarCmdExtend:
				atomic.AddInt64(&bar.Total, msg.Arg.(int64))
				if bar.Total > 0 && !started {
					started = true
					bar.Start()
				}
			case cpBarCmdProgress:
				bar.Add64(msg.Arg.(int64))
			case cpBarCmdFinish:
				if started {
					bar.Finish()
				}
				finishCh <- true
				return
			}
		}
	}(cmdCh, finishCh)

	return barSend{cmdCh, finishCh}
}
