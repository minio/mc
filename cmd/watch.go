/*
 * MinIO Client (C) 2014, 2015 MinIO, Inc.
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

package cmd

import (
	"context"
	"sync"
	"time"

	"github.com/minio/mc/pkg/probe"
)

// EventType represents the type of the event occurred.
type EventType string

const (
	// EventCreate notifies when a new object is created
	EventCreate EventType = "ObjectCreated"

	// EventCreateCopy notifies when there was a server side copy
	EventCreateCopy EventType = "ObjectCreated:Copy"

	// EventRemove notifies when a new object is deleted
	EventRemove = "ObjectRemoved"

	// Following are MinIO server specific events

	// EventCreatePutRetention notifies when a retention configuration is added to an object
	EventCreatePutRetention EventType = "ObjectCreated:PutRetention"

	// EventCreatePutLegalHold notifies when a legal hold configuration is added to an object
	EventCreatePutLegalHold EventType = "ObjectCreated:PutLegalHold"

	// EventAccessed notifies when an object is accessed.
	EventAccessed = "ObjectAccessed"
	// EventAccessedRead notifies when an object is accessed (specifically read/get).
	EventAccessedRead = "ObjectAccessed:Read"
	// EventAccessedStat notifies when an object is accessed (specifically stat).
	EventAccessedStat = "ObjectAccessed:Stat"
)

// EventInfo contains the information of the event that occurred and the source
// IP:PORT of the client which triggerred the event.
type EventInfo struct {
	Time         string
	Size         int64
	UserMetadata map[string]string
	Path         string
	Type         EventType
	Host         string
	Port         string
	UserAgent    string
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
		Events:    []string{"put", "delete"},
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
