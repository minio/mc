// +build darwin freebsd openbsd netbsd

package console

import (
	"syscall"
	"unsafe"
)

/// *BSD console functions
//
// http://fxr.watson.org/fxr/source/sys/ttycom.h?v=FREEBSD6;im=3#L69
//
const ioctlReadTermios = syscall.TIOCGETA

// isatty return true if the file descriptor is terminal.
func isatty(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}
