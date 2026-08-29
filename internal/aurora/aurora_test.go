package aurora

import "testing"

func TestLikelihood(t *testing.T) {
	// Equator: never.
	if got := Likelihood(0, 9); got != 0 {
		t.Errorf("Likelihood(0, 9) = %v, want 0", got)
	}
	// High latitude without geomagnetic activity: no aurora.
	if got := Likelihood(60, 0); got != 0 {
		t.Errorf("Likelihood(60, 0) = %v, want 0", got)
	}
	// High latitude with a storm: aurora likely.
	if got := Likelihood(60, 7); got <= 0 {
		t.Errorf("Likelihood(60, 7) = %v, want > 0", got)
	}
	// Very high latitude always has some chance during minor activity.
	if got := Likelihood(80, 2); got <= 0 {
		t.Errorf("Likelihood(80, 2) = %v, want > 0", got)
	}
	// Symmetric for the southern hemisphere.
	if got := Likelihood(-60, 7); got <= 0 {
		t.Errorf("Likelihood(-60, 7) = %v, want > 0", got)
	}
	// Intensity grows with Kp at the same latitude.
	low := Likelihood(60, 3)
	high := Likelihood(60, 8)
	if high <= low {
		t.Errorf("higher Kp should give a stronger likelihood: Kp3=%v Kp8=%v", low, high)
	}
}

func TestParseKpForecast(t *testing.T) {
	body := []byte(`[
		{"predicted_kp_index": 2.0},
		{"predicted_kp_index": 5.0},
		{"predicted_kp_index": 3.3}
	]`)
	kp, err := parseKpForecast(body)
	if err != nil {
		t.Fatal(err)
	}
	if kp != 5.0 {
		t.Errorf("max predicted Kp = %v, want 5.0", kp)
	}
}
