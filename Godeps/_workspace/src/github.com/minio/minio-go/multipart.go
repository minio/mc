/*
 * Minimal object storage library (C) 2015 Minio, Inc.
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

package client

import (
	"bytes"
	"io"
)

// part - message structure for results from the MultiPart
type part struct {
	Data io.ReadSeeker
	Err  error
	Len  int64
	Num  int // part number
}

// multiPart reads from io.Reader, partitions the data into chunks of given chunksize, and sends
// each chunk as io.ReadSeeker to the caller over a channel
//
// This method runs until an EOF or error occurs. If an error occurs,
// the method sends the error over the channel and returns.
// Before returning, the channel is always closed.
//
// additionally this function also skips list of parts if provided
func multiPart(reader io.Reader, chunkSize uint64, skipParts []int) <-chan part {
	ch := make(chan part)
	go multiPartInRoutine(reader, chunkSize, skipParts, ch)
	return ch
}

func multiPartInRoutine(reader io.Reader, chunkSize uint64, skipParts []int, ch chan part) {
	defer close(ch)
	p := make([]byte, chunkSize)
	n, err := io.ReadFull(reader, p)
	if err == io.EOF || err == io.ErrUnexpectedEOF { // short read, only single part return
		ch <- part{
			Data: bytes.NewReader(p[0:n]),
			Err:  nil,
			Len:  int64(n),
			Num:  1,
		}
		return
	}
	// catastrophic error send error and return
	if err != nil {
		ch <- part{
			Data: nil,
			Err:  err,
			Num:  0,
		}
		return
	}
	// send the first part
	var num = 1
	if !isPartNumberUploaded(num, skipParts) {
		ch <- part{
			Data: bytes.NewReader(p),
			Err:  nil,
			Len:  int64(n),
			Num:  num,
		}
	}
	for err == nil {
		var n int
		p := make([]byte, chunkSize)
		n, err = io.ReadFull(reader, p)
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF { // catastrophic error
				ch <- part{
					Data: nil,
					Err:  err,
					Num:  0,
				}
				return
			}
		}
		num++
		if isPartNumberUploaded(num, skipParts) {
			continue
		}
		ch <- part{
			Data: bytes.NewReader(p[0:n]),
			Err:  nil,
			Len:  int64(n),
			Num:  num,
		}

	}
}

// to verify if partNumber is part of the skip part list
func isPartNumberUploaded(partNumber int, skipParts []int) bool {
	for _, part := range skipParts {
		if part == partNumber {
			return true
		}
	}
	return false
}
