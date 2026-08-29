//go:build darwin

package term

import "syscall"

// macOS names the termios ioctl requests TIOCGETA/TIOCSETA (Linux calls
// them TCGETS/TCSETS). Alias them so term_unix.go compiles on both.
const (
	TCGETS = syscall.TIOCGETA
	TCSETS = syscall.TIOCSETA
)
