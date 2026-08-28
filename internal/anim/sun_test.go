package anim

import (
	"strings"
	"testing"
	"time"

	"weatherterm/internal/render"
)

func TestSunMoonProgress(t *testing.T) {
	// Fortaleza: sunrise 08:34 UTC, sunset 20:36 UTC, tz -10800.
	sunrise := time.Date(2026, 8, 28, 8, 34, 0, 0, time.UTC).Unix()
	sunset := time.Date(2026, 8, 28, 20, 36, 0, 0, time.UTC).Unix()
	s := NewSun(false, MoonOptions{})
	s.SetTimes(SkyTimes{Sunrise: sunrise, Sunset: sunset, Timezone: -10800})

	if p, ok := s.sunProgress(time.Date(2026, 8, 28, 8, 34, 0, 0, time.UTC)); !ok || p > 0.01 {
		t.Errorf("at sunrise: p=%v ok=%v", p, ok)
	}
	if p, ok := s.sunProgress(time.Date(2026, 8, 28, 14, 35, 0, 0, time.UTC)); !ok || p < 0.49 || p > 0.51 {
		t.Errorf("at solar noon: p=%v ok=%v", p, ok)
	}
	if _, ok := s.sunProgress(time.Date(2026, 8, 28, 21, 0, 0, 0, time.UTC)); ok {
		t.Error("sun should be down after sunset")
	}
	// Moon is up during the night, peaking near local midnight (03:00 UTC
	// next day for Fortaleza = 00:00 local).
	if p, ok := s.moonProgress(time.Date(2026, 8, 29, 3, 0, 0, 0, time.UTC)); !ok || p < 0.45 || p > 0.55 {
		t.Errorf("moon mid-night: p=%v ok=%v", p, ok)
	}
	if _, ok := s.moonProgress(time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)); ok {
		t.Error("moon should be down during the day")
	}

	// Clock fallback (no SkyTimes, e.g. offline manual mode): the night is
	// 18:00-06:00 local. Early morning must wrap to the far side of the
	// night, not go negative (that would push the Moon off-screen).
	s2 := NewSun(true, MoonOptions{})
	if p, ok := s2.moonProgress(time.Date(2026, 8, 28, 2, 48, 0, 0, time.Local)); !ok || p < 0.6 || p > 0.8 {
		t.Errorf("fallback 02:48 should be ~0.73 into the night, got p=%v ok=%v", p, ok)
	}
	if p, ok := s2.moonProgress(time.Date(2026, 8, 28, 21, 0, 0, 0, time.Local)); !ok || p < 0.2 || p > 0.3 {
		t.Errorf("fallback 21:00 should be ~0.25 into the night, got p=%v ok=%v", p, ok)
	}
	if _, ok := s2.moonProgress(time.Date(2026, 8, 28, 12, 0, 0, 0, time.Local)); ok {
		t.Error("fallback moon should be down at noon")
	}
}

func TestSkyArcRisesThenSets(t *testing.T) {
	s := NewSun(false, MoonOptions{})
	s.Resize(100, 24)
	x0, y0 := s.skyArc(0.0)
	xm, ym := s.skyArc(0.5)
	x1, y1 := s.skyArc(1.0)
	if !(x0 < xm && xm < x1) {
		t.Errorf("x should rise left->right: %d, %d, %d", x0, xm, x1)
	}
	if !(ym < y0 && ym < y1) {
		t.Errorf("arc should peak at the top in the middle: y0=%d ym=%d y1=%d", y0, ym, y1)
	}
	if y0 <= 0 || y0 > s.h-3 {
		t.Errorf("sunrise should be near the bottom, got y=%d (h=%d)", y0, s.h)
	}
}

func TestSkyArcStaysBelowPanel(t *testing.T) {
	s := NewSun(false, MoonOptions{})
	s.Resize(80, 24)
	s.SetTopMargin(10) // panel occupies rows ~0..9
	_, ym := s.skyArc(0.5)
	_, rh := s.discRadius()
	if top := ym - rh; top < 10 {
		t.Errorf("at the peak the disc top should be at/below the margin: disc top=%d (center %d, rh %d), want >=10", top, ym, rh)
	}
	if _, y0 := s.skyArc(0.0); y0 <= 10 {
		t.Errorf("sunrise should be near the bottom, got y=%d with h=%d", y0, s.h)
	}
}

