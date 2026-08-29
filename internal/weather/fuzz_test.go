package weather

import (
	"encoding/json"
	"testing"
)

func FuzzParseWeatherAPI(f *testing.F) {
	f.Add([]byte(`{"location":{"name":"x","lat":1,"lon":2},"current":{"temp_c":20,"humidity":50,"wind_kph":10,"wind_degree":90,"condition":{"text":"t","code":1000}}}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseWeatherAPI(data, "fallback")
	})
}

func FuzzParseTomorrow(f *testing.F) {
	f.Add([]byte(`{"data":{"timelines":[{"intervals":[{"startTime":"2026-01-01T00:00:00Z","values":{"weatherCode":"clear","temperature":20,"humidity":50,"windSpeed":5,"windDirection":90}}]}]}}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = parseTomorrow(data, &geoInfo{City: "x"})
	})
}

func FuzzParseMeteoForecast(f *testing.F) {
	f.Add([]byte(`{"hourly":{"time":["2026-08-28T20:00"],"weather_code":[0],"temperature_2m":[20]},"daily":{"time":["2026-08-28"],"weather_code":[0],"temperature_2m_max":[30],"temperature_2m_min":[20]}}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var m meteoResponse
		if err := json.Unmarshal(data, &m); err != nil {
			return
		}
		parseMeteoHourly(m.Hourly.Time, m.Hourly.Code, m.Hourly.Temperature)
		parseMeteoDaily(m.Daily.Time, m.Daily.Code, m.Daily.TempMax, m.Daily.TempMin)
	})
}
