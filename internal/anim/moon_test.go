package anim

import (
	"math"
	"testing"
	"time"

	"awapp/internal/render"
)

func TestMoonPhaseKnownDates(t *testing.T) {
	// Known new moon: 2000-01-06 18:14 UTC.
	if p := MoonPhaseAt(time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)); p > 0.05 {
		t.Errorf("new moon phase = %v, want near 0", p)
	}
	// Known full moon: 2000-01-21 04:40 UTC (about half a synodic month later).
	if p := MoonPhaseAt(time.Date(2000, 1, 21, 4, 40, 0, 0, time.UTC)); math.Abs(p-0.5) > 0.06 {
		t.Errorf("full moon phase = %v, want near 0.5", p)
	}
}

func TestMoonPhaseRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 90*24; i++ {
		p := MoonPhaseAt(start.Add(time.Duration(i) * time.Hour))
		if p < 0 || p >= 1 {
			t.Fatalf("phase out of range at hour %d: %v", i, p)
		}
	}
}

func TestMoonIllumAndNames(t *testing.T) {
	if a := MoonIllum(0.0); a > 0.02 {
		t.Errorf("new moon illum = %v, want ~0", a)
	}
	if a := MoonIllum(0.5); a < 0.98 {
		t.Errorf("full moon illum = %v, want ~1", a)
	}
	if a := MoonIllum(0.25); a < 0.48 || a > 0.52 {
		t.Errorf("quarter illum = %v, want ~0.5", a)
	}

	cases := map[float64]string{
		0.00: "new moon",
		0.12: "waxing crescent",
		0.25: "first quarter",
		0.40: "waxing gibbous",
		0.50: "full moon",
		0.60: "waning gibbous",
		0.75: "last quarter",
		0.90: "waning crescent",
	}
	for p, want := range cases {
		if got := MoonPhaseName(p); got != want {
			t.Errorf("MoonPhaseName(%v) = %q, want %q", p, got, want)
		}
	}
}

func TestEclipseProgress(t *testing.T) {
	start := time.Unix(1000, 0)
	dur := 2 * time.Minute

	if p, act := EclipseProgress(start, dur, start); p != 0 || !act {
		t.Errorf("at start: p=%v act=%v, want 0,true", p, act)
	}
	if p, act := EclipseProgress(start, dur, start.Add(1*time.Minute)); math.Abs(p-0.5) > 0.01 || !act {
		t.Errorf("midpoint: p=%v act=%v, want ~0.5,true", p, act)
	}
	if p, act := EclipseProgress(start, dur, start.Add(2*time.Minute)); p != 1 || act {
		t.Errorf("at end: p=%v act=%v, want 1,false", p, act)
	}
	if _, act := EclipseProgress(time.Time{}, dur, time.Now()); act {
		t.Error("zero start should be inactive")
	}
	if _, act := EclipseProgress(start, 0, time.Now()); act {
		t.Error("zero duration should be inactive")
	}
}

func TestDrawMoonAllPhases(t *testing.T) {
	buf := render.NewBuffer(40, 20)
	for i := 0; i < 8; i++ {
		p := float64(i) / 8
		buf.Clear(17)
		DrawMoon(buf, 20, 10, 7, 3, p, 0, false)
		if s := buf.Frame(); len(s) == 0 {
			t.Fatalf("empty frame for phase %v", p)
		}
	}
}

func TestDrawMoonTotalityHidden(t *testing.T) {
	// At totality (progress 0.5) the shadow covers the whole disc, so the
	// Moon is fully invisible.
	buf := render.NewBuffer(40, 20)
	DrawMoon(buf, 20, 10, 7, 3, 0.5, 0.5, true)
	if containsBraille(buf.Text()) {
		t.Errorf("totality should hide the moon, got: %q", buf.Text())
	}
}

func TestDrawMoonFullVisible(t *testing.T) {
	// Before/after the eclipse the full moon is fully visible.
	buf := render.NewBuffer(40, 20)
	DrawMoon(buf, 20, 10, 7, 3, 0.5, 0, false)
	if !containsBraille(buf.Text()) {
		t.Error("full moon should be visible (braille texture expected)")
	}
}

// containsBraille reports whether s contains any braille dot-pattern cell.
func containsBraille(s string) bool {
	for _, r := range s {
		if r >= '\u2800' && r <= '\u28ff' {
			return true
		}
	}
	return false
}

// countBraille returns the number of non-blank braille dot-pattern cells.
func countBraille(s string) int {
	n := 0
	for _, r := range s {
		if r > '\u2800' && r <= '\u28ff' {
			n++
		}
	}
	return n
}

func TestMoonVisible(t *testing.T) {
	// Auto mode: new moon hides itself, full moon shows.
	s := NewSun(true, MoonOptions{Auto: true, PhaseOverride: 0.0})
	if s.MoonVisible() {
		t.Error("new moon should auto-hide")
	}
	s.Opts.PhaseOverride = 0.5
	if !s.MoonVisible() {
		t.Error("full moon should be visible")
	}
	// Forced off / on.
	hidden := NewSun(true, MoonOptions{Auto: false, Visible: false, PhaseOverride: 0.5})
	if hidden.MoonVisible() {
		t.Error("forced-off moon should stay hidden")
	}
	shown := NewSun(true, MoonOptions{Auto: false, Visible: true, PhaseOverride: 0.0})
	if !shown.MoonVisible() {
		t.Error("forced-on moon should show even at new moon")
	}
}

func TestDrawMoonCenteredNotClipped(t *testing.T) {
	buf := render.NewBuffer(20, 10)
	DrawMoon(buf, 5, 3, 4, 2, 0.5, 0, false)    // near top-left, must not panic
	DrawMoon(buf, 19, 9, 4, 2, 0.75, 0.5, true) // near bottom-right, must not panic
}
