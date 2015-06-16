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
	"runtime"
	"sync/atomic"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/minio/mc/pkg/console"
)

type pbBarCmd int

const (
	pbBarCmdExtend pbBarCmd = iota
	pbBarCmdProgress
	pbBarCmdFinish
	pbBarCmdPutError
	pbBarCmdGetError
	pbBarCmdSetCaption
)

type proxyReader struct {
	io.Reader
	bar *barSend
}

func (r *proxyReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.bar.progress(int64(n))
	return
}

type barMsg struct {
	Cmd pbBarCmd
	Arg interface{}
}

type barSend struct {
	cmdCh    chan<- barMsg
	finishCh <-chan bool
}

func (b barSend) Extend(total int64) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdExtend, Arg: total}
}

func (b barSend) progress(progress int64) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdProgress, Arg: progress}
}

func (b barSend) ErrorPut(size int64) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdPutError, Arg: size}
}

func (b barSend) ErrorGet(size int64) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdGetError, Arg: size}
}

func (b *barSend) NewProxyReader(r io.Reader) *proxyReader {
	return &proxyReader{r, b}
}

func (b *barSend) SetCaption(c string) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdSetCaption, Arg: c}
}

func (b barSend) Finish() {
	defer close(b.cmdCh)
	b.cmdCh <- barMsg{Cmd: pbBarCmdFinish}
	<-b.finishCh
	console.Println()
}

func cursorAnimate() <-chan rune {
	cursorCh := make(chan rune)
	var cursors string
	if runtime.GOOS == "windows" {
		cursors = "|/-\\"
	} else {
		cursors = "➩➪➫➬➭➮➯➱"
	}
	go func() {
		for {
			for _, cursor := range cursors {
				cursorCh <- cursor
			}
		}
	}()
	return cursorCh
}

func fixateBarCaption(c string, s string, width int) string {
	if len(c) > width {
		// Trim caption to fit within the screen
		trimSize := len(c) - width + 3 + 1
		if trimSize < len(c) {
			c = "..." + c[trimSize:]
		}
	}
	return s + " " + c
}

func getFixedWidth(width, percent int) int {
	return width * percent / 100
}

// newCpBar - instantiate a pbBar.
func newCpBar() barSend {
	cmdCh := make(chan barMsg)
	finishCh := make(chan bool)
	go func(cmdCh <-chan barMsg, finishCh chan<- bool) {
		var started bool
		var totalBytesRead int64 // total amounts of bytes read
		bar := pb.New64(0)
		bar.SetUnits(pb.U_BYTES)
		bar.SetRefreshRate(time.Millisecond * 125)
		bar.NotPrint = true
		bar.ShowSpeed = true
		bar.Callback = func(s string) {
			console.Bar(s + "\r")
		}
		cursorCh := cursorAnimate()
		// Feels like wget
		bar.Format("[=> ]")
		for msg := range cmdCh {
			switch msg.Cmd {
			case pbBarCmdSetCaption:
				bar.Prefix(fixateBarCaption(msg.Arg.(string), string(<-cursorCh), getFixedWidth(bar.GetWidth(), 10)))
			case pbBarCmdExtend:
				atomic.AddInt64(&bar.Total, msg.Arg.(int64))
			case pbBarCmdProgress:
				if bar.Total > 0 && !started {
					started = true
					bar.Start()
				}
				if msg.Arg.(int64) > 0 {
					totalBytesRead += msg.Arg.(int64)
					bar.Add64(msg.Arg.(int64))
				}
			case pbBarCmdPutError:
				if totalBytesRead > msg.Arg.(int64) {
					bar.Set64(totalBytesRead - msg.Arg.(int64))
				}
			case pbBarCmdGetError:
				if msg.Arg.(int64) > 0 {
					bar.Add64(msg.Arg.(int64))
				}
			case pbBarCmdFinish:
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
