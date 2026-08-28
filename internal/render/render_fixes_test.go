package render

import (
	"strings"
	"testing"
)

// SGR state persists across cursor moves, so a solid-color buffer should
// emit exactly one color escape, not one per row (the old code reset
// lastFG at the top of every row, wasting bytes per frame).
func TestFrameSGRStateAcrossRows(t *testing.T) {
	b := NewBuffer(8, 3)
	b.Color = true
	b.Clear(120)
	f := b.Frame()
	if n := strings.Count(f, "38;5;"); n != 1 {
		t.Fatalf("solid-color frame should emit exactly one color code, got %d in %q", n, f)
	}
}

// Two rows of different colors still need (and get) their own escapes.
func TestFrameColorChangeAcrossRows(t *testing.T) {
	b := NewBuffer(8, 2)
	b.Color = true
	for x := 0; x < b.W; x++ {
		b.Set(x, 0, 'a', 10, false)
		b.Set(x, 1, 'b', 20, false)
	}
	f := b.Frame()
	if n := strings.Count(f, "38;5;"); n != 2 {
		t.Fatalf("two different-color rows should emit two color codes, got %d in %q", n, f)
	}
	if !strings.Contains(f, "38;5;10m") || !strings.Contains(f, "38;5;20m") {
		t.Fatalf("missing expected colors in %q", f)
	}
}
