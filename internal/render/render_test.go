package render

import (
	"strings"
	"testing"
)

func TestFrameMonochromeHasNoColorCodes(t *testing.T) {
	b := NewBuffer(10, 4)
	b.Set(1, 1, 'x', 120, true)
	b.Set(3, 2, 'o', 0, false)
	f := b.Frame()
	if strings.Contains(f, "38;5;") {
		t.Fatalf("monochrome frame contains a color code: %q", f)
	}
	if !strings.Contains(f, "x") || !strings.Contains(f, "o") {
		t.Fatalf("monochrome frame missing characters: %q", f)
	}
}

func TestFrameColorHasCodes(t *testing.T) {
	b := NewBuffer(10, 4)
	b.Color = true
	b.Set(1, 1, 'x', 120, true)
	f := b.Frame()
	if !strings.Contains(f, "\x1b[1;38;5;120m") {
		t.Fatalf("colored frame missing bold color code: %q", f)
	}
	if !strings.Contains(f, "38;5;0m") {
		t.Fatalf("colored frame missing a color code: %q", f)
	}
}

func TestFrameEveryRowPresent(t *testing.T) {
	b := NewBuffer(6, 3)
	b.Set(0, 0, 'a', 0, false)
	b.Set(5, 2, 'z', 0, false)
	f := b.Frame()
	for _, want := range []string{"\x1b[1;1H", "\x1b[2;1H", "\x1b[3;1H", "a", "z"} {
		if !strings.Contains(f, want) {
			t.Fatalf("frame missing %q: %q", want, f)
		}
	}
}
