/*
 * Mini Copy, (C) 2015 Minio, Inc.
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

package s3

import (
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/minio-io/minio/pkg/iodine"
)

/// Object Operations PUT - keeping this in a separate file for readability

// Put - upload new object to bucket
func (c *s3Client) Put(bucket, key, md5HexString string, size int64) (io.WriteCloser, error) {
	r, w := io.Pipe()
	blockingWriter := NewBlockingWriteCloser(w)
	go func() {
		req := newReq(c.keyURL(bucket, key), c.UserAgent, r)
		req.Method = "PUT"
		req.ContentLength = size

		// set Content-MD5 only if md5 is provided
		if strings.TrimSpace(md5HexString) != "" {
			md5, err := hex.DecodeString(md5HexString)
			if err != nil {
				err := iodine.New(err, nil)
				r.CloseWithError(err)
				blockingWriter.Release(err)
				return
			}
			req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(md5))
		}
		c.signRequest(req, c.Host)

		client := http.Client{}
		res, err := client.Do(req)

		if err != nil {
			err := iodine.New(err, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}

		if res.StatusCode != http.StatusOK {
			err := iodine.New(err, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		blockingWriter.Release(nil)
		r.Close()
	}()
	return blockingWriter, nil
}

// BlockingWriteCloser is a WriteCloser that blocks until released
type BlockingWriteCloser struct {
	w       io.WriteCloser
	release *sync.WaitGroup
	err     error
}

// Write to the underlying writer
func (b *BlockingWriteCloser) Write(p []byte) (int, error) {
	n, err := b.w.Write(p)
	err = iodine.New(err, nil)
	return n, err
}

// Close blocks until another goroutine calls Release(error). Returns error code if either
// writer fails or Release is called with an error.
func (b *BlockingWriteCloser) Close() error {
	err := b.w.Close()
	if err != nil {
		b.err = err
	}
	b.release.Wait()
	return b.err
}

// Release the Close, causing it to unblock. Only call this once. Calling it multiple times results in a panic.
func (b *BlockingWriteCloser) Release(err error) {
	b.release.Done()
	if err != nil {
		b.err = err
	}
}

// NewBlockingWriteCloser Creates a new write closer that must be released by the read consumer.
func NewBlockingWriteCloser(w io.WriteCloser) *BlockingWriteCloser {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &BlockingWriteCloser{w: w, release: wg}
}
