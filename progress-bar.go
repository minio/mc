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
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/minio/pb"

	"github.com/minio/mc/pkg/console"
)

// pbBar type of operation.
type pbBar int

// collection of different progress bar operations.
const (
	pbBarProgress pbBar = iota
	pbBarFinish
	pbBarPutError
	pbBarGetError
	pbBarSetCaption
)

// proxyReader progress bar proxy reader for barSend inherits io.ReadCloser.
type proxyReader struct {
	io.ReadSeeker
	bar *barSend
}

// Read proxy Read sends progress for each Read operation.
func (r *proxyReader) Read(p []byte) (n int, err error) {
	n, err = r.ReadSeeker.Read(p)
	if err != nil {
		return
	}
	r.bar.Progress(int64(n))
	return
}

func (r *proxyReader) Seek(offset int64, whence int) (n int64, err error) {
	n, err = r.ReadSeeker.Seek(offset, whence)
	if err != nil {
		return
	}
	r.bar.Progress(n)
	return
}

// barMsg progress bar message for a given operation.
type barMsg struct {
	Op  pbBar
	Arg interface{}
}

// barSend implements various methods for progress bar operation.
type barSend struct {
	opCh     chan<- barMsg
	finishCh <-chan bool
}

// Instantiate a new progress bar proxy reader.
func (b *barSend) NewProxyReader(r io.ReadSeeker) *proxyReader {
	return &proxyReader{r, b}
}

// Progress send current progress message.
func (b barSend) Progress(progress int64) {
	b.opCh <- barMsg{Op: pbBarProgress, Arg: progress}
}

// ErrorPut send message for error in put operation.
func (b barSend) ErrorPut(size int64) {
	b.opCh <- barMsg{Op: pbBarPutError, Arg: size}
}

// ErrorGet send message for error in get operation.
func (b barSend) ErrorGet(size int64) {
	b.opCh <- barMsg{Op: pbBarGetError, Arg: size}
}

// SetCaption set an additional prefix/caption for an active progress bar.
func (b *barSend) SetCaption(c string) {
	b.opCh <- barMsg{Op: pbBarSetCaption, Arg: c}
}

// Finish finishes the progress bar and closes the message channel.
func (b barSend) Finish() {
	defer close(b.opCh)
	b.opCh <- barMsg{Op: pbBarFinish}
	<-b.finishCh
}

// cursorAnimate - returns a animated rune through read channel for every read.
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

// fixateBarCaption - fancify bar caption based on the terminal width.
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

// getFixedWidth - get a fixed width based for a given percentage.
func getFixedWidth(width, percent int) int {
	return width * percent / 100
}

// newProgressBar - instantiate a progress bar.
func newProgressBar(total int64) *barSend {
	// Progress bar speific theme customization.
	console.SetColor("Bar", color.New(color.FgGreen, color.Bold))

	cmdCh := make(chan barMsg)
	finishCh := make(chan bool)
	go func(total int64, cmdCh <-chan barMsg, finishCh chan<- bool) {
		var started bool         // has the progress bar started? default is false.
		var totalBytesRead int64 // total amounts of bytes read

		// get the new original progress bar.
		bar := pb.New64(total)
		bar.SetUnits(pb.U_BYTES)

		// refresh rate for progress bar is set to 125 milliseconds.
		bar.SetRefreshRate(time.Millisecond * 125)

		// Do not print a newline by default handled, it is handled manually.
		bar.NotPrint = true

		// Show current speed is true.
		bar.ShowSpeed = true

		// Custom callback with colorized bar.
		bar.Callback = func(s string) {
			console.Print(console.Colorize("Bar", "\r"+s))
		}

		// Use different unicodes for Linux, OS X and Windows.
		switch runtime.GOOS {
		case "linux":
			bar.Format("┃▓█░┃")
			// bar.Format("█▓▒░█")
		case "darwin":
			bar.Format(" ▓ ░ ")
		default:
			bar.Format("[=> ]")
		}

		// Look for incoming progress bar messages.
		for msg := range cmdCh {
			switch msg.Op {
			case pbBarSetCaption:
				// Sets a new caption prefixed along with progress bar.
				bar.Prefix(fixateBarCaption(msg.Arg.(string), getFixedWidth(bar.GetWidth(), 18)))
			case pbBarProgress:
				// Initializes the progerss bar, if already started bumps up the totalBytes.
				if bar.Total > 0 && !started {
					started = true
					bar.Start()
				}
				if msg.Arg.(int64) > 0 {
					totalBytesRead += msg.Arg.(int64)
					bar.Add64(msg.Arg.(int64))
				}
			case pbBarPutError:
				// Negates any put error of size from totalBytes.
				if totalBytesRead > msg.Arg.(int64) {
					bar.Set64(totalBytesRead - msg.Arg.(int64))
				}
			case pbBarGetError:
				// Retains any size transferred but failed.
				if msg.Arg.(int64) > 0 {
					bar.Add64(msg.Arg.(int64))
				}
			case pbBarFinish:
				// Progress finishes here.
				if started {
					bar.Finish()
				}
				// All done send true.
				finishCh <- true
				return
			}
		}
	}(total, cmdCh, finishCh)
	return &barSend{cmdCh, finishCh}
}
