package air

import "testing"

func TestLabelFor(t *testing.T) {
	cases := map[int]string{
		0: "Good", 50: "Good", 51: "Moderate", 100: "Moderate",
		101: "Unhealthy for sensitive groups", 150: "Unhealthy for sensitive groups",
		151: "Unhealthy", 200: "Unhealthy", 201: "Very unhealthy",
		300: "Very unhealthy", 301: "Hazardous", 500: "Hazardous",
	}
	for aqi, want := range cases {
		if got := labelFor(aqi); got != want {
			t.Errorf("labelFor(%d) = %q, want %q", aqi, got, want)
		}
	}
}
