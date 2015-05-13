/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
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
	"net"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/iodine"
)

// isValidRetry - check if we should retry for the given error sequence
func isValidRetry(err error) bool {
	err = iodine.New(err, nil)
	if err == nil {
		return false
	}
	// DNSError, Network Operation error
	switch e := iodine.ToError(err).(type) {
	case *net.AddrError:
		return true
	case *net.DNSError:
		return true
	case *net.OpError:
		switch e.Op {
		case "read", "write", "dial":
			return true
		}
	}
	return false
}

// StartBar -- instantiate a progressbar
func startBar(size int64) *pb.ProgressBar {
	bar := pb.New64(size)
	bar.SetUnits(pb.U_BYTES)
	bar.SetRefreshRate(time.Millisecond * 10)
	bar.NotPrint = true
	bar.ShowSpeed = true
	bar.Callback = func(s string) {
		// Colorize
		console.Print("\r" + s)
	}
	// Feels like wget
	bar.Format("[=> ]")
	return bar
}
