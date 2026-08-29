package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// WeatherAPI (https://www.weatherapi.com) — free tier, needs a key.
// ---------------------------------------------------------------------------

// WeatherAPIClient talks to WeatherAPI's forecast endpoint (current +
// today's hourly strip). The query accepts a city name, "lat,lon",
// postcode, or US zip code.
type WeatherAPIClient struct {
	APIKey string
	Query  string
	HTTP   *http.Client
}

func NewWeatherAPIClient(apiKey, query string) *WeatherAPIClient {
	return &WeatherAPIClient{
		APIKey: apiKey,
		Query:  query,
		HTTP:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *WeatherAPIClient) Fetch(ctx context.Context) (Report, error) {
	if c.APIKey == "" {
		return Report{}, fmt.Errorf("weatherapi: no API key configured")
	}
	if c.Query == "" {
		return Report{}, fmt.Errorf("weatherapi: no location configured")
	}
	u := "https://api.weatherapi.com/v1/forecast.json?" + url.Values{
		"key":  {c.APIKey},
		"q":    {c.Query},
		"days": {"1"},
		"aqi":  {"no"},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Report{}, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Report{}, fmt.Errorf("weatherapi: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Report{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Report{}, fmt.Errorf("weatherapi: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return parseWeatherAPI(body, c.Query)
}

// weatherAPIResponse mirrors the subset of the forecast.json payload we
// use.
type weatherAPIResponse struct {
	Location struct {
		Name      string  `json:"name"`
		Lat       float64 `json:"lat"`
		Lon       float64 `json:"lon"`
		LocalTime string  `json:"localtime"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		Humidity  int     `json:"humidity"`
		WindKph   float64 `json:"wind_kph"`
		WindDeg   float64 `json:"wind_degree"`
		Condition struct {
			Text string `json:"text"`
			Code int    `json:"code"`
		} `json:"condition"`
	} `json:"current"`
	Forecast struct {
		Forecastday []struct {
			Hour []struct {
				Time      string  `json:"time"`
				TempC     float64 `json:"temp_c"`
				Condition struct {
					Code int `json:"code"`
				} `json:"condition"`
			} `json:"hour"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

func parseWeatherAPI(body []byte, fallbackCity string) (Report, error) {
	var w weatherAPIResponse
	if err := json.Unmarshal(body, &w); err != nil {
		return Report{}, fmt.Errorf("weatherapi decode: %w", err)
	}
	r := Report{
		City:       w.Location.Name,
		Condition:  weatherAPICondition(w.Current.Condition.Code),
		Desc:       w.Current.Condition.Text,
		TempKelvin: w.Current.TempC + 273.15,
		Humidity:   w.Current.Humidity,
		WindMS:     w.Current.WindKph / 3.6,
		WindDir:    w.Current.WindDeg,
		Lat:        w.Location.Lat,
		Lon:        w.Location.Lon,
		FetchedAt:  time.Now(),
	}
	if r.City == "" {
		r.City = fallbackCity
	}
	if len(w.Forecast.Forecastday) > 0 {
		for _, h := range w.Forecast.Forecastday[0].Hour {
			if t, err := time.Parse("2006-01-02 15:04", h.Time); err == nil {
				r.Hourly = append(r.Hourly, HourPoint{
					When:      t,
					Condition: weatherAPICondition(h.Condition.Code),
					TempC:     h.TempC,
				})
			}
		}
	}
	// WeatherAPI doesn't expose unix sunrise/sunset without extra timezone
	// math; the app falls back to the local clock for day/night.
	return r, nil
}

// weatherAPICondition maps a WeatherAPI condition code to our set.
// Codes: https://www.weatherapi.com/docs/weather_conditions.json
func weatherAPICondition(code int) Condition {
	switch {
	case code == 1000:
		return Clear
	case code >= 1003 && code <= 1009:
		return Clouds // partly cloudy .. overcast
	case code == 1030:
		return Mist // mist
	case code >= 1063 && code <= 1080:
		return Rain // patchy rain, freezing rain, ice pellets
	case code == 1087:
		return Thunderstorm
	case code >= 1114 && code <= 1117:
		return Snow // blowing snow, blizzard
	case code >= 1135 && code <= 1147:
		return Mist // fog
	case code >= 1150 && code <= 1183:
		return Rain // light/ moderate drizzle
	case code >= 1186 && code <= 1246:
		return Rain // rain, heavy rain, thundery outbreaks
	case code == 1255 || code == 1258:
		return Snow
	case code == 1261 || code == 1264:
		return Rain // ice pellets
	case code >= 1273 && code <= 1282:
		return Thunderstorm
	default:
		return Clouds
	}
}

// ---------------------------------------------------------------------------
// Tomorrow.io (https://www.tomorrow.io) — free tier, needs a key. Unlike
// the others it requires coordinates, so the city is geocoded first via
// Open-Meteo's free geocoder.
// ---------------------------------------------------------------------------

// TomorrowIOClient queries the Tomorrow.io v4 timelines API for current
// conditions and a short 1-hour strip.
type TomorrowIOClient struct {
	APIKey string
	Query  string
	HTTP   *http.Client
	geo    *geoInfo
	geoURL string
	apiURL string
}

func NewTomorrowIOClient(apiKey, query string) *TomorrowIOClient {
	return &TomorrowIOClient{
		APIKey: apiKey,
		Query:  query,
		HTTP:   &http.Client{Timeout: 10 * time.Second},
		geoURL: "https://geocoding-api.open-meteo.com/v1/search",
		apiURL: "https://api.tomorrow.io/v4/timelines",
	}
}

func (c *TomorrowIOClient) Fetch(ctx context.Context) (Report, error) {
	if c.APIKey == "" {
		return Report{}, fmt.Errorf("tomorrow.io: no API key configured")
	}
	if c.Query == "" {
		return Report{}, fmt.Errorf("tomorrow.io: no location configured")
	}
	if c.geo == nil {
		g, err := geocodeCity(ctx, c.HTTP, c.geoURL, c.Query)
		if err != nil {
			return Report{}, fmt.Errorf("tomorrow.io geocode: %w", err)
		}
		c.geo = &g
	}
	loc := fmt.Sprintf("%.4f,%.4f", c.geo.Lat, c.geo.Lon)
	u := c.apiURL + "?" + url.Values{
		"apikey":    {c.APIKey},
		"location":  {loc},
		"fields":    {"weatherCode,temperature,humidity,windSpeed,windDirection"},
		"timesteps": {"current,1h"},
		"units":     {"metric"},
		"timezone":  {"auto"},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Report{}, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Report{}, fmt.Errorf("tomorrow.io: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Report{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Report{}, fmt.Errorf("tomorrow.io: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return parseTomorrow(body, c.geo)
}

// tomorrowResponse mirrors the timelines payload we use.
type tomorrowResponse struct {
	Data struct {
		Timelines []struct {
			Intervals []struct {
				StartTime string `json:"startTime"`
				Values    struct {
					WeatherCode string  `json:"weatherCode"`
					Temperature float64 `json:"temperature"`
					Humidity    float64 `json:"humidity"`
					WindSpeed   float64 `json:"windSpeed"`
					WindDir     float64 `json:"windDirection"`
				} `json:"values"`
			} `json:"intervals"`
		} `json:"timelines"`
	} `json:"data"`
}

func parseTomorrow(body []byte, g *geoInfo) (Report, error) {
	var t tomorrowResponse
	if err := json.Unmarshal(body, &t); err != nil {
		return Report{}, fmt.Errorf("tomorrow.io decode: %w", err)
	}
	if len(t.Data.Timelines) == 0 || len(t.Data.Timelines[0].Intervals) == 0 {
		return Report{}, fmt.Errorf("tomorrow.io: no data")
	}
	r := Report{
		City:       g.City,
		Condition:  tomorrowCondition(t.Data.Timelines[0].Intervals[0].Values.WeatherCode),
		TempKelvin: t.Data.Timelines[0].Intervals[0].Values.Temperature + 273.15,
		Humidity:   int(t.Data.Timelines[0].Intervals[0].Values.Humidity),
		WindMS:     t.Data.Timelines[0].Intervals[0].Values.WindSpeed,
		WindDir:    t.Data.Timelines[0].Intervals[0].Values.WindDir,
		Lat:        g.Lat,
		Lon:        g.Lon,
		FetchedAt:  time.Now(),
	}
	for i, iv := range t.Data.Timelines[0].Intervals {
		if i == 0 {
			continue // current
		}
		if i > 8 {
			break
		}
		if when, err := time.Parse(time.RFC3339, iv.StartTime); err == nil {
			r.Hourly = append(r.Hourly, HourPoint{
				When:      when,
				Condition: tomorrowCondition(iv.Values.WeatherCode),
				TempC:     iv.Values.Temperature,
			})
		}
	}
	return r, nil
}

// tomorrowCondition maps a Tomorrow.io weatherCode string to our set.
func tomorrowCondition(code string) Condition {
	switch strings.ToLower(code) {
	case "clear", "mostlyclear":
		return Clear
	case "partlycloudy", "mostlycloudy", "cloudly", "cloudy", "overcast":
		return Clouds
	case "fog", "lightfog", "densefog", "icefog", "freezingfog", "haze", "smoke", "dust", "sand":
		return Mist
	case "drizzle", "lightdrizzle", "heavydrizzle", "rain", "lightrain", "heavyrain",
		"freezingrain", "freezingdrizzle", "sleet", "icepellets":
		return Rain
	case "snow", "lightsnow", "heavysnow":
		return Snow
	case "thunderstorm", "thunderstormwithrain", "thunderstormwithhail":
		return Thunderstorm
	default:
		if strings.Contains(strings.ToLower(code), "snow") {
			return Snow
		}
		return Clouds
	}
}
