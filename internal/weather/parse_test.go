package weather

import (
	"testing"
	"time"
)

func TestParseLocalTime(t *testing.T) {
	// "2026-08-28T05:34" is the LOCAL wall clock (Fortaleza, UTC-3). Its UTC
	// instant is 08:34 UTC.
	want := time.Date(2026, 8, 28, 8, 34, 0, 0, time.UTC).Unix()
	if got := parseLocalTime("2026-08-28T05:34", -10800); got != want {
		t.Errorf("parseLocalTime = %v, want %v", got, want)
	}
	// A UTC+2 city: local 05:34 -> 03:34 UTC.
	want2 := time.Date(2026, 8, 28, 3, 34, 0, 0, time.UTC).Unix()
	if got := parseLocalTime("2026-08-28T05:34", 7200); got != want2 {
		t.Errorf("parseLocalTime(+2) = %v, want %v", got, want2)
	}
	if parseLocalTime("", 0) != 0 {
		t.Error("empty string should parse to 0")
	}
}
