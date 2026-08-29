package weather

import (
	"testing"
)

const weatherAPISample = `{
  "location": {"name": "Fortaleza", "lat": -3.72, "lon": -38.54, "localtime": "2026-08-28 20:00"},
  "current": {
    "temp_c": 27.0, "humidity": 78, "wind_kph": 18, "wind_degree": 90,
    "condition": {"text": "Partly cloudy", "code": 1003}
  },
  "forecast": {
    "forecastday": [
      {"hour": [
        {"time": "2026-08-28 21:00", "temp_c": 26.5, "condition": {"code": 1006}},
        {"time": "2026-08-28 22:00", "temp_c": 26.0, "condition": {"code": 1183}}
      ]}
    ]
  }
}`

func TestParseWeatherAPI(t *testing.T) {
	r, err := parseWeatherAPI([]byte(weatherAPISample), "Fortaleza")
	if err != nil {
		t.Fatal(err)
	}
	if r.City != "Fortaleza" {
		t.Errorf("city = %q, want Fortaleza", r.City)
	}
	if r.Condition != Clouds {
		t.Errorf("condition = %v, want Clouds (code 1003)", r.Condition)
	}
	if r.Desc != "Partly cloudy" {
		t.Errorf("desc = %q", r.Desc)
	}
	if got := r.TempKelvin - 273.15; got < 26.9 || got > 27.1 {
		t.Errorf("temp C = %.2f, want ~27", got)
	}
	if r.WindMS != 5.0 {
		t.Errorf("wind = %.2f m/s, want 5.0 (18 kph)", r.WindMS)
	}
	if len(r.Hourly) != 2 {
		t.Fatalf("hourly len = %d, want 2", len(r.Hourly))
	}
	if r.Hourly[0].Condition != Clouds || r.Hourly[1].Condition != Rain {
		t.Errorf("hourly conditions wrong: %v, %v", r.Hourly[0].Condition, r.Hourly[1].Condition)
	}
}

func TestWeatherAPICondition(t *testing.T) {
	cases := map[int]Condition{
		1000: Clear, 1003: Clouds, 1009: Clouds, 1030: Mist, 1063: Rain,
		1087: Thunderstorm, 1114: Snow, 1135: Mist, 1183: Rain, 1273: Thunderstorm,
	}
	for code, want := range cases {
		if got := weatherAPICondition(code); got != want {
			t.Errorf("weatherAPICondition(%d) = %v, want %v", code, got, want)
		}
	}
}

const tomorrowSample = `{
  "data": {
    "timelines": [
      {
        "intervals": [
          {"startTime": "2026-08-28T20:00:00Z", "values": {"weatherCode": "clear", "temperature": 27, "humidity": 75, "windSpeed": 5, "windDirection": 90}},
          {"startTime": "2026-08-28T21:00:00Z", "values": {"weatherCode": "cloudly", "temperature": 26.5, "humidity": 76, "windSpeed": 5, "windDirection": 90}},
          {"startTime": "2026-08-28T22:00:00Z", "values": {"weatherCode": "heavySnow", "temperature": 26.0, "humidity": 77, "windSpeed": 5, "windDirection": 90}}
        ]
      }
    ]
  }
}`

func TestParseTomorrow(t *testing.T) {
	g := &geoInfo{City: "Fortaleza", Lat: -3.71722, Lon: -38.54306}
	r, err := parseTomorrow([]byte(tomorrowSample), g)
	if err != nil {
		t.Fatal(err)
	}
	if r.Condition != Clear {
		t.Errorf("condition = %v, want Clear", r.Condition)
	}
	if r.City != "Fortaleza" || r.Lat != -3.71722 {
		t.Errorf("location not carried through: %+v", r)
	}
	if got := r.TempKelvin - 273.15; got != 27 {
		t.Errorf("temp C = %.2f, want 27", got)
	}
	if len(r.Hourly) != 2 {
		t.Fatalf("hourly len = %d, want 2 (current skipped)", len(r.Hourly))
	}
	if r.Hourly[0].Condition != Clouds || r.Hourly[1].Condition != Snow {
		t.Errorf("hourly conditions wrong: %v, %v", r.Hourly[0].Condition, r.Hourly[1].Condition)
	}
}

func TestTomorrowCondition(t *testing.T) {
	cases := map[string]Condition{
		"clear": Clear, "mostlyClear": Clear,
		"partlyCloudy": Clouds, "cloudly": Clouds, "overcast": Clouds,
		"fog": Mist, "haze": Mist,
		"drizzle": Rain, "heavyRain": Rain, "freezingRain": Rain, "sleet": Rain,
		"snow": Snow, "heavySnow": Snow, "iceSnow": Snow,
		"thunderstorm": Thunderstorm,
		"bogus":        Clouds,
	}
	for code, want := range cases {
		if got := tomorrowCondition(code); got != want {
			t.Errorf("tomorrowCondition(%q) = %v, want %v", code, got, want)
		}
	}
}
