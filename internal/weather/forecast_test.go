package weather

import (
	"testing"
	"time"
)

func TestParseMeteoHourly(t *testing.T) {
	now := time.Now().UTC()
	times := make([]string, 0, 14)
	codes := make([]int, 0, 14)
	temps := make([]float64, 0, 14)
	// One already-passed hour, then 13 upcoming hours.
	times = append(times, now.Add(-time.Hour).Format("2006-01-02T15:04"))
	codes = append(codes, 0)
	temps = append(temps, 20)
	for i := 0; i < 13; i++ {
		times = append(times, now.Add(time.Duration(i+1)*time.Hour).Format("2006-01-02T15:04"))
		codes = append(codes, 3) // overcast
		temps = append(temps, 25)
	}
	out := parseMeteoHourly(times, codes, temps)
	if len(out) != 12 {
		t.Fatalf("expected 12 points (past hour skipped, list capped), got %d", len(out))
	}
	if out[0].Condition != Clouds || out[0].TempC != 25 {
		t.Errorf("first point should be the upcoming overcast hour, got %+v", out[0])
	}
	if out[0].When.Before(now) {
		t.Error("the already-passed hour should have been skipped")
	}
}

func TestParseMeteoHourlyEmpty(t *testing.T) {
	if got := parseMeteoHourly(nil, nil, nil); len(got) != 0 {
		t.Errorf("expected no points for empty input, got %d", len(got))
	}
}

func TestParseMeteoDaily(t *testing.T) {
	times := []string{"2026-08-28", "2026-08-29", "2026-08-30"}
	codes := []int{2, 61, 3}
	max := []float64{31, 30, 29}
	min := []float64{25, 24, 23}
	out := parseMeteoDaily(times, codes, max, min)
	if len(out) != 3 {
		t.Fatalf("expected 3 days, got %d", len(out))
	}
	if out[0].Day.Format("2006-01-02") != "2026-08-28" || out[0].Condition != Clouds {
		t.Errorf("first day wrong: %+v", out[0])
	}
	if out[1].Condition != Rain || out[1].HighC != 30 || out[1].LowC != 24 {
		t.Errorf("second day should be rain 30/24, got %+v", out[1])
	}
}
