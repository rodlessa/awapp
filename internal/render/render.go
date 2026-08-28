// Package render implements a small double-buffered character grid
// that is flattened to a single ANSI escape-sequence string per frame,
// using 256-color codes (\x1b[38;5;Nm) for broad terminal-emulator
// compatibility rather than assuming truecolor support.
package render

import (
	"strconv"
	"strings"
)

type Cell struct {
	Ch   rune
	FG   uint8
	Bold bool
	set  bool
}

// Buffer is a W x H grid of cells rebuilt fresh every frame by the
// animators, then flattened into one ANSI string for a single write.
// When Color is false the frame is emitted as plain monochrome text
// (no color escape codes) so it renders identically on any terminal.
type Buffer struct {
	W, H  int
	Color bool
	cells []Cell
}

func NewBuffer(w, h int) *Buffer {
	b := &Buffer{}
	b.Resize(w, h)
	return b
}

func (b *Buffer) Resize(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	b.W, b.H = w, h
	b.cells = make([]Cell, w*h)
}

// Clear blanks every cell to a space with the given foreground (used
// as the "background" tone, since we don't paint terminal backgrounds).
func (b *Buffer) Clear(fg uint8) {
	for i := range b.cells {
		b.cells[i] = Cell{Ch: ' ', FG: fg, set: true}
	}
}

// Set paints a single cell if the coordinates are on-screen. Off-screen
// writes are silently ignored so animators don't need bounds checks.
func (b *Buffer) Set(x, y int, ch rune, fg uint8, bold bool) {
	if x < 0 || y < 0 || x >= b.W || y >= b.H {
		return
	}
	b.cells[y*b.W+x] = Cell{Ch: ch, FG: fg, Bold: bold, set: true}
}

// SetString paints a horizontal run of text starting at x,y, clipped
// to the buffer bounds.
func (b *Buffer) SetString(x, y int, s string, fg uint8, bold bool) {
	for i, r := range []rune(s) {
		b.Set(x+i, y, r, fg, bold)
	}
}

// Frame renders the buffer to one ANSI string: cursor to home, then
// row by row. In color mode it only emits a color/bold escape when the
// value actually changes from the previous cell written; in monochrome
// mode it writes the plain characters with no color codes at all.
func (b *Buffer) Frame() string {
	var sb strings.Builder
	sb.Grow(len(b.cells)*4 + 64)
	sb.WriteString("\x1b[H")

	if !b.Color {
		for y := 0; y < b.H; y++ {
			sb.WriteString("\x1b[")
			sb.WriteString(strconv.Itoa(y + 1))
			sb.WriteString(";1H")
			for x := 0; x < b.W; x++ {
				c := b.cells[y*b.W+x]
				if !c.set {
					c = Cell{Ch: ' '}
				}
				sb.WriteRune(c.Ch)
			}
		}
		sb.WriteString("\x1b[0m")
		return sb.String()
	}

	lastFG := int(-1)
	lastBold := false
	for y := 0; y < b.H; y++ {
		sb.WriteString("\x1b[")
		sb.WriteString(strconv.Itoa(y + 1))
		sb.WriteString(";1H")
		// SGR state survives the cursor move, so don't reset per row:
		// only emit a color code when it actually changes.
		for x := 0; x < b.W; x++ {
			c := b.cells[y*b.W+x]
			if !c.set {
				c = Cell{Ch: ' ', FG: 0}
			}
			if int(c.FG) != lastFG || c.Bold != lastBold {
				sb.WriteString("\x1b[")
				if c.Bold {
					sb.WriteString("1;")
				} else {
					sb.WriteString("0;")
				}
				sb.WriteString("38;5;")
				sb.WriteString(strconv.Itoa(int(c.FG)))
				sb.WriteByte('m')
				lastFG = int(c.FG)
				lastBold = c.Bold
			}
			sb.WriteRune(c.Ch)
		}
	}
	sb.WriteString("\x1b[0m")
	return sb.String()
}

// Text returns the buffer contents as plain text (no ANSI codes), one
// line per row. Handy for debugging and for non-terminal previews.
func (b *Buffer) Text() string {
	var sb strings.Builder
	for y := 0; y < b.H; y++ {
		for x := 0; x < b.W; x++ {
			c := b.cells[y*b.W+x]
			if !c.set {
				c = Cell{Ch: ' '}
			}
			sb.WriteRune(c.Ch)
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// Box draws a single-line-style rectangle outline (for the status
// overlay), width/height in cells, top-left at x,y.
func (b *Buffer) Box(x, y, w, h int, fg uint8) {
	if w < 2 || h < 2 {
		return
	}
	b.Set(x, y, '┌', fg, false)
	b.Set(x+w-1, y, '┐', fg, false)
	b.Set(x, y+h-1, '└', fg, false)
	b.Set(x+w-1, y+h-1, '┘', fg, false)
	for i := 1; i < w-1; i++ {
		b.Set(x+i, y, '─', fg, false)
		b.Set(x+i, y+h-1, '─', fg, false)
	}
	for i := 1; i < h-1; i++ {
		b.Set(x, y+i, '│', fg, false)
		b.Set(x+w-1, y+i, '│', fg, false)
	}
}

// FillBox paints a filled rectangle with spaces (used to clear behind
// the overlay before drawing its border and text over the animation).
func (b *Buffer) FillBox(x, y, w, h int, fg uint8) {
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			b.Set(x+i, y+j, ' ', fg, false)
		}
	}
}
