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

package objectstorage

import (
	"bytes"
	"io"
)

// Part - message structure for results from the MultiPart
type Part struct {
	Data io.ReadSeeker
	Err  error
	Len  int64
	Num  int // part number
}

// MultiPart reads from io.Reader, partitions the data into chunks of given chunksize, and sends
// each chunk as io.ReadSeeker to the caller over a channel
//
// This method runs until an EOF or error occurs. If an error occurs,
// the method sends the error over the channel and returns.
// Before returning, the channel is always closed.
func MultiPart(reader io.Reader, chunkSize uint64) <-chan Part {
	ch := make(chan Part)
	go multiPartInRoutine(reader, chunkSize, ch)
	return ch
}

func multiPartInRoutine(reader io.Reader, chunkSize uint64, ch chan Part) {
	defer close(ch)
	part := make([]byte, chunkSize)
	n, err := io.ReadFull(reader, part)
	if err == io.EOF || err == io.ErrUnexpectedEOF { // short read, only single part return
		ch <- Part{
			Data: bytes.NewReader(part[0:n]),
			Err:  nil,
			Len:  int64(n),
			Num:  1,
		}
		return
	}
	// catastrophic error send error and return
	if err != nil {
		ch <- Part{
			Data: nil,
			Err:  err,
			Num:  0,
		}
		return
	}
	// send the first part
	var num = 1
	ch <- Part{
		Data: bytes.NewReader(part),
		Err:  nil,
		Len:  int64(n),
		Num:  num,
	}
	for err == nil {
		var n int
		part := make([]byte, chunkSize)
		n, err = io.ReadFull(reader, part)
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF { // catastrophic error
				ch <- Part{
					Data: nil,
					Err:  err,
					Num:  0,
				}
				return
			}
		}
		num++
		ch <- Part{
			Data: bytes.NewReader(part[0:n]),
			Err:  nil,
			Len:  int64(n),
			Num:  num,
		}

	}
}
