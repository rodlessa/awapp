package app

import (
	"testing"
	"time"

	"awapp/internal/weather"
)

// Regression test: Fortaleza on 2026-08-28 — sunrise 05:34 local, sunset
// 17:36 local, UTC-3. At 06:38 local (just after sunrise) it must be DAY.
func TestIsNightFortaleza(t *testing.T) {
	sunrise := time.Date(2026, 8, 28, 8, 34, 0, 0, time.UTC).Unix() // 05:34 local
	sunset := time.Date(2026, 8, 28, 20, 36, 0, 0, time.UTC).Unix() // 17:36 local
	r := weather.Report{Sunrise: sunrise, Sunset: sunset, Timezone: -10800}

	r.FetchedAt = time.Date(2026, 8, 28, 9, 38, 0, 0, time.UTC) // 06:38 local
	if isNight(r) {
		t.Fatal("expected DAY at 06:38 local (sunrise was 05:34)")
	}

	// Exact values observed from a live Open-Meteo fetch.
	r.FetchedAt = time.Date(2026, 8, 28, 9, 40, 26, 0, time.UTC) // 06:40 local
	if isNight(r) {
		t.Fatalf("expected DAY at 06:40 local; sunrise=%d sunset=%d tz=%d",
			r.Sunrise, r.Sunset, r.Timezone)
	}

	// Regression: time.Now() is a local-zoned time; the absolute instant is
	// what matters, so a -03-zoned 06:40 local must still be day.
	r.FetchedAt = time.Date(2026, 8, 28, 6, 40, 26, 0, time.FixedZone("BRT", -3*3600))
	if isNight(r) {
		t.Fatal("expected DAY for a -03-zoned 06:40 FetchedAt (local-zone add bug)")
	}

	r.FetchedAt = time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC) // 09:00 local
	if isNight(r) {
		t.Fatal("expected DAY at 09:00 local")
	}

	r.FetchedAt = time.Date(2026, 8, 28, 21, 0, 0, 0, time.UTC) // 18:00 local
	if !isNight(r) {
		t.Fatal("expected NIGHT at 18:00 local (sunset was 17:36)")
	}
}

// The clock fallback must also treat early morning as day (h=6).
func TestIsNightClockFallbackMorning(t *testing.T) {
	r := weather.Report{} // no sunrise/sunset -> clock fallback
	if isNight(r) && time.Now().Hour() >= 6 && time.Now().Hour() < 18 {
		t.Fatal("clock fallback wrongly said night during the day")
	}
}
