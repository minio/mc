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
	"os"
	"sync"
	"time"

	"github.com/minio/mc/pkg/probe"
	"github.com/minio/minio/pkg/quick"
)

// shareEntryV1 - container for each download/upload entries.
type shareEntryV1 struct {
	URL         string        `json:"share"` // Object URL.
	VersionID   string        `json:"versionID"`
	Date        time.Time     `json:"date"`
	Expiry      time.Duration `json:"expiry"`
	ContentType string        `json:"contentType,omitempty"` // Only used by upload cmd.
}

// JSON file to persist previously shared uploads.
type shareDBV1 struct {
	Version string `json:"version"`
	mutex   *sync.Mutex

	// key is unique share URL.
	Shares map[string]shareEntryV1 `json:"shares"`
}

// Instantiate a new uploads structure for persistence.
func newShareDBV1() *shareDBV1 {
	s := &shareDBV1{
		Version: "1",
	}
	s.Shares = make(map[string]shareEntryV1)
	s.mutex = &sync.Mutex{}
	return s
}

// Set upload info for each share.
func (s *shareDBV1) Set(objectURL string, shareURL string, expiry time.Duration, contentType string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Shares[shareURL] = shareEntryV1{
		URL:         objectURL,
		Date:        UTCNow(),
		Expiry:      expiry,
		ContentType: contentType,
	}
}

// Delete upload info if it exists.
func (s *shareDBV1) Delete(objectURL string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(s.Shares, objectURL)
}

// Delete all expired uploads.
func (s *shareDBV1) deleteAllExpired() {
	for shareURL, share := range s.Shares {
		if (share.Expiry - time.Since(share.Date)) <= 0 {
			// Expired entry. Safe to drop.
			delete(s.Shares, shareURL)
		}
	}
}

// Load shareDB entries from disk. Any entries held in memory are reset.
func (s *shareDBV1) Load(filename string) *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if the db file exist.
	if _, e := os.Stat(filename); e != nil {
		return probe.NewError(e)
	}

	// Initialize and load using quick package.
	qs, e := quick.NewConfig(newShareDBV1(), nil)
	if e != nil {
		return probe.NewError(e).Trace(filename)
	}
	e = qs.Load(filename)
	if e != nil {
		return probe.NewError(e).Trace(filename)
	}

	// Copy map over.
	for k, v := range qs.Data().(*shareDBV1).Shares {
		s.Shares[k] = v
	}

	// Filter out expired entries and save changes back to disk.
	s.deleteAllExpired()
	s.save(filename)

	return nil
}

// Persist share uploads to disk.
func (s shareDBV1) save(filename string) *probe.Error {
	// Initialize a new quick file.
	qs, e := quick.NewConfig(s, nil)
	if e != nil {
		return probe.NewError(e).Trace(filename)
	}
	if e := qs.Save(filename); e != nil {
		return probe.NewError(e).Trace(filename)
	}
	return nil
}

// Persist share uploads to disk.
func (s shareDBV1) Save(filename string) *probe.Error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.save(filename)
}
