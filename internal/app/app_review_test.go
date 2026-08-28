package app

import (
	"testing"
	"time"

	"awapp/internal/anim"
	"awapp/internal/weather"
)

// The -season override must win over the date/latitude computation.
func TestResolveSeasonOverride(t *testing.T) {
	cases := []struct {
		override string
		want     anim.Season
	}{
		{"spring", anim.SeasonSpring},
		{"summer", anim.SeasonSummer},
		{"fall", anim.SeasonFall},
		{"autumn", anim.SeasonFall}, // alias
		{"winter", anim.SeasonWinter},
	}
	for _, c := range cases {
		a := &App{seasonOverride: c.override}
		if got := a.resolveSeason(0); got != c.want {
			t.Errorf("resolveSeason(%q) = %s, want %s", c.override, got, c.want)
		}
	}
}

// Season math: months are flipped in the southern hemisphere.
func TestSeasonForHemisphere(t *testing.T) {
	if got := anim.SeasonFor(time.January, 40); got != anim.SeasonWinter {
		t.Errorf("northern January should be winter, got %s", got)
	}
	if got := anim.SeasonFor(time.January, -23); got != anim.SeasonSummer {
		t.Errorf("southern January should be summer (Fortaleza!), got %s", got)
	}
	if got := anim.SeasonFor(time.July, -23); got != anim.SeasonWinter {
		t.Errorf("southern July should be winter, got %s", got)
	}
	if got := anim.SeasonFor(time.October, -23); got != anim.SeasonSpring {
		t.Errorf("southern October should be spring, got %s", got)
	}
}

// animatorFor must pick the right animator family per condition.
func TestAnimatorForConditionModes(t *testing.T) {
	if an := animatorFor(weather.Report{Condition: weather.Clear}, anim.MoonOptions{}, true); an == nil {
		t.Fatal("clear should produce an animator")
	}
	if _, ok := animatorFor(weather.Report{Condition: weather.Clear}, anim.MoonOptions{}, true).(*anim.Sun); !ok {
		t.Error("clear should produce a Sun animator")
	}
	if _, ok := animatorFor(weather.Report{Condition: weather.Clouds}, anim.MoonOptions{}, true).(*anim.Clouds); !ok {
		t.Error("clouds should produce a Clouds animator")
	}
	if _, ok := animatorFor(weather.Report{Condition: weather.Mist}, anim.MoonOptions{}, true).(*anim.Clouds); !ok {
		t.Error("mist should produce a Clouds animator")
	}
	snow := animatorFor(weather.Report{Condition: weather.Snow}, anim.MoonOptions{}, true)
	if p, ok := snow.(*anim.Precip); !ok || p.Mode != anim.ModeSnow {
		t.Errorf("snow should produce a snow Precip, got %T mode=%v", snow, modeOf(snow))
	}
	storm := animatorFor(weather.Report{Condition: weather.Thunderstorm}, anim.MoonOptions{}, true)
	if p, ok := storm.(*anim.Precip); !ok || !p.Thunder {
		t.Errorf("thunderstorm should produce a thunder Precip, got %T", storm)
	}
}

func modeOf(an animatorIface) anim.Mode {
	if p, ok := an.(*anim.Precip); ok {
		return p.Mode
	}
	return -1
}

func TestSecsOfDay(t *testing.T) {
	if got := secsOfDay(time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)); got != 0 {
		t.Errorf("midnight = %d, want 0", got)
	}
	if got := secsOfDay(time.Date(2026, 8, 28, 12, 30, 45, 0, time.UTC)); got != 12*3600+30*60+45 {
		t.Errorf("12:30:45 = %d, want %d", got, 12*3600+30*60+45)
	}
	if got := secsOfDay(time.Date(2026, 8, 28, 23, 59, 59, 0, time.UTC)); got != 86399 {
		t.Errorf("23:59:59 = %d, want 86399", got)
	}
}
