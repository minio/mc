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

package fs

import (
	"io"
	"os"
	"path/filepath"

	"github.com/minio/minio/pkg/iodine"
)

// PutObject - create a new file
func (f *fsClient) PutObject(size uint64, data io.Reader) error {
	objectDir, _ := filepath.Split(f.path)
	objectPath := f.path
	if objectDir != "" {
		if err := os.MkdirAll(objectDir, 0700); err != nil {
			return iodine.New(err, nil)
		}
	}
	fs, err := os.Create(objectPath)
	if err != nil {
		return iodine.New(err, nil)
	}
	defer fs.Close()

	_, err = io.CopyN(fs, data, int64(size))
	if err != nil {
		return iodine.New(err, nil)
	}
	return nil
}
