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
	"bytes"
	"crypto/md5"
	"io"
)

// piece - container for each piece
type piece struct {
	MD5Sum     []byte
	ReadSeeker io.ReadSeeker
	Err        error
	Len        int64
	Num        int // part number
}

// skipPiece - container for skip piece
type skipPiece struct {
	md5sum      []byte
	pieceNumber int
}

// chopper reads from io.Reader, partitions the data into pieces of given pieceSize, and sends
// each piece as io.ReadSeeker to the caller over a channel
//
// This method runs until an EOF or an error occurs. Before returning, the channel is always closed.
//
// Additionally this function also skips list of pieces if provided
func chopper(reader io.Reader, pieceSize int64, skipPieces []skipPiece) <-chan piece {
	ch := make(chan piece, 3)
	go chopperInRoutine(reader, pieceSize, skipPieces, ch)
	return ch
}

func chopperInRoutine(reader io.Reader, pieceSize int64, skipPieces []skipPiece, ch chan<- piece) {
	defer close(ch)
	p := make([]byte, pieceSize)
	n, err := io.ReadFull(reader, p)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		// short read, only single piece return.
		m := md5.Sum(p[0:n])
		ch <- piece{
			MD5Sum:     m[:],
			ReadSeeker: bytes.NewReader(p[0:n]),
			Err:        nil,
			Len:        int64(n),
			Num:        1,
		}
		return
	}
	// unknown error considered catastrophic error, return here.
	if err != nil {
		ch <- piece{
			ReadSeeker: nil,
			Err:        err,
			Num:        0,
		}
		return
	}
	// send the first piece
	var num = 1
	md5SumBytes := md5.Sum(p)
	sp := skipPiece{
		pieceNumber: num,
		md5sum:      md5SumBytes[:],
	}
	if !isPieceNumberUploaded(sp, skipPieces) {
		ch <- piece{
			MD5Sum:     md5SumBytes[:],
			ReadSeeker: bytes.NewReader(p),
			Err:        nil,
			Len:        int64(n),
			Num:        num,
		}
	}
	for err == nil {
		var n int
		p := make([]byte, pieceSize)
		n, err = io.ReadFull(reader, p)
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF { // catastrophic error
				ch <- piece{
					ReadSeeker: nil,
					Err:        err,
					Num:        0,
				}
				return
			}
		}
		num++
		md5SumBytes := md5.Sum(p[0:n])
		sp := skipPiece{
			pieceNumber: num,
			md5sum:      md5SumBytes[:],
		}
		if isPieceNumberUploaded(sp, skipPieces) {
			continue
		}
		ch <- piece{
			MD5Sum:     md5SumBytes[:],
			ReadSeeker: bytes.NewReader(p[0:n]),
			Err:        nil,
			Len:        int64(n),
			Num:        num,
		}

	}
}

// to verify if piece is already uploaded
func isPieceNumberUploaded(piece skipPiece, skipPieces []skipPiece) bool {
	for _, p := range skipPieces {
		if p.pieceNumber == piece.pieceNumber && bytes.Equal(p.md5sum, piece.md5sum) {
			return true
		}
	}
	return false
}
