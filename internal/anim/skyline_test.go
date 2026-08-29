package anim

import (
	"testing"
	"time"
)

func TestMeteorActive(t *testing.T) {
	// Perseids peak ~Aug 12.
	if !meteorActive(time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)) {
		t.Error("expected meteorActive on Aug 12 (Perseids)")
	}
	// Geminids peak ~Dec 14.
	if !meteorActive(time.Date(2026, 12, 14, 0, 0, 0, 0, time.UTC)) {
		t.Error("expected meteorActive on Dec 14 (Geminids)")
	}
	// Off-season.
	if meteorActive(time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)) {
		t.Error("no major shower expected mid-June")
	}
}

func TestStarFieldSkylineLevels(t *testing.T) {
	f := NewStarField()
	f.SetLocation(40, -100)
	f.Resize(80, 24)
	f.SetSkyline(3)
	if f.skyline != 3 {
		t.Errorf("skyline = %d, want 3", f.skyline)
	}
	f.SetSkyline(99)
	if f.skyline != 3 {
		t.Errorf("skyline should clamp to 3, got %d", f.skyline)
	}
	f.SetSkyline(-1)
	if f.skyline != 0 {
		t.Errorf("skyline should clamp to 0, got %d", f.skyline)
	}
}
