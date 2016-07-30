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

type EventType uint8

const (
	EventCreate EventType = iota
	EventRemove
)

type Event struct {
	Path   string
	Client Client
	Type   EventType
}

type ClientWatcher interface {
	Watch(recursive bool) (*watchObject, *probe.Error)
}

type watcher struct {
	errorsChan chan *probe.Error
	eventsChan chan Event

	o  []*watchObject
	wg sync.WaitGroup
}

func NewWatcher() *watcher {
	return &watcher{
		errorsChan: make(chan *probe.Error),
		eventsChan: make(chan Event),

		o: []*watchObject{},
	}
}

func (w *watcher) Errors() chan *probe.Error {
	return w.errorsChan
}

func (w *watcher) Events() chan Event {
	return w.eventsChan
}

func (w *watcher) Stop() {
	// close all running goroutines
	for _, wo := range w.o {
		wo.Close()
	}

	w.wg.Wait()

	close(w.errorsChan)
	close(w.eventsChan)
}

func (w *watcher) Watching() bool {
	return (len(w.o) > 0)
}

func (w *watcher) Wait() {
	w.wg.Wait()
}

func (w *watcher) Join(client Client, recursive bool) *probe.Error {
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
