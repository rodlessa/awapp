// Package term provides minimal, dependency-free Linux terminal control:
// raw mode, size queries, SIGWINCH-based resize notifications, and
// non-blocking keyboard input. It talks to the kernel directly via
// syscalls so the resulting binary has zero third-party dependencies
// and works in any ANSI/VT100-compatible terminal emulator.
package term

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

// Size is a terminal's dimensions in character cells.
type Size struct {
	Cols int
	Rows int
}

// Term wraps the controlling terminal and its saved state.
type Term struct {
	fd       int
	orig     syscall.Termios
	in       *bufio.Reader
	resizeCh chan os.Signal
}

// Open puts stdin into raw mode and returns a handle used to read
// input, query size, and restore the terminal on exit.
func Open() (*Term, error) {
	fd := int(os.Stdin.Fd())

	var orig syscall.Termios
	if err := ioctl(fd, syscall.TCGETS, uintptr(unsafe.Pointer(&orig))); err != nil {
		return nil, fmt.Errorf("term: get attrs: %w", err)
	}

	raw := orig
	// cfmakeraw-equivalent: disable canonical mode, echo, signals,
	// input translation; read one byte at a time with no timeout.
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP |
		syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	// VMIN=1, VTIME=0: block until at least one byte is available.
	// (VMIN=0 makes read() return immediately with zero bytes when
	// idle, which starves bufio into ErrNoProgress under a tight
	// read loop — block instead and let the reader goroutine park.)
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := ioctl(fd, syscall.TCSETS, uintptr(unsafe.Pointer(&raw))); err != nil {
		return nil, fmt.Errorf("term: set raw: %w", err)
	}

	t := &Term{
		fd:       fd,
		orig:     orig,
		in:       bufio.NewReaderSize(os.Stdin, 256),
		resizeCh: make(chan os.Signal, 4),
	}
	signal.Notify(t.resizeCh, syscall.SIGWINCH)
	return t, nil
}

// IsTerminal reports whether the given file descriptor is a terminal. It
// is used to decide whether an interactive prompt is possible.
func IsTerminal(fd int) bool {
	var t syscall.Termios
	return ioctl(fd, syscall.TCGETS, uintptr(unsafe.Pointer(&t))) == nil
}

// Restore returns the terminal to its original (cooked) mode.
func (t *Term) Restore() {
	signal.Stop(t.resizeCh)
	_ = ioctl(t.fd, syscall.TCSETS, uintptr(unsafe.Pointer(&t.orig)))
}

// Size returns the current terminal dimensions in character cells.
func (t *Term) Size() (Size, error) {
	var ws struct {
		Row, Col, Xpixel, Ypixel uint16
	}
	if err := ioctl(t.fd, syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&ws))); err != nil {
		return Size{}, err
	}
	cols, rows := int(ws.Col), int(ws.Row)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return Size{Cols: cols, Rows: rows}, nil
}

// Resized returns the channel that fires (SIGWINCH) whenever the
// terminal is resized.
func (t *Term) Resized() <-chan os.Signal { return t.resizeCh }

// ReadByte reads one byte from stdin, blocking until input arrives.
// Meant to be run in its own goroutine feeding a channel; VMIN=0 means
// individual reads can return 0 bytes, so this loops internally.
func (t *Term) ReadByte() (byte, error) {
	for {
		b, err := t.in.ReadByte()
		if err == nil {
			return b, nil
		}
		if !isAgain(err) {
			return 0, err
		}
	}
}

func isAgain(err error) bool {
	return err == syscall.EAGAIN || err == syscall.EINTR
}

func ioctl(fd int, req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

// --- Screen / cursor control (ANSI, works on any VT100+ terminal) ---

const (
	AltScreenOn  = "\x1b[?1049h"
	AltScreenOff = "\x1b[?1049l"
	CursorHide   = "\x1b[?25l"
	CursorShow   = "\x1b[?25h"
	ClearScreen  = "\x1b[2J\x1b[H"
)

// EnterFullscreen switches to the terminal's alternate screen buffer
// and hides the cursor, leaving the user's shell scrollback untouched.
func EnterFullscreen() {
	fmt.Print(AltScreenOn + CursorHide + ClearScreen)
}

// ExitFullscreen restores the cursor and switches back to the normal
// screen buffer.
func ExitFullscreen() {
	fmt.Print(CursorShow + AltScreenOff)
}
