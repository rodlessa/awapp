// Package term provides minimal, dependency-free terminal control: raw
// mode, size queries, resize notifications, and keyboard input. The
// platform-specific pieces live in term_unix.go (Linux/macOS syscalls)
// and term_windows.go (Windows console API); this file holds the shared,
// cross-platform parts. Works in any ANSI/VT100-compatible terminal.
package term

import "fmt"

// Size is a terminal's dimensions in character cells.
type Size struct {
	Cols int
	Rows int
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
