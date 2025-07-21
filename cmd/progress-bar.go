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
	"io"
	"runtime"
	"strings"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/fatih/color"
	"github.com/minio/pkg/v3/console"
)

// progress extender.
type progressBar struct {
	*pb.ProgressBar
}

func newPB(total int64) *pb.ProgressBar {
	// Progress bar specific theme customization.
	console.SetColor("Bar", color.New(color.FgGreen, color.Bold))

	// get the new original progress bar.
	bar := pb.New64(total)

	// Set new human friendly print units.
	bar.SetUnits(pb.U_BYTES)

	// Refresh rate for progress bar is set to 125 milliseconds.
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
		// Need to add '\x00' as delimiter for unicode characters.
		bar.Format("┃\x00▓\x00█\x00░\x00┃")
	case "darwin":
		// Need to add '\x00' as delimiter for unicode characters.
		bar.Format(" \x00▓\x00 \x00░\x00 ")
	default:
		// Default to non unicode characters.
		bar.Format("[=> ]")
	}

	// Start the progress bar.
	return bar.Start()
}

func newProgressReader(r io.Reader, caption string, total int64) *pb.Reader {
	bar := newPB(total)

	if caption != "" {
		bar.Prefix(caption)
	}

	return bar.NewProxyReader(r)
}

// newProgressBar - instantiate a progress bar.
func newProgressBar(total int64) *progressBar {
	bar := newPB(total)

	// Return new progress bar here.
	return &progressBar{ProgressBar: bar}
}

// Set caption.
func (p *progressBar) SetCaption(caption string) *progressBar {
	caption = fixateBarCaption(caption, getFixedWidth(p.GetWidth(), 18))
	p.Prefix(caption)
	return p
}

func (p *progressBar) Finish() {
	p.ProgressBar.Finish()
}

func (p *progressBar) Set64(length int64) *progressBar {
	p.ProgressBar = p.ProgressBar.Set64(length)
	return p
}

func (p *progressBar) Read(buf []byte) (n int, err error) {
	defer func() {
		// Upload retry can read one object twice; Avoid read to be greater than Total
		if n, t := p.Get(), p.Total; t > 0 && n > t {
			p.ProgressBar.Set64(t)
		}
	}()

	return p.ProgressBar.Read(buf)
}

func (p *progressBar) SetTotal(total int64) {
	p.Total = total
}

// cursorAnimate - returns a animated rune through read channel for every read.
func cursorAnimate() <-chan string {
	cursorCh := make(chan string)
	var cursors []string

	switch runtime.GOOS {
	case "linux":
		// cursors = "➩➪➫➬➭➮➯➱"
		// cursors = "▁▃▄▅▆▇█▇▆▅▄▃"
		cursors = []string{"◐", "◓", "◑", "◒"}
		// cursors = "←↖↑↗→↘↓↙"
		// cursors = "◴◷◶◵"
		// cursors = "◰◳◲◱"
		// cursors = "⣾⣽⣻⢿⡿⣟⣯⣷"
	case "darwin":
		cursors = []string{"◐", "◓", "◑", "◒"}
	default:
		cursors = []string{"|", "/", "-", "\\"}
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
