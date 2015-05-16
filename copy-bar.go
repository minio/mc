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
	"time"

	"github.com/cheggaaa/pb"
	"github.com/minio/mc/pkg/console"
)

type copyBarCmd int

const (
	copyBarCmdExtend copyBarCmd = iota
	copyBarCmdProgress
	copyBarCmdFinish
)

type barMsg struct {
	Cmd copyBarCmd
	io.Writer
	Arg interface{}
}

type barSend struct {
	cmdCh    chan<- barMsg
	finishCh <-chan bool
}

func (b barSend) Extend(total int64) {
	b.cmdCh <- barMsg{Cmd: copyBarCmdExtend, Arg: total}
}

func (b barSend) Progress(progress int64) {
	b.cmdCh <- barMsg{Cmd: copyBarCmdProgress, Arg: progress}
}

func (b *barSend) Write(p []byte) (n int, err error) {
	length := len(p)
	b.Progress(int64(length))
	return length, nil
}

func (b barSend) Finish() {
	defer close(b.cmdCh)
	b.cmdCh <- barMsg{Cmd: copyBarCmdFinish}
	<-b.finishCh
}

// newCopyBar - instantiate a copyBar. When 'Quiet' is set to true, it CopyBar only prints text message for each item.
func newCopyBar(quiet bool) barSend {
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
			console.Print("\r" + s)
		}

		// Feels like wget
		bar.Format("[=> ]")
		for msg := range cmdCh {
			switch msg.Cmd {
			case copyBarCmdExtend:
				bar.Total += msg.Arg.(int64)
				if bar.Total > 0 && !started {
					started = true
					bar.Start()
				}
			case copyBarCmdProgress:
				bar.Add64(msg.Arg.(int64))
			case copyBarCmdFinish:
				bar.Finish()
				finishCh <- true
				return
			}
		}
	}(cmdCh, finishCh)

	return barSend{cmdCh, finishCh}
}
