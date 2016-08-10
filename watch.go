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
	"sync"

	"github.com/minio/minio/pkg/probe"
)

// EventType is the type of the event that occured
type EventType uint8

const (
	// EventCreate notifies when a new object has been created
	EventCreate EventType = iota
	// EventRemove notifies when a new object has been deleted
	EventRemove
)

// Event contains the information of the event that occured
type Event struct {
	Path   string
	Client Client
	Type   EventType
}

// ClientWatcher is the interface being implemented by the different clients
type ClientWatcher interface {
	Watch(recursive bool) (*watchObject, *probe.Error)
}

// Watcher can be used to have one or multiple clients watch for notifications
type Watcher struct {
	errorsChan chan *probe.Error
	eventsChan chan Event

	o  []*watchObject
	wg sync.WaitGroup
}

// NewWatcher creates a new watcher
func NewWatcher() *Watcher {
	return &Watcher{
		errorsChan: make(chan *probe.Error),
		eventsChan: make(chan Event),

		o: []*watchObject{},
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

// Join the watcher with client
func (w *Watcher) Join(client Client, recursive bool) *probe.Error {
	cw, ok := client.(ClientWatcher)
	if !ok {
		return probe.NewError(fmt.Errorf("Client has no Watcher interface."))
	}

	wo, err := cw.Watch(recursive)
	if err != nil {
		return err
	}

	w.o = append(w.o, wo)

	// join monitoring waitgroup
	w.wg.Add(1)
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
