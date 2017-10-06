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

package cmd

import (
	"sync"
	"time"

	"github.com/minio/mc/pkg/probe"
)

// EventType represents the type of the event occurred.
type EventType string

const (
	// EventCreate notifies when a new object is created
	EventCreate EventType = "ObjectCreated"
	// EventRemove notifies when a new object is deleted
	EventRemove = "ObjectRemoved"
	// EventAccessed notifies when an object is accessed.
	EventAccessed = "ObjectAccessed"
	// EventAccessedRead notifies when an object is accessed (specifically read).
	EventAccessedRead = "ObjectAccessed:Read"
	// EventAccessedStat notifies when an object is accessed (specifically stat).
	EventAccessedStat = "ObjectAccessed:Stat"
)

// EventInfo contains the information of the event that occurred and the source
// IP:PORT of the client which triggerred the event.
type EventInfo struct {
	Time      string
	Size      int64
	Path      string
	Type      EventType
	Host      string
	Port      string
	UserAgent string
}

type watchParams struct {
	prefix    string
	suffix    string
	events    []string
	recursive bool
}

type watchObject struct {
	// eventInfo will be put on this chan
	eventInfoChan chan EventInfo
	// errors will be put on this chan
	errorChan chan *probe.Error
	// will stop the watcher goroutines
	doneChan chan bool
}

// Events returns the chan receiving events
func (w *watchObject) Events() chan EventInfo {
	return w.eventInfoChan
}

// Errors returns the chan receiving errors
func (w *watchObject) Errors() chan *probe.Error {
	return w.errorChan
}

// Close the watcher, will stop all goroutines
func (w *watchObject) Close() {
	close(w.doneChan)
}

// Watcher can be used to have one or multiple clients watch for notifications
type Watcher struct {
	sessionStartTime time.Time

	// all error will be added to this chan
	errorChan chan *probe.Error
	// all events will be added to this chan
	eventInfoChan chan EventInfo

	// array of watchers joined
	o []*watchObject

	// all watchers joining will enter this waitgroup
	wg sync.WaitGroup
}

// NewWatcher creates a new watcher
func NewWatcher(sessionStartTime time.Time) *Watcher {
	return &Watcher{
		sessionStartTime: sessionStartTime,
		errorChan:        make(chan *probe.Error),
		eventInfoChan:    make(chan EventInfo),
		o:                []*watchObject{},
	}
}

// Errors returns a channel which will receive errors
func (w *Watcher) Errors() chan *probe.Error {
	return w.errorChan
}

// Events returns a channel which will receive events
func (w *Watcher) Events() chan EventInfo {
	return w.eventInfoChan
}

// Stop watcher
func (w *Watcher) Stop() {
	// close all running goroutines
	for _, wo := range w.o {
		wo.Close()
	}

	w.wg.Wait()

	close(w.errorChan)
	close(w.eventInfoChan)
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
func (w *Watcher) Join(client Client, recursive bool) *probe.Error {
	wo, err := client.Watch(watchParams{
		recursive: recursive,
		events:    []string{"put", "delete"},
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
			case event, ok := <-wo.Events():
				if !ok {
					return
				}

				w.eventInfoChan <- event
			case err, ok := <-wo.Errors():
				if !ok {
					return
				}

				w.errorChan <- err
			}
		}
	}()

	return nil
}
