/*
 * Minio Client (C) 2015 Minio, Inc.
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
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/minio/mc/pkg/client"
	"github.com/minio/minio/pkg/probe"
)

type sortedList struct {
	name string
	file *os.File
	dec  *gob.Decoder
	enc  *gob.Encoder
}

func getSortedListDir() (string, *probe.Error) {
	configDir, err := getMcConfigDir()
	if err != nil {
		return "", err.Trace()
	}
	sortedListDir := filepath.Join(configDir, golbalSortedListDir)
	return sortedListDir, nil
}

func createSortedListDir() *probe.Error {
	sortedListDir, err := getSortedListDir()
	if err != nil {
		return err.Trace()
	}
	if _, err := os.Stat(sortedListDir); err == nil {
		return nil
	}
	if err := os.MkdirAll(sortedListDir, 0700); err != nil {
		return probe.NewError(err)
	}
	return nil
}

// Create create an on disk sorted file from clnt
func (sl *sortedList) Create(clnt client.Client, id string) *probe.Error {
	var e error
	if err := createSortedListDir(); err != nil {
		return err.Trace()
	}
	sortedListDir, err := getSortedListDir()
	if err != nil {
		return err.Trace()
	}
	sl.name = filepath.Join(sortedListDir, id)
	sl.file, e = os.OpenFile(sl.name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0666)
	if e != nil {
		return probe.NewError(e)
	}
	sl.enc = gob.NewEncoder(sl.file)
	sl.dec = gob.NewDecoder(sl.file)
	for content := range clnt.List(true) {
		if content.Err != nil {
			switch err := content.Err.ToGoError().(type) {
			case client.ISBrokenSymlink:
				// FIXME: send the error to caller using channel
				errorIf(content.Err.Trace(), fmt.Sprintf("Skipping broken Symlink ‘%s’.", err.Path))
				continue
			}
			if os.IsNotExist(content.Err.ToGoError()) || os.IsPermission(content.Err.ToGoError()) {
				// FIXME: abstract this at fs.go layer
				if content.Content != nil {
					if content.Content.Type.IsDir() && (content.Content.Type&os.ModeSymlink == os.ModeSymlink) {
						continue
					}
				}
				errorIf(content.Err.Trace(), fmt.Sprintf("Skipping ‘%s’.", content.Content.Name))
				continue
			}
			return content.Err.Trace()
		}
		sl.enc.Encode(*content.Content)
	}
	if _, err := sl.file.Seek(0, os.SEEK_SET); err != nil {
		return probe.NewError(err)
	}
	return nil
}

// List list the entries from the sorted file
func (sl sortedList) List(recursive bool) <-chan client.ContentOnChannel {
	ch := make(chan client.ContentOnChannel)
	go func() {
		defer close(ch)
		for {
			var c client.Content
			err := sl.dec.Decode(&c)
			if err == io.EOF {
				break
			}
			if err != nil {
				ch <- client.ContentOnChannel{Content: nil, Err: probe.NewError(err)}
				break
			}
			ch <- client.ContentOnChannel{Content: &c, Err: nil}
		}
	}()
	return ch
}

// Delete close and delete the ondisk file
func (sl sortedList) Delete() *probe.Error {
	if err := sl.file.Close(); err != nil {
		return probe.NewError(err)
	}
	if err := os.Remove(sl.name); err != nil {
		return probe.NewError(err)
	}
	return nil
}
