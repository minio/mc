// +build windows

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

package console

import (
	"syscall"
	"unsafe"
)

/// Console functions
//
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms683167%28v=vs.85%29.aspx
var kernel32 = syscall.NewLazyDLL("kernel32.dll") // ugly DLL loading but cannot do much on windows
var procGetConsoleMode = kernel32.NewProc("GetConsoleMode")

// isatty return true if the file descriptor is terminal.
func isatty(fd uintptr) bool {
	var st uint32
	r, _, e := syscall.Syscall(procGetConsoleMode.Addr(), 2, fd, uintptr(unsafe.Pointer(&st)), 0)
	return r != 0 && e == 0
}
