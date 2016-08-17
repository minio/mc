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

package mc

import (
	"fmt"
	"sync"
	"time"

	"github.com/minio/minio/pkg/probe"
)

// EventType is the type of the event that occurred
type EventType string

const (
	// EventCreate notifies when a new object has been created
	EventCreate EventType = "ObjectCreated"
	// EventRemove notifies when a new object has been deleted
	EventRemove = "ObjectRemoved"
)

// Event contains the information of the event that occurred
type Event struct {
	Time   time.Time `json:"time"`
	Path   string    `json:"path"`
	Client Client    `json:"-"`
	Type   EventType `json:"type"`
}

type watchParams struct {
	accountID string
	prefix    string
	suffix    string
	events    string
	recursive bool
}

type watchObject struct {
	// events will be put on this chan
	events chan Event
	// errors will be put on this chan
	errors chan *probe.Error
	// will stop the watcher goroutines
	done chan bool
}

// Watcher can be used to have one or multiple clients watch for notifications
type Watcher struct {
	sessionStartTime time.Time

	// all errors will be added to this chan
	errorsChan chan *probe.Error
	// all events will be added to this chan
	eventsChan chan Event

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
		eventsChan:       make(chan Event),
		o:                []*watchObject{},
	}
}

// Errors returns a channel which will receive errors
func (w *Watcher) Errors() chan *probe.Error {
	return w.errorsChan
}

// Events returns a channel which will receive events
func (w *Watcher) Events() chan Event {
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

// Unjoin the watcher from client.
func (w *Watcher) Unjoin(client Client, recursive bool) *probe.Error {
	err := client.Unwatch(watchParams{
		recursive: recursive,
		accountID: fmt.Sprintf("%d", w.sessionStartTime.Unix()),
	})
	if err != nil {
		return err
	}
	return nil
}

// Join the watcher with client
func (w *Watcher) Join(client Client, recursive bool) *probe.Error {
	wo, err := client.Watch(watchParams{
		recursive: recursive,
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
