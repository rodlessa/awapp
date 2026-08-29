//go:build linux

package term

import "syscall"

// Linux names the termios ioctl requests TCGETS/TCSETS in Go's stdlib
// syscall package.
const (
	TCGETS = syscall.TCGETS
	TCSETS = syscall.TCSETS
)
