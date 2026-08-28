package anim

import (
	"strings"
	"testing"

	"weatherterm/internal/render"
)

func TestCloudArtSelection(t *testing.T) {
	arts := map[string][]string{
		"storm":   (&Clouds{Storm: true}).pickArt(),
		"stratus": (&Clouds{Dense: true}).pickArt(),
		"cirrus":  (&Clouds{Misty: true}).pickArt(),
		"cumulus": (&Clouds{}).pickArt(),
	}
	seen := map[string]bool{}
	for name, art := range arts {
		if len(art) == 0 {
			t.Fatalf("%s art is empty", name)
		}
		key := strings.Join(art, "\n")
		if seen[key] {
			t.Errorf("%s art collides with another cloud type", name)
		}
		seen[key] = true
	}
}

func TestWindScale(t *testing.T) {
	c := &Clouds{wind: 0}
	if s := c.windScale(); s < 0.9 || s > 1.1 {
		t.Errorf("calm wind scale = %v, want ~1", s)
	}
	c.wind = 10
	if s := c.windScale(); s < 1.9 || s > 2.1 {
		t.Errorf("wind 10 scale = %v, want ~2", s)
	}
	c.wind = 50
	if s := c.windScale(); s != 3 {
		t.Errorf("huge wind scale = %v, want capped at 3", s)
	}
}

func TestDrawCloudsAllKinds(t *testing.T) {
	for _, storm := range []bool{false, true} {
		for _, dense := range []bool{false, true} {
			for _, misty := range []bool{false, true} {
				c := NewClouds(dense, misty, storm, 6.0, 270)
				buf := render.NewBuffer(80, 24)
				c.Resize(80, 24)
				c.Tick()
				c.Draw(buf)
				if f := buf.Frame(); len(f) == 0 {
					t.Fatal("empty frame")
				}
			}
		}
	}
}

func TestDrawCloudBandStorm(t *testing.T) {
	buf := render.NewBuffer(80, 24)
	drawCloudBand(buf, 0, cloudArtStormRight, 80, 0, 1.0, 235)
	if !strings.Contains(buf.Text(), `\`) {
		t.Error("right storm band should render rain curtains leaning right (\\)")
	}
	buf2 := render.NewBuffer(80, 24)
	drawCloudBand(buf2, 0, cloudArtStormLeft, 80, 0, 1.0, 235)
	if !strings.Contains(buf2.Text(), "/") {
		t.Error("left storm band should render rain curtains leaning left (/)")
	}
}

func TestWindEastward(t *testing.T) {
	if windEastward(270) <= 0 {
		t.Error("wind from W (270) should have a positive eastward component")
	}
	if windEastward(90) >= 0 {
		t.Error("wind from E (90) should have a negative eastward component")
	}
}

func TestStormArtByWindDirection(t *testing.T) {
	if got := stormArt(270); got[5] != cloudArtStormRight[5] {
		t.Error("wind from W (270) should pick the right-leaning storm art")
	}
	if got := stormArt(90); got[5] != cloudArtStormLeft[5] {
		t.Error("wind from E (90) should pick the left-leaning storm art")
	}
}

func TestDrawPrecipAllModes(t *testing.T) {
	modes := []struct {
		mode    Mode
		heavy   bool
		thunder bool
	}{
		{ModeRain, false, false},
		{ModeRain, true, false},
		{ModeRain, true, true},
		{ModeSnow, true, false},
	}
	for _, m := range modes {
		p := NewPrecip(m.mode, m.heavy, m.thunder, 8.0, 270)
		buf := render.NewBuffer(80, 24)
		p.Resize(80, 24)
		p.Tick()
		p.Draw(buf)
		if f := buf.Frame(); len(f) == 0 {
			t.Fatal("empty frame")
		}
	}
}
