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
	"fmt"
	"io"
	"strings"
	"sync"
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

type caption struct {
	message   string
	separator rune
}

func (b *barSend) SetCaption(c caption) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdSetCaption, Arg: c}
}

func (b barSend) Finish() {
	defer close(b.cmdCh)
	b.cmdCh <- barMsg{Cmd: pbBarCmdFinish}
	<-b.finishCh
}

func trimBarCaption(c caption, width int) string {
	if len(c.message) > width {
		// Trim caption to fit within the screen
		trimSize := len(c.message) - width + 3 + 1
		if trimSize < len(c.message) {
			c.message = "..." + c.message[trimSize:]
			// Further trim partial names.
			partialTrimSize := strings.IndexByte(c.message, byte(c.separator))
			if partialTrimSize > 0 {
				c.message = c.message[partialTrimSize:]
			}
		}
	}
	return c.message
}

// newCpBar - instantiate a pbBar.
func newCpBar() barSend {
	cmdCh := make(chan barMsg)
	finishCh := make(chan bool)
	go func(cmdCh <-chan barMsg, finishCh chan<- bool) {
		var started bool
		var barCaption string
		var totalBytesRead int64 // total amounts of bytes read
		bar := pb.New64(0)
		bar.SetUnits(pb.U_BYTES)
		bar.SetRefreshRate(time.Millisecond * 10)
		bar.NotPrint = true
		bar.ShowSpeed = true
		cursorUp := fmt.Sprintf("%c[%dA", 27, 1)
		eraseCurrentLine := fmt.Sprintf("%c[2K\r", 27)
		cursorDown := fmt.Sprintf("%c[%dB", 27, 1)
		firstTime := true
		barLock := &sync.Mutex{}
		bar.Callback = func(s string) {
			barLock.Lock()
			{
				if !firstTime {
					console.Print(cursorUp)
					console.Print(eraseCurrentLine)
				}
				console.Bar(barCaption)
				console.Print(cursorDown)
				console.Print(eraseCurrentLine)
				console.Bar(s)

				firstTime = false
			}
			barLock.Unlock()
		}
		// Feels like wget
		bar.Format("[=> ]")
		for msg := range cmdCh {
			switch msg.Cmd {
			case pbBarCmdSetCaption:
				barCaption = trimBarCaption(msg.Arg.(caption), bar.GetWidth())
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
