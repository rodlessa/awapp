package anim

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"awapp/internal/render"
)

// Golden-file tests: render fixed scenes and compare to snapshots under
// testdata/. Regenerate with:  go test ./internal/anim -run TestGolden -update
var update = flag.Bool("update", false, "regenerate golden files")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func goldenPath(name string) string {
	return filepath.Join("testdata", name+".golden")
}

func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := goldenPath(name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden file %s (run with -update to create): %v", path, err)
	}
	if string(want) != got {
		t.Errorf("%s: rendered frame changed (run with -update to accept)", name)
	}
}

func TestGoldenNightSky(t *testing.T) {
	SetClock(func() time.Time {
		return time.Date(2026, 8, 12, 2, 0, 0, 0, time.UTC) // Perseids night, meteor active
	})
	defer SetClock(nil)
	buf := render.NewBuffer(100, 30)
	s := NewSun(true, MoonOptions{PhaseOverride: 0.5, Auto: true})
	s.SetLocation(40, -100)
	s.Resize(100, 30)
	s.SetStarFactor(1.0)
	s.SetSkyline(2)
	s.SetAurora(0.6)
	buf.Clear(17)
	s.Draw(buf)
	assertGolden(t, "night_sky", buf.Text())
}

func TestGoldenDaySun(t *testing.T) {
	SetClock(func() time.Time {
		return time.Date(2026, 8, 12, 15, 0, 0, 0, time.UTC)
	})
	defer SetClock(nil)
	buf := render.NewBuffer(100, 30)
	s := NewSun(false, MoonOptions{})
	s.SetLocation(-3.7, -38.5)
	s.SetTimes(SkyTimes{
		Sunrise:  time.Date(2026, 8, 12, 8, 34, 0, 0, time.UTC).Unix(),
		Sunset:   time.Date(2026, 8, 12, 20, 36, 0, 0, time.UTC).Unix(),
		Timezone: -10800,
	})
	s.Resize(100, 30)
	buf.Clear(17)
	s.Draw(buf)
	assertGolden(t, "day_sun", buf.Text())
}

func TestGoldenRain(t *testing.T) {
	buf := render.NewBuffer(100, 30)
	p := NewPrecip(ModeRain, true, false, 5, 90)
	p.Resize(100, 30)
	for i := 0; i < 20; i++ {
		p.Tick()
	}
	buf.Clear(17)
	p.Draw(buf)
	assertGolden(t, "rain", buf.Text())
}

func TestGoldenMist(t *testing.T) {
	buf := render.NewBuffer(100, 30)
	c := NewClouds(true, true, false, 2, 0)
	c.Resize(100, 30)
	for i := 0; i < 10; i++ {
		c.Tick()
	}
	buf.Clear(17)
	c.Draw(buf)
	assertGolden(t, "mist", buf.Text())
}
