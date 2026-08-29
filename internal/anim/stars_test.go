package anim

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"awapp/internal/render"
)

func TestGMSTAtJ2000(t *testing.T) {
	// At J2000.0 (JD 2451545.0) GMST is 18h 41m 50.5s = 18.6974 h.
	g := gmstHours(2451545.0)
	if math.Abs(g-18.697374558) > 0.01 {
		t.Errorf("GMST at J2000 = %.6f h, want ~18.6974", g)
	}
	// A sidereal day is ~23h56m: GMST advances ~24.0657 h per calendar day.
	diff := math.Mod(gmstHours(2451546.0)-gmstHours(2451545.0), 24)
	if diff < 0 {
		diff += 24
	}
	if math.Abs(diff-0.06571) > 0.001 {
		t.Errorf("GMST drift per day = %.5f h, want ~0.06571", diff)
	}
}

func TestAltAzPoles(t *testing.T) {
	jd := julianDay(time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC))
	// Polaris (dec ~89.26) at the north pole is always ~89.26° up.
	alt, _ := altAz(90, 0, 2.5317, 89.264, jd)
	if math.Abs(alt-89.264) > 0.5 {
		t.Errorf("Polaris at N pole: alt=%.2f, want ~89.26", alt)
	}
	// A dec -52° star at the south pole sits 52° above the horizon.
	alt, _ = altAz(-90, 0, 0, -52, jd)
	if math.Abs(alt-52) > 0.5 {
		t.Errorf("dec -52 at S pole: alt=%.2f, want ~52", alt)
	}
	// At the north pole a star with dec < 0 is always below the horizon.
	alt, _ = altAz(90, 0, 0, -30, jd)
	if alt > 0 {
		t.Errorf("dec -30 at N pole should be below the horizon, got alt=%.2f", alt)
	}
}

func TestAltAzMeridianTransit(t *testing.T) {
	// At 2000-01-01 12:00 UTC (JD 2451545.0) at Greenwich, a star whose RA
	// equals the local sidereal time is on the meridian; a dec-10 star
	// transits at alt = 90 - (lat - dec) = 90 - (51.4778 - 10) ≈ 48.52.
	jd := julianDay(time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC))
	lst := localSiderealHours(jd, 0)
	alt, _ := altAz(51.4778, 0, lst, 10, jd)
	if math.Abs(alt-48.52) > 0.5 {
		t.Errorf("meridian transit: alt=%.2f, want ~48.52", alt)
	}
}

func TestStarsVisibilityScalesWithPollution(t *testing.T) {
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)
	f := NewStarField()
	f.SetLocation(40, -100)
	f.Resize(120, 30)
	buf := render.NewBuffer(120, 30)

	buf.Clear(17)
	f.Draw(buf, 0.0, now)
	city := countStars(buf)

	buf.Clear(17)
	f.Draw(buf, 1.0, now)
	dark := countStars(buf)

	if city == 0 {
		t.Fatal("expected a few bright stars even in a light-polluted city")
	}
	if dark <= city {
		t.Errorf("dark sky should show more stars than a city: city=%d dark=%d", city, dark)
	}
}

func TestStarsRequireLocation(t *testing.T) {
	f := NewStarField() // location never set
	f.Resize(80, 24)
	buf := render.NewBuffer(80, 24)
	buf.Clear(17)
	f.Draw(buf, 1.0, time.Now())
	if countStars(buf) != 0 {
		t.Error("no stars should be drawn without a known location")
	}
	f.SetLocation(40, -100)
	buf.Clear(17)
	f.Draw(buf, 1.0, time.Now())
	if countStars(buf) == 0 {
		t.Error("stars should appear once the location is set")
	}
}

func TestStarsCanBeHidden(t *testing.T) {
	f := NewStarField()
	f.SetLocation(40, -100)
	f.Resize(80, 24)
	buf := render.NewBuffer(80, 24)
	now := time.Date(2026, 8, 28, 2, 0, 0, 0, time.UTC)

	f.SetVisible(false)
	buf.Clear(17)
	f.Draw(buf, 1.0, now)
	if countStars(buf) != 0 {
		t.Error("stars should be hidden when visibility is off")
	}

	f.SetVisible(true)
	buf.Clear(17)
	f.Draw(buf, 1.0, now)
	if countStars(buf) == 0 {
		t.Error("stars should reappear when visibility is on")
	}
}

func TestStarsMoveAcrossTheNight(t *testing.T) {
	// The sky must not be frozen: six hours apart, the sidereal clock has
	// rotated the sky, so the star pattern lands in different columns.
	f := NewStarField()
	f.SetLocation(40, -100)
	f.Resize(120, 30)
	buf := render.NewBuffer(120, 30)
	t1 := time.Date(2026, 8, 28, 22, 0, 0, 0, time.UTC)
	t2 := t1.Add(6 * time.Hour)

	buf.Clear(17)
	f.Draw(buf, 1.0, t1)
	first := columnCounts(buf)

	buf.Clear(17)
	f.Draw(buf, 1.0, t2)
	second := columnCounts(buf)

	if reflect.DeepEqual(first, second) {
		t.Error("expected the star pattern to change over 6h")
	}
}

// columnCounts tallies how many stars land in each screen column.
func columnCounts(b *render.Buffer) map[int]int {
	cols := map[int]int{}
	for y, row := range strings.Split(b.Text(), "\n") {
		if y >= b.H {
			break
		}
		for x, r := range row {
			if r != ' ' {
				cols[x]++
			}
		}
	}
	return cols
}

func countStars(b *render.Buffer) int {
	s := b.Text()
	return strings.Count(s, "*") + strings.Count(s, ".")
}
