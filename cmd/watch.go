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
	"fmt"
	"sync"
	"time"

	"github.com/minio/minio/pkg/probe"
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

// Event contains the information of the event that occurred
type Event struct {
	Time   string    `json:"time"`
	Size   int64     `json:"size"`
	Path   string    `json:"path"`
	Client Client    `json:"-"`
	Type   EventType `json:"type"`
}

// Source obtains the information of the client which generated the event.
type Source struct {
	IP        string `json:"ip"`
	Port      string `json:"port"`
	UserAgent string `json:"userAgent"`
}

type watchParams struct {
	accountID string
	prefix    string
	suffix    string
	events    []string
	recursive bool
}

type watchObject struct {
	// events will be put on this chan
	events chan struct {
		Event  Event
		Source Source
	}
	// errors will be put on this chan
	errors chan *probe.Error
	// will stop the watcher goroutines
	done chan bool
}

// Events returns the chan receiving events
func (w *watchObject) Events() chan struct {
	Event  Event
	Source Source
} {
	return w.events
}

// Errors returns the chan receiving errors
func (w *watchObject) Errors() chan *probe.Error {
	return w.errors
}

// Close the watcher, will stop all goroutines
func (w *watchObject) Close() {
	close(w.done)
}

// Watcher can be used to have one or multiple clients watch for notifications
type Watcher struct {
	sessionStartTime time.Time

	// all errors will be added to this chan
	errorsChan chan *probe.Error
	// all events will be added to this chan
	eventsChan chan struct {
		Event  Event
		Source Source
	}

	// array of watchers joined
	o []*watchObject

	// all watchers joining will enter this waitgroup
	wg sync.WaitGroup
}

// NewWatcher creates a new watcher
func NewWatcher(sessionStartTime time.Time) *Watcher {
	return &Watcher{
		sessionStartTime: sessionStartTime,
		errorsChan:       make(chan *probe.Error),
		eventsChan: make(chan struct {
			Event  Event
			Source Source
		}),
		o: []*watchObject{},
	}
}

// Errors returns a channel which will receive errors
func (w *Watcher) Errors() chan *probe.Error {
	return w.errorsChan
}

// Events returns a channel which will receive events
func (w *Watcher) Events() chan struct {
	Event  Event
	Source Source
} {
	return w.eventsChan
}

// Stop watcher
func (w *Watcher) Stop() {
	// close all running goroutines
	for _, wo := range w.o {
		wo.Close()
	}

	w.wg.Wait()

	close(w.errorsChan)
	close(w.eventsChan)
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
		accountID: fmt.Sprintf("%d", w.sessionStartTime.Unix()),
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

				w.eventsChan <- event
			case err, ok := <-wo.Errors():
				if !ok {
					return
				}

				w.errorsChan <- err
			}
		}
	}()

	return nil
}