func TestDiscRadiusDefaultAndClamp(t *testing.T) {
	s := NewSun(false, MoonOptions{})
	s.Resize(80, 24)
	rw, rh := s.discRadius()
	if rw != 6 || rh != 3 {
		t.Errorf("default disc = %dx%d (half %d/%d), want 12x6 at 15%% of 80", rw*2, rh*2, rw, rh)
	}
	s.SetSize(30)
	rw, _ = s.discRadius()
	if rw != 12 {
		t.Errorf("rw at 30%% = %d, want 12", rw)
	}
	s.SetSize(999)
	if s.SizePct() != 60 {
		t.Errorf("size should clamp to 60, got %v", s.SizePct())
	}
	s.SetSize(-5)
	if s.SizePct() != 4 {
		t.Errorf("size should clamp to 4, got %v", s.SizePct())
	}
}

func TestSetStarFactorScalesStars(t *testing.T) {
	s := NewSun(true, MoonOptions{})
	s.Resize(80, 24)
	s.SetStarFactor(1.0)
	full := len(s.sparkles)
	s.SetStarFactor(0.0)
	none := len(s.sparkles)
	if full == 0 {
		t.Fatal("expected stars at full factor")
	}
	if none != 0 {
		t.Errorf("expected 0 stars at factor 0, got %d", none)
	}
}

func TestSolarProgress(t *testing.T) {
	dur := 2 * time.Minute
	s := NewSun(false, MoonOptions{})
	s.SetSolar(true, time.Now(), dur)
	if p, act := s.Solar(); !act || p > 0.05 {
		t.Errorf("at start: p=%v act=%v, want ~0,true", p, act)
	}
	s.SetSolar(true, time.Now().Add(-dur), dur)
	if _, act := s.Solar(); act {
		t.Error("eclipse should be over once the duration has elapsed")
	}
}

func TestDrawSolarEclipseTotality(t *testing.T) {
	buf := render.NewBuffer(80, 24)
	dur := 2 * time.Minute
	s := NewSun(false, MoonOptions{}) // day scene
	s.Resize(80, 24)
	s.SetSolar(true, time.Now().Add(-dur/2), dur) // at totality
	s.Draw(buf)
	if f := buf.Frame(); len(f) == 0 {
		t.Fatal("empty frame")
	}
	// At totality the Sun disc is fully covered -> no braille remains, and
	// a corona ring of '*' shows around it.
	if containsBraille(buf.Text()) {
		t.Error("sun should be fully hidden at totality")
	}
	if !strings.Contains(buf.Text(), "*") {
		t.Error("expected corona '*' at totality")
	}
}

func TestDrawSolarEclipsePartial(t *testing.T) {
	// A partial eclipse still shows part of the sun, but fewer cells than
	// a full disc (the Moon's silhouette has bitten into it).
	full := render.NewBuffer(80, 24)
	sf := NewSun(false, MoonOptions{})
	sf.Resize(80, 24)
	sf.Draw(full)
	fullCount := countBraille(full.Text())

	part := render.NewBuffer(80, 24)
	sp := NewSun(false, MoonOptions{})
	sp.Resize(80, 24)
	sp.SetSolar(true, time.Now().Add(-30*time.Second), 2*time.Minute) // partial
	sp.Draw(part)
	partCount := countBraille(part.Text())

	if partCount == 0 {
		t.Error("part of the sun should still be visible during a partial eclipse")
	}
	if partCount >= fullCount {
		t.Errorf("partial eclipse should block some sun cells: full=%d partial=%d", fullCount, partCount)
	}
}

func TestDrawNightWithLightPollution(t *testing.T) {
	buf := render.NewBuffer(80, 24)
	s := NewSun(true, MoonOptions{PhaseOverride: 0.5})
	s.Resize(80, 24)
	s.SetStarFactor(0.1) // heavy light pollution -> few stars
	s.Draw(buf)
	// Even with heavy pollution a few bright stars remain.
	if !strings.ContainsAny(buf.Text(), "*.") {
		t.Error("expected a few visible stars under heavy light pollution")
	}
}
