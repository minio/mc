// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

/*
 * Mini Copy (C) 2015 Minio, Inc.
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

package fs

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/minio-io/mc/pkg/client"
	"github.com/minio-io/minio/pkg/iodine"
)

// Put - upload new object to bucket
func (f *fsClient) Put(md5HexString string, size int64) (io.WriteCloser, error) {
	r, w := io.Pipe()
	blockingWriter := client.NewBlockingWriteCloser(w)
	go func() {
		// handle md5HexString match internally
		if size < 0 {
			err := iodine.New(client.InvalidArgument{Err: errors.New("invalid argument")}, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		objectDir, _ := filepath.Split(f.path)
		objectPath := f.path
		if objectDir != "" {
			if err := os.MkdirAll(objectDir, 0700); err != nil {
				err := iodine.New(err, nil)
				r.CloseWithError(err)
				blockingWriter.Release(err)
				return
			}
		}
		fs, err := os.Create(objectPath)
		if err != nil {
			err := iodine.New(err, nil)
			r.CloseWithError(err)
			blockingWriter.Release(err)
			return
		}
		_, err = io.CopyN(fs, r, size)
		if err != nil {
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
