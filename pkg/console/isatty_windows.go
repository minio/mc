// +build windows
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
