//go:build windows

package term

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// Windows console implementation. Talks to kernel32 directly (no third-
// party deps): puts the console into a raw-ish mode, enables ANSI/VT
// processing so the app's escape-sequence rendering works, reads keys
// from stdin, and polls for resize (Windows has no SIGWINCH).

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode      = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode      = kernel32.NewProc("SetConsoleMode")
	procGetScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

// Console mode flags (wincon.h).
const (
	enableProcessedInput      = 0x0001
	enableLineInput           = 0x0002
	enableEchoInput           = 0x0004
	enableProcessedOutput     = 0x0001
	enableVirtualTerminalProc = 0x0004
)

// Term wraps the Windows console handles and their saved modes.
type Term struct {
	in         *bufio.Reader
	inMode     uint32
	outMode    uint32
	inHandle   syscall.Handle
	outHandle  syscall.Handle
	resizeCh   chan os.Signal
	cols, rows int
}

type coord struct{ X, Y int16 }

type smallRect struct{ Left, Top, Right, Bottom int16 }

// consoleScreenBufferInfo mirrors CONSOLE_SCREEN_BUFFER_INFO.
type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

func getConsoleMode(h syscall.Handle) (uint32, error) {
	var m uint32
	r, _, e := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&m)))
	if r == 0 {
		return 0, e
	}
	return m, nil
}

func setConsoleMode(h syscall.Handle, m uint32) error {
	r, _, e := procSetConsoleMode.Call(uintptr(h), uintptr(m))
	if r == 0 {
		return e
	}
	return nil
}

func sizeNow(h syscall.Handle) (int, int) {
	var info consoleScreenBufferInfo
	r, _, _ := procGetScreenBufferInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 80, 24
	}
	cols, rows := int(info.Size.X), int(info.Size.Y)
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return cols, rows
}

// Open puts stdin into a raw-ish mode, enables VT processing on stdout
// (so ANSI alt-screen/colour codes work), and returns a handle used to
// read input, query size, and restore the terminal on exit.
func Open() (*Term, error) {
	in := syscall.Handle(os.Stdin.Fd())
	out := syscall.Handle(os.Stdout.Fd())

	inMode, err := getConsoleMode(in)
	if err != nil {
		return nil, fmt.Errorf("term: not a console: %w", err)
	}
	outMode, _ := getConsoleMode(out)

	// Raw-ish input: no line buffering, no echo.
	_ = setConsoleMode(in, inMode&^(enableLineInput|enableEchoInput|enableProcessedInput))
	// Best-effort VT processing on output (Windows 10+).
	_ = setConsoleMode(out, outMode|enableProcessedOutput|enableVirtualTerminalProc)

	cols, rows := sizeNow(out)
	t := &Term{
		in:        bufio.NewReaderSize(os.Stdin, 256),
		inMode:    inMode,
		outMode:   outMode,
		inHandle:  in,
		outHandle: out,
		resizeCh:  make(chan os.Signal, 4),
		cols:      cols,
		rows:      rows,
	}
	go t.watchResize()
	return t, nil
}

// IsTerminal reports whether the given file descriptor is a console.
func IsTerminal(fd int) bool {
	_, err := getConsoleMode(syscall.Handle(fd))
	return err == nil
}

// Restore returns the console to its original mode.
func (t *Term) Restore() {
	_ = setConsoleMode(t.inHandle, t.inMode)
	_ = setConsoleMode(t.outHandle, t.outMode)
}

// Size returns the current console dimensions in character cells.
func (t *Term) Size() (Size, error) {
	cols, rows := sizeNow(t.outHandle)
	t.cols, t.rows = cols, rows
	return Size{Cols: cols, Rows: rows}, nil
}

// Resized returns a channel that fires (with a dummy signal) whenever
// the console size changes, detected by polling.
func (t *Term) Resized() <-chan os.Signal { return t.resizeCh }

// watchResize polls the console size and pings the resize channel when
// it changes (Windows has no SIGWINCH).
func (t *Term) watchResize() {
	for {
		time.Sleep(200 * time.Millisecond)
		c, r := sizeNow(t.outHandle)
		if c != t.cols || r != t.rows {
			t.cols, t.rows = c, r
			select {
			case t.resizeCh <- os.Interrupt:
			default:
			}
		}
	}
}

// ReadByte reads one byte from stdin, blocking until input arrives.
// The app runs this in its own goroutine feeding a channel.
func (t *Term) ReadByte() (byte, error) {
	return t.in.ReadByte()
}
