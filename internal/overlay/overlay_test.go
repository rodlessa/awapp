package overlay

import (
	"strings"
	"testing"

	"weatherterm/internal/render"
)

// Truncating a panel line on a narrow terminal must never split a
// multi-byte UTF-8 rune — byte-based slicing used to produce U+FFFD
// replacement glyphs exactly on the smallest terminals.
func TestPanelTruncationNoReplacement(t *testing.T) {
	buf := render.NewBuffer(7, 10)
	Draw(buf, Info{
		HasData:    true,
		HasTemp:    true,
		City:       "São",
		Desc:       "light rain",
		TempC:      26.5,
		Fahrenheit: false,
		Live:       true,
	})
	if strings.ContainsRune(buf.Text(), '\uFFFD') {
		t.Fatalf("panel truncation produced a replacement glyph:\n%s", buf.Text())
	}
	// The truncated city keeps its accented rune intact (rune-safe cut).
	if !strings.Contains(buf.Text(), "Sã") {
		t.Fatalf("expected rune-safe truncation of the city, got:\n%s", buf.Text())
	}
}

// The panel should report the season and whether the leaf layer is on.
func TestPanelShowsSeasonAndLeaves(t *testing.T) {
	lines := panelLines(Info{
		HasData:  true,
		HasTemp:  true,
		City:     "Fortaleza",
		Desc:     "light rain",
		TempC:    26.5,
		Live:     true,
		Season:   "winter",
		LeavesOn: true,
	})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Season: winter") {
		t.Errorf("panel should show the season, got:\n%s", joined)
	}
	if !strings.Contains(joined, "leaves on") {
		t.Errorf("panel should show leaves on, got:\n%s", joined)
	}
	// Leaves off -> "leaves off"
	off := strings.Join(panelLines(Info{Season: "summer", LeavesOn: false}), "\n")
	if !strings.Contains(off, "leaves off") {
		t.Errorf("panel should show leaves off, got:\n%s", off)
	}
}

// The error line (why the app is offline) must be rendered and counted
// in the panel height so the Sun/Moon arc reserves room for it.
func TestPanelErrorLineIncluded(t *testing.T) {
	info := Info{Err: "openweathermap: HTTP 401: bad key"}
	lines := panelLines(info)
	found := false
	for _, l := range lines {
		if strings.Contains(l, "401") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("error line not rendered in the panel: %q", lines)
	}
	if PanelHeight(info) != len(lines)+2 {
		t.Fatalf("PanelHeight mismatch: %d vs %d", PanelHeight(info), len(lines)+2)
	}
}
