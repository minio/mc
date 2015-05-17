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
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio/pkg/iodine"
)

// CreateObject - upload new object to bucket
func (f *fsClient) CreateObject(md5HexString string, size uint64, data io.Reader) error {
	// handle md5HexString match internally
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
	// calculate md5 to verify - incoming md5
	h := md5.New()
	mw := io.MultiWriter(fs, h)

	_, err = io.CopyN(mw, data, int64(size))
	if err != nil {
		return iodine.New(err, nil)
	}

	// ignore invalid md5 string sent by Amazon
	if !strings.Contains(md5HexString, "-") {
		expectedMD5, err := base64.StdEncoding.DecodeString(md5HexString)
		if err != nil {
			return iodine.New(err, nil)
		}
		if !bytes.Equal(expectedMD5, h.Sum(nil)) {
			return iodine.New(errors.New("md5sum mismatch"), nil)
		}
	}
	return nil
}
