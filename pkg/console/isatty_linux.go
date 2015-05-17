// +build linux

package console

import (
	"syscall"
	"unsafe"
)

// Standard ioctls from Linux - /usr/include/asm-generic/ioctls.h:#define TCGETS  0x5401
const ioctlReadTermios = syscall.TCGETS

// isatty return true if the file descriptor is terminal.
func isatty(fd uintptr) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, ioctlReadTermios, uintptr(unsafe.Pointer(&termios)), 0, 0, 0)
	return err == 0
}
