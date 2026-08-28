package anim

import (
	"testing"
	"time"

	"awapp/internal/render"
)

func TestSeasonForHemispheres(t *testing.T) {
	cases := []struct {
		month time.Month
		lat   float64
		want  Season
	}{
		{time.January, 40, SeasonWinter},
		{time.January, -30, SeasonSummer},
		{time.June, 40, SeasonSummer},
		{time.June, -30, SeasonWinter},
		{time.April, 40, SeasonSpring},
		{time.April, -30, SeasonFall},
		{time.October, 40, SeasonFall},
		{time.October, -30, SeasonSpring},
		{time.August, -3.7, SeasonWinter}, // Fortaleza, southern hemisphere
	}
	for _, c := range cases {
		if got := SeasonFor(c.month, c.lat); got != c.want {
			t.Errorf("SeasonFor(%v, %.1f) = %v, want %v", c.month, c.lat, got, c.want)
		}
	}
}

// The leaf art must survive trimming: non-empty bounding box, and every
// kept row has at least one visible (non-blank) cell.
func TestLeafArtTrimmedNonEmpty(t *testing.T) {
	if len(leafSummer) == 0 {
		t.Fatal("summer leaf should have rows after trim")
	}
	for i, art := range leafFall {
		if len(art) == 0 {
			t.Fatalf("fall leaf %d should have rows after trim", i)
		}
		for _, row := range art {
			visible := false
			for _, r := range row {
				if !leafBlank(r) {
					visible = true
					break
				}
			}
			if !visible {
				t.Fatalf("fall leaf %d: trimmed row should contain visible cells", i)
			}
		}
	}
}

func TestLeavesDrawSeason(t *testing.T) {
	for _, s := range []Season{SeasonSpring, SeasonSummer, SeasonFall, SeasonWinter} {
		l := NewLeaves()
		l.SetSeason(s)
		l.SetWind(5, 270) // from the west -> blowing right
		l.Resize(80, 24)
		for i := 0; i < 10; i++ {
			l.Tick()
		}
		if s == SeasonWinter {
			if len(l.leaves) != 0 {
				t.Error("winter is bare — snow comes from the weather, not the season")
			}
			continue
		}
		if len(l.leaves) == 0 {
			t.Fatalf("%v should have drifting leaves", s)
		}
		buf := render.NewBuffer(80, 24)
		l.Draw(buf)
		if countVisible(buf.Text()) == 0 {
			t.Errorf("%v leaves should render something", s)
		}
	}
}

func countVisible(s string) int {
	n := 0
	for _, r := range s {
		if !leafBlank(r) {
			n++
		}
	}
	return n
}

// Most leaves must be background (small); full-size foreground leaves
// are only a small fraction.
func TestLeavesScaleDepth(t *testing.T) {
	l := NewLeaves()
	l.SetSeason(SeasonFall)
	l.Resize(120, 40)
	if len(l.leaves) == 0 {
		t.Fatal("fall should have leaves")
	}
	small, large := 0, 0
	for _, lf := range l.leaves {
		switch lf.scale {
		case scaleLarge:
			large++
		case scaleSmall:
			small++
		}
	}
	if small == 0 {
		t.Error("expected background (small) leaves")
	}
	if large >= len(l.leaves)/2 {
		t.Errorf("large leaves (%d/%d) should be a small minority", large, len(l.leaves))
	}
}

// The scale picker itself must honour ~5% large / ~70% small over many
// draws (checked statistically, not per-frame, to avoid flakiness).
func TestPickScaleDistribution(t *testing.T) {
	l := NewLeaves()
	const n = 20000
	large, small := 0, 0
	for i := 0; i < n; i++ {
		switch l.pickScale() {
		case scaleLarge:
			large++
		case scaleSmall:
			small++
		}
	}
	if frac := float64(large) / n; frac < 0.02 || frac > 0.10 {
		t.Errorf("large leaves should be ~5%%, got %.1f%%", frac*100)
	}
	if frac := float64(small) / n; frac < 0.60 || frac > 0.80 {
		t.Errorf("small leaves should be ~70%%, got %.1f%%", frac*100)
	}
}

// The 'l' toggle must stop leaves from drawing.
func TestLeavesToggleOffSkipsDraw(t *testing.T) {
	l := NewLeaves()
	l.SetSeason(SeasonSummer)
	l.Resize(80, 24)

	on := render.NewBuffer(80, 24)
	l.Draw(on)
	if countLeafBraille(on.Text()) == 0 {
		t.Fatal("leaves should draw when on")
	}

	l.SetOn(false)
	off := render.NewBuffer(80, 24)
	l.Draw(off)
	if countLeafBraille(off.Text()) != 0 {
		t.Error("leaves should draw nothing when toggled off")
	}
}

func countLeafBraille(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x2800 && r <= 0x28FF {
			n++
		}
	}
	return n
}
