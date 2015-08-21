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
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/minio/mc/internal/github.com/dustin/go-humanize"
	"github.com/minio/mc/internal/github.com/minio/minio/pkg/probe"
	"github.com/minio/mc/internal/github.com/minio/pb"
	"github.com/minio/mc/internal/github.com/olekukonko/ts"

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
	io.ReadCloser
	bar *barSend
}

func (r *proxyReader) Read(p []byte) (n int, err error) {
	n, err = r.ReadCloser.Read(p)
	r.bar.Progress(int64(n))
	return
}

func (r *proxyReader) Close() (err error) {
	return r.ReadCloser.Close()
}

type barMsg struct {
	Cmd pbBarCmd
	Arg interface{}
}

type barSend struct {
	cmdCh    chan<- barMsg
	finishCh <-chan bool
}

func (b *barSend) NewProxyReader(r io.ReadCloser) *proxyReader {
	return &proxyReader{r, b}
}

func (b barSend) Extend(total int64) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdExtend, Arg: total}
}

func (b barSend) Progress(progress int64) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdProgress, Arg: progress}
}

func (b barSend) ErrorPut(size int64) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdPutError, Arg: size}
}

func (b barSend) ErrorGet(size int64) {
	b.cmdCh <- barMsg{Cmd: pbBarCmdGetError, Arg: size}
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

	switch runtime.GOOS {
	case "linux":
		// cursors = "➩➪➫➬➭➮➯➱"
		// cursors = "▁▃▄▅▆▇█▇▆▅▄▃"
		cursors = "◐◓◑◒"
		// cursors = "←↖↑↗→↘↓↙"
		// cursors = "◴◷◶◵"
		// cursors = "◰◳◲◱"
	case "darwin":
		cursors = "◐◓◑◒"
		//cursors = "⣾⣽⣻⢿⡿⣟⣯⣷"
	default:
		cursors = "|/-\\"
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

func fixateBarCaption(caption string, width int) string {
	switch {
	case len(caption) > width:
		// Trim caption to fit within the screen
		trimSize := len(caption) - width + 3
		if trimSize < len(caption) {
			caption = "..." + caption[trimSize:]
		}
	case len(caption) < width:
		caption += strings.Repeat(" ", width-len(caption))
	}
	return caption
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
			console.Print(console.Colorize("Bar", "\r"+s))
		}
		switch runtime.GOOS {
		case "linux":
			bar.Format("┃▓█░┃")
			// bar.Format("█▓▒░█")
		case "darwin":
			bar.Format(" ▓ ░ ")
		default:
			bar.Format("[=> ]")
		}
		for msg := range cmdCh {
			switch msg.Cmd {
			case pbBarCmdSetCaption:
				bar.Prefix(fixateBarCaption(msg.Arg.(string), getFixedWidth(bar.GetWidth(), 18)))
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

/******************************** Scan Bar ************************************/
// fixateScanBar truncates long text to fit within the terminal size.
func fixateScanBar(text string, width int) string {
	if len([]rune(text)) > width {
		// Trim text to fit within the screen
		trimSize := len([]rune(text)) - width + 3 //"..."
		if trimSize < len([]rune(text)) {
			text = "..." + text[trimSize:]
		}
	}
	return text
}

// Progress bar function report objects being scaned.
type scanBarFunc func(string)

// scanBarFactory returns a progress bar function to report URL scanning.
func scanBarFactory(prefix string) scanBarFunc {
	prevLineSize := 0
	fileCount := 0
	termSize, err := ts.GetSize()
	if err != nil {
		fatalIf(probe.NewError(err), "Unable to get terminal size.")
	}
	termWidth := termSize.Col()
	cursorCh := cursorAnimate()

	return func(source string) {
		scanPrefix := fmt.Sprintf("[%s] %s ", humanize.Comma(int64(fileCount)), string(<-cursorCh))
		if prefix != "" {
			scanPrefix = fmt.Sprintf("Scanning %s [%s] %s ", prefix, humanize.Comma(int64(fileCount)), string(<-cursorCh))
		}
		if prevLineSize != 0 { // erase previous line
			console.PrintC("\r" + scanPrefix + strings.Repeat(" ", prevLineSize-len([]rune(scanPrefix))))
		}
		source = fixateScanBar(source, termWidth-len([]rune(scanPrefix))-1)
		barText := "\r" + scanPrefix + source
		console.PrintC(barText)
		prevLineSize = len([]rune(barText))
		fileCount++
	}
}
