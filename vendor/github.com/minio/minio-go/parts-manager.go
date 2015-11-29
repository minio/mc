/*
 * Minio Go Library for Amazon S3 Compatible Cloud Storage (C) 2015 Minio, Inc.
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

package minio

import (
	"crypto/md5"
	"crypto/sha256"
	"hash"
	"io"
)

// partsManager reads from io.Reader, partitions data into individual partMetadata{}, backed by a
// temporary file which deletes itself upon Close().
//
// This method runs until an EOF or an error occurs. Before returning, the channel is always closed.
func partsManager(reader io.Reader, partSize int64, isEnableSha256Sum bool) <-chan partMetadata {
	ch := make(chan partMetadata, 3)
	go partsManagerInRoutine(reader, partSize, isEnableSha256Sum, ch)
	return ch
}

func partsManagerInRoutine(reader io.Reader, partSize int64, isEnableSha256Sum bool, ch chan<- partMetadata) {
	defer close(ch)
	tmpFile, err := newTempFile("multiparts$")
	if err != nil {
		ch <- partMetadata{
			Err: err,
		}
		return
	}
	var hashMD5 hash.Hash
	var hashSha256 hash.Hash
	var writer io.Writer
	hashMD5 = md5.New()
	mwwriter := io.MultiWriter(hashMD5)
	if isEnableSha256Sum {
		hashSha256 = sha256.New()
		mwwriter = io.MultiWriter(hashMD5, hashSha256)
	}
	writer = io.MultiWriter(tmpFile, mwwriter)
	n, err := io.CopyN(writer, reader, partSize)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		// Seek back to beginning.
		tmpFile.Seek(0, 0)

		// short read, only single partMetadata return.
		partMdata := partMetadata{
			MD5Sum:     hashMD5.Sum(nil),
			ReadCloser: tmpFile,
			Size:       n,
			Err:        nil,
		}
		if isEnableSha256Sum {
			partMdata.Sha256Sum = hashSha256.Sum(nil)
		}
		ch <- partMdata
		return
	}
	// unknown error considered catastrophic error, return here.
	if err != nil {
		ch <- partMetadata{
			Err: err,
		}
		return
	}
	// Seek back to beginning.
	tmpFile.Seek(0, 0)
	partMdata := partMetadata{
		MD5Sum:     hashMD5.Sum(nil),
		ReadCloser: tmpFile,
		Size:       n,
		Err:        nil,
	}
	if isEnableSha256Sum {
		partMdata.Sha256Sum = hashSha256.Sum(nil)
	}
	ch <- partMdata
	for err == nil {
		var n int64
		tmpFile, err = newTempFile("multiparts$")
		if err != nil {
			ch <- partMetadata{
				Err: err,
			}
			return
		}
		hashMD5 = md5.New()
		mwwriter := io.MultiWriter(hashMD5)
		if isEnableSha256Sum {
			hashSha256 = sha256.New()
			mwwriter = io.MultiWriter(hashMD5, hashSha256)
		}
		writer = io.MultiWriter(tmpFile, mwwriter)
		n, err = io.CopyN(writer, reader, partSize)
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF { // catastrophic error
				ch <- partMetadata{
					Err: err,
				}
				return
			}
		}
		// Seek back to beginning.
		tmpFile.Seek(0, 0)
		partMdata := partMetadata{
			MD5Sum:     hashMD5.Sum(nil),
			ReadCloser: tmpFile,
			Size:       n,
			Err:        nil,
		}
		if isEnableSha256Sum {
			partMdata.Sha256Sum = hashSha256.Sum(nil)
		}
		ch <- partMdata
	}
}
