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
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
)

type accountingReader struct {
	io.ReadCloser
	acct *accounter
}

type accounter struct {
	current int64

	Total        int64
	startTime    time.Time
	startValue   int64
	refreshRate  time.Duration
	currentValue int64
	finishOnce   sync.Once
	isFinished   chan struct{}
}

// Instantiate a new accounter
func newAccounter(total int64) *accounter {
	acct := &accounter{
		Total:        total,
		startTime:    time.Now(),
		startValue:   0,
		refreshRate:  time.Millisecond * 200,
		isFinished:   make(chan struct{}),
		currentValue: -1,
	}
	go acct.writer()
	return acct
}

func (a *accounter) write(current int64) string {
	var speedBox string
	fromStart := time.Now().Sub(a.startTime)
	currentFromStart := current - a.startValue

	if currentFromStart > 0 {
		speed := float64(currentFromStart) / (float64(fromStart) / float64(time.Second))
		speedBox = humanize.IBytes(uint64(speed))
	}
	return speedBox + "/s"
}

func (a *accounter) writer() {
	a.Update()
	for {
		select {
		case <-a.isFinished:
			return
		case <-time.After(a.refreshRate):
			a.Update()
		}
	}
}

func (a *accounter) Finish() string {
	var message string
	a.finishOnce.Do(func() {
		close(a.isFinished)
		message = fmt.Sprintf("Total: %s, Transferred: %s, Speed: %s", humanize.IBytes(uint64(a.Total)),
			humanize.IBytes(uint64(a.current)), a.write(atomic.LoadInt64(&a.current)))
	})
	return message
}

func (a *accounter) Update() {
	c := atomic.LoadInt64(&a.current)
	if c != a.currentValue {
		a.write(c)
		a.currentValue = c
	}
}

func (a *accounter) Add(n int64) int64 {
	return atomic.AddInt64(&a.current, n)
}

func (a *accounter) NewProxyReader(r io.ReadCloser) *accountingReader {
	return &accountingReader{r, a}
}

func (a *accountingReader) Read(p []byte) (n int, err error) {
	n, err = a.ReadCloser.Read(p)
	if err != nil {
		return
	}
	a.acct.Add(int64(n))
	return
}
