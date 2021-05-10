// Copyright (c) 2015-2021 MinIO, Inc.
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
	"context"
	"sync"
	"time"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio-go/v7/pkg/notification"
)

// EventInfo contains the information of the event that occurred and the source
// IP:PORT of the client which triggerred the event.
type EventInfo struct {
	Time         string
	Size         int64
	UserMetadata map[string]string
	Path         string
	Host         string
	Port         string
	UserAgent    string
	Type         notification.EventType
}

// WatchOptions contains watch configuration options
type WatchOptions struct {
	Prefix    string
	Suffix    string
	Events    []string
	Recursive bool
}

// WatchObject captures watch channels to read and listen on.
type WatchObject struct {
	// eventInfo will be put on this chan
	EventInfoChan chan []EventInfo
	// errors will be put on this chan
	ErrorChan chan *probe.Error
	// will stop the watcher goroutines
	DoneChan chan struct{}
}

// Events returns the chan receiving events
func (w *WatchObject) Events() chan []EventInfo {
	return w.EventInfoChan
}

// Errors returns the chan receiving errors
func (w *WatchObject) Errors() chan *probe.Error {
	return w.ErrorChan
}

// Watcher can be used to have one or multiple clients watch for notifications
type Watcher struct {
	sessionStartTime time.Time

	// all error will be added to this chan
	ErrorChan chan *probe.Error
	// all events will be added to this chan
	EventInfoChan chan []EventInfo

	// array of watchers joined
	o []*WatchObject

	// all watchers joining will enter this waitgroup
	wg sync.WaitGroup
}

// NewWatcher creates a new watcher
func NewWatcher(sessionStartTime time.Time) *Watcher {
	return &Watcher{
		sessionStartTime: sessionStartTime,
		ErrorChan:        make(chan *probe.Error),
		EventInfoChan:    make(chan []EventInfo),
		o:                []*WatchObject{},
	}
}

// Errors returns a channel which will receive errors
func (w *Watcher) Errors() chan *probe.Error {
	return w.ErrorChan
}

// Events returns a channel which will receive events
func (w *Watcher) Events() chan []EventInfo {
	return w.EventInfoChan
}

// Watching returns if the watcher is watching for notifications
func (w *Watcher) Watching() bool {
	return (len(w.o) > 0)
}

// Wait for watcher to wait
func (w *Watcher) Wait() {
	w.wg.Wait()
}

// Join the watcher with client
func (w *Watcher) Join(ctx context.Context, client Client, recursive bool) *probe.Error {
	wo, err := client.Watch(ctx, WatchOptions{
		Recursive: recursive,
		Events:    []string{"put", "delete", "bucket-creation", "bucket-removal"},
	})
	if err != nil {
		return err
	}

	w.o = append(w.o, wo)

	// join monitoring waitgroup
	w.wg.Add(1)

	// wait for events and errors of individual client watchers
	// and sent then to eventsChan and errorsChan
	go func() {
		defer w.wg.Done()

		for {
			select {
			case events, ok := <-wo.Events():
				if !ok {
					return
				}
				w.EventInfoChan <- events
			case err, ok := <-wo.Errors():
				if !ok {
					return
				}

				w.ErrorChan <- err
			}
		}
	}()

	return nil
}
