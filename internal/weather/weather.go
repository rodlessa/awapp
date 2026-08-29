// Package weather fetches current conditions from weather services and
// polls them on a fixed interval. It supports two interchangeable
// sources:
//
//   - OpenWeatherMap's "current weather" API (needs a free API key; the
//     city can be configured or geolocated from the IP address).
//   - A keyless IP-location source that geolocates via ipinfo.io and
//     reads current conditions from Open-Meteo (no registration).
package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Condition is the small set of visual states the renderer knows how
// to draw. Anything a weather source reports gets bucketed into one of
// these.
type Condition int

const (
	Clear Condition = iota
	Clouds
	Rain
	Thunderstorm
	Snow
	Mist
)

func (c Condition) String() string {
	switch c {
	case Clear:
		return "Clear"
	case Clouds:
		return "Clouds"
	case Rain:
		return "Rain"
	case Thunderstorm:
		return "Thunderstorm"
	case Snow:
		return "Snow"
	case Mist:
		return "Mist"
	default:
		return "Unknown"
	}
}

// HourPoint is one future hour in the forecast strip. TempC is Celsius.
type HourPoint struct {
	When      time.Time
	Condition Condition
	TempC     float64
}

// DayPoint is one future day in the forecast strip. Temps are Celsius.
type DayPoint struct {
	Day       time.Time
	Condition Condition
	HighC     float64
	LowC      float64
}

// Report is a normalized snapshot of current conditions.
type Report struct {
	City       string
	Condition  Condition
	Desc       string // human description, e.g. "light rain"
	TempKelvin float64
	Humidity   int
	WindMS     float64  // wind speed, meters per second
	WindDir    float64  // wind direction, degrees — where it comes FROM (0=N, 90=E, 180=S, 270=W)
	UVIndex    float64  // current UV index (0 = not provided)
	Alerts     []string // active severe-weather alerts (short text)
	Sunrise    int64    // unix UTC seconds, 0 if unknown
	Sunset     int64    // unix UTC seconds, 0 if unknown
	Timezone   int64    // seconds east of UTC (from the API)
	Lat, Lon   float64  // coordinates, for light-pollution lookups
	FetchedAt  time.Time
	// Hourly/Daily hold the forecast shown in the status-panel strip. They
	// are left empty when the source doesn't provide a forecast.
	Hourly []HourPoint
	Daily  []DayPoint
}

// TempC / TempF convert the stored Kelvin reading on demand, so the UI
// can flip units without a re-fetch.
func (r Report) TempC() float64 { return r.TempKelvin - 273.15 }
func (r Report) TempF() float64 { return r.TempKelvin*9/5 - 459.67 }

// Fetcher is implemented by the OpenWeatherMap and keyless IP clients.
type Fetcher interface {
	Fetch(ctx context.Context) (Report, error)
}

// ---------------------------------------------------------------------------
// IP geolocation (ipinfo.io, no key required)
// ---------------------------------------------------------------------------

type geoInfo struct {
	City     string
	Lat, Lon float64
}

func geolocate(ctx context.Context, hc *http.Client, u string) (geoInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return geoInfo{}, err
	}
	req.Header.Set("User-Agent", "awapp/1.0")
	resp, err := hc.Do(req)
	if err != nil {
		return geoInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return geoInfo{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return geoInfo{}, fmt.Errorf("geolocation: HTTP %d", resp.StatusCode)
	}
	var g struct {
		City string `json:"city"`
		Loc  string `json:"loc"` // "lat,lon"
	}
	if err := json.Unmarshal(body, &g); err != nil {
		return geoInfo{}, err
	}
	lat, lon := 0.0, 0.0
	if i := strings.IndexByte(g.Loc, ','); i >= 0 {
		lat, _ = strconv.ParseFloat(g.Loc[:i], 64)
		lon, _ = strconv.ParseFloat(g.Loc[i+1:], 64)
	}
	return geoInfo{City: g.City, Lat: lat, Lon: lon}, nil
}

// ---------------------------------------------------------------------------
// OpenWeatherMap client
// ---------------------------------------------------------------------------

// owmResponse mirrors the fields we need from OpenWeatherMap's
// /data/2.5/weather payload.
type owmResponse struct {
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
		ID          int    `json:"id"`
	} `json:"weather"`
	Main struct {
		Temp     float64 `json:"temp"`
		Humidity int     `json:"humidity"`
	} `json:"main"`
	Coord struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"coord"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   float64 `json:"deg"`
	} `json:"wind"`
	Sys struct {
		Sunrise int64 `json:"sunrise"`
		Sunset  int64 `json:"sunset"`
	} `json:"sys"`
	Dt       int64  `json:"dt"`
	Timezone int64  `json:"timezone"`
	Name     string `json:"name"`
	Cod      int    `json:"cod"`
}

// owmForecastResponse mirrors the fields we need from OpenWeatherMap's
// /data/2.5/forecast payload (3-hour steps).
type owmForecastResponse struct {
	List []struct {
		Dt   int64 `json:"dt"`
		Main struct {
			Temp float64 `json:"temp"`
		} `json:"main"`
		Weather []struct {
			ID int `json:"id"`
		} `json:"weather"`
	} `json:"list"`
}

// Client talks to the OpenWeatherMap API.
type Client struct {
	APIKey string
	City   string
	HTTP   *http.Client
	geo    *geoInfo
	geoURL string
}

func NewClient(apiKey, city string) *Client {
	return &Client{
		APIKey: apiKey,
		City:   city,
		HTTP:   &http.Client{Timeout: 10 * time.Second},
		geoURL: "https://ipinfo.io/json",
	}
}

// Fetch performs a single current-weather lookup. If no city was
// configured, the city is geolocated from the IP address (city-level,
// so it isn't invasive).
func (c *Client) Fetch(ctx context.Context) (Report, error) {
	if c.APIKey == "" {
		return Report{}, fmt.Errorf("no API key configured")
	}
	city := c.City
	if city == "" {
		g, err := c.geoInfo(ctx)
		if err != nil {
			return Report{}, fmt.Errorf("no city configured and geolocation failed: %w", err)
		}
		city = g.City
	}
	if city == "" {
		return Report{}, fmt.Errorf("no city configured")
	}

	u := "https://api.openweathermap.org/data/2.5/weather?" + url.Values{
		"q":     {city},
		"appid": {c.APIKey},
		"units": {"standard"}, // Kelvin; we convert locally for C/F
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Report{}, err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Report{}, fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Report{}, err
	}

	if resp.StatusCode != http.StatusOK {
		return Report{}, fmt.Errorf("openweathermap: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var owm owmResponse
	if err := json.Unmarshal(body, &owm); err != nil {
		return Report{}, fmt.Errorf("decode: %w", err)
	}
	if owm.Cod != 0 && owm.Cod != 200 {
		return Report{}, fmt.Errorf("openweathermap: cod=%d", owm.Cod)
	}

	r := Report{
		City:       owm.Name,
		TempKelvin: owm.Main.Temp,
		Humidity:   owm.Main.Humidity,
		WindMS:     owm.Wind.Speed,
		WindDir:    owm.Wind.Deg,
		Sunrise:    owm.Sys.Sunrise,
		Sunset:     owm.Sys.Sunset,
		Timezone:   owm.Timezone,
		Lat:        owm.Coord.Lat,
		Lon:        owm.Coord.Lon,
		FetchedAt:  time.Now(),
	}
	if r.City == "" {
		r.City = city
	}
	if len(owm.Weather) > 0 {
		r.Desc = owm.Weather[0].Description
		r.Condition = classify(owm.Weather[0].ID)
	}
	r.Hourly = c.fetchForecast(ctx, city)
	return r, nil
}

// fetchForecast is a best-effort second call to OpenWeatherMap's 3-hourly
// forecast endpoint, feeding the panel strip. Failures are swallowed — the
// current conditions are still valid without it.
func (c *Client) fetchForecast(ctx context.Context, city string) []HourPoint {
	u := "https://api.openweathermap.org/data/2.5/forecast?" + url.Values{
		"q":     {city},
		"appid": {c.APIKey},
		"units": {"metric"}, // Celsius; HourPoint stores Celsius
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil
	}
	var f owmForecastResponse
	if err := json.Unmarshal(body, &f); err != nil {
		return nil
	}
	out := make([]HourPoint, 0, 8)
	for i, e := range f.List {
		if i >= 8 {
			break
		}
		code := 800
		if len(e.Weather) > 0 {
			code = e.Weather[0].ID
		}
		out = append(out, HourPoint{
			When:      time.Unix(e.Dt, 0),
			Condition: classify(code),
			TempC:     e.Main.Temp,
		})
	}
	return out
}

func (c *Client) geoInfo(ctx context.Context) (geoInfo, error) {
	if c.geo == nil {
		g, err := geolocate(ctx, c.HTTP, c.geoURL)
		if err != nil {
			return geoInfo{}, err
		}
		c.geo = &g
	}
	return *c.geo, nil
}

// classify buckets an OpenWeatherMap condition code into our reduced
// set of renderable conditions. Codes: https://openweathermap.org/weather-conditions
func classify(id int) Condition {
	switch {
	case id >= 200 && id < 300:
		return Thunderstorm
	case id >= 300 && id < 400:
		return Rain // drizzle renders like light rain
	case id >= 500 && id < 600:
		return Rain
	case id >= 600 && id < 700:
		return Snow
	case id >= 700 && id < 800:
		return Mist // fog, haze, dust, smoke...
	case id == 800:
		return Clear
	case id > 800 && id < 900:
		return Clouds
	default:
		return Clouds
	}
}

// ---------------------------------------------------------------------------
// Keyless IP-location client (ipinfo.io + Open-Meteo)
// ---------------------------------------------------------------------------

// meteoResponse mirrors the fields we need from Open-Meteo's forecast
// endpoint (current conditions, today's sunrise/sunset, and the next few
// days of hourly/daily forecast for the panel strip).
type meteoResponse struct {
	UTCOffset int64 `json:"utc_offset_seconds"`
	Current   struct {
		Temperature float64 `json:"temperature_2m"`
		Humidity    float64 `json:"relative_humidity_2m"`
		Code        int     `json:"weather_code"`
		Wind        float64 `json:"wind_speed_10m"`
		WindDir     float64 `json:"wind_direction_10m"`
		UV          float64 `json:"uv_index"`
	} `json:"current"`
	Hourly struct {
		Time        []string  `json:"time"`
		Code        []int     `json:"weather_code"`
		Temperature []float64 `json:"temperature_2m"`
	} `json:"hourly"`
	Daily struct {
		Time    []string  `json:"time"`
		Code    []int     `json:"weather_code"`
		TempMax []float64 `json:"temperature_2m_max"`
		TempMin []float64 `json:"temperature_2m_min"`
		Sunrise []string  `json:"sunrise"`
		Sunset  []string  `json:"sunset"`
	} `json:"daily"`
	Alerts struct {
		Alert []struct {
			Event    string `json:"event"`
			Severity string `json:"severity"`
		} `json:"alert"`
	} `json:"alerts"`
}

// IPClient geolocates the user once and then reads current weather from
// Open-Meteo — no API key, no registration.
type IPClient struct {
	HTTP     *http.Client
	geo      *geoInfo
	geoURL   string
	meteoURL string
}

func NewIPClient() *IPClient {
	return &IPClient{
		HTTP:     &http.Client{Timeout: 10 * time.Second},
		geoURL:   "https://ipinfo.io/json",
		meteoURL: "https://api.open-meteo.com/v1/forecast",
	}
}

// Fetch geolocates once, then queries Open-Meteo for the current
// conditions at that location.
func (c *IPClient) Fetch(ctx context.Context) (Report, error) {
	if c.geo == nil {
		g, err := geolocate(ctx, c.HTTP, c.geoURL)
		if err != nil {
			return Report{}, fmt.Errorf("ip weather: %w", err)
		}
		c.geo = &g
	}
	return meteoFetch(ctx, c.HTTP, c.meteoURL, c.geo)
}

// CityClient resolves a named city with Open-Meteo's free geocoder and
// reads current conditions from Open-Meteo — no API key is required, so
// `-city` (or the config "city" key) alone is enough to get live weather
// for a specific place.
type CityClient struct {
	HTTP     *http.Client
	city     string
	geo      *geoInfo
	geoURL   string
	meteoURL string
}

func NewCityClient(city string) *CityClient {
	return &CityClient{
		HTTP:     &http.Client{Timeout: 10 * time.Second},
		city:     city,
		geoURL:   "https://geocoding-api.open-meteo.com/v1/search",
		meteoURL: "https://api.open-meteo.com/v1/forecast",
	}
}

// Fetch geocodes the city once, then queries Open-Meteo for its current
// conditions.
func (c *CityClient) Fetch(ctx context.Context) (Report, error) {
	if c.geo == nil {
		g, err := geocodeCity(ctx, c.HTTP, c.geoURL, c.city)
		if err != nil {
			return Report{}, fmt.Errorf("city weather: %w", err)
		}
		c.geo = &g
	}
	return meteoFetch(ctx, c.HTTP, c.meteoURL, c.geo)
}

// geocodeCity resolves a city name to coordinates and its canonical name
// using Open-Meteo's free geocoding API. It accepts the common
// "City,CC" form (e.g. "Fortaleza,BR") by splitting the country code
// into Open-Meteo's `country` parameter.
func geocodeCity(ctx context.Context, hc *http.Client, u, city string) (geoInfo, error) {
	name, country := city, ""
	if i := strings.IndexByte(city, ','); i >= 0 {
		name = strings.TrimSpace(city[:i])
		country = strings.TrimSpace(city[i+1:])
	}
	params := url.Values{
		"name":     {name},
		"count":    {"1"},
		"language": {"en"},
	}
	if country != "" {
		params.Set("country", country)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u+"?"+params.Encode(), nil)
	if err != nil {
		return geoInfo{}, err
	}
	req.Header.Set("User-Agent", "awapp/1.0")
	resp, err := hc.Do(req)
	if err != nil {
		return geoInfo{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return geoInfo{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return geoInfo{}, fmt.Errorf("geocode: HTTP %d", resp.StatusCode)
	}
	var g struct {
		Results []struct {
			Name      string  `json:"name"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &g); err != nil {
		return geoInfo{}, err
	}
	if len(g.Results) == 0 {
		return geoInfo{}, fmt.Errorf("geocode: no result for %q", city)
	}
	return geoInfo{
		City: g.Results[0].Name,
		Lat:  g.Results[0].Latitude,
		Lon:  g.Results[0].Longitude,
	}, nil
}

// meteoFetch queries Open-Meteo for current conditions at the given
// coordinates and builds a Report. Shared by the IP and city clients.
func meteoFetch(ctx context.Context, hc *http.Client, meteoURL string, g *geoInfo) (Report, error) {
	u := meteoURL + "?" + url.Values{
		"latitude":        {fmt.Sprintf("%.4f", g.Lat)},
		"longitude":       {fmt.Sprintf("%.4f", g.Lon)},
		"current":         {"temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m,wind_direction_10m,uv_index"},
		"hourly":          {"weather_code,temperature_2m"},
		"daily":           {"weather_code,temperature_2m_max,temperature_2m_min,sunrise,sunset"},
		"forecast_days":   {"3"},
		"alerts":          {"true"},
		"timezone":        {"auto"},
		"wind_speed_unit": {"ms"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Report{}, err
	}
	req.Header.Set("User-Agent", "awapp/1.0")
	resp, err := hc.Do(req)
	if err != nil {
		return Report{}, fmt.Errorf("open-meteo: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Report{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Report{}, fmt.Errorf("open-meteo: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var m meteoResponse
	if err := json.Unmarshal(body, &m); err != nil {
		return Report{}, fmt.Errorf("open-meteo decode: %w", err)
	}

	r := Report{
		City:       g.City,
		Condition:  classifyWMO(m.Current.Code),
		Desc:       wmoDescription(m.Current.Code),
		TempKelvin: m.Current.Temperature + 273.15, // Open-Meteo returns C
		Humidity:   int(m.Current.Humidity),
		WindMS:     m.Current.Wind,
		WindDir:    m.Current.WindDir,
		Sunrise:    parseLocalTime(first(m.Daily.Sunrise), m.UTCOffset),
		Sunset:     parseLocalTime(first(m.Daily.Sunset), m.UTCOffset),
		Timezone:   m.UTCOffset,
		Lat:        g.Lat,
		Lon:        g.Lon,
		FetchedAt:  time.Now(),
	}
	if r.City == "" {
		r.City = "your location"
	}
	r.Hourly = parseMeteoHourly(m.Hourly.Time, m.Hourly.Code, m.Hourly.Temperature)
	r.Daily = parseMeteoDaily(m.Daily.Time, m.Daily.Code, m.Daily.TempMax, m.Daily.TempMin)
	r.UVIndex = m.Current.UV
	for _, a := range m.Alerts.Alert {
		if len(r.Alerts) >= 3 {
			break
		}
		r.Alerts = append(r.Alerts, a.Event)
	}
	return r, nil
}

// parseMeteoHourly converts Open-Meteo's parallel hourly arrays into a
// bounded list of HourPoint entries (capped, and skipping stale hours).
func parseMeteoHourly(times []string, codes []int, temps []float64) []HourPoint {
	now := time.Now().Add(-15 * time.Minute) // skip the already-passed hour
	out := make([]HourPoint, 0, 12)
	for i := range times {
		if len(out) >= 12 {
			break
		}
		if i >= len(codes) || i >= len(temps) {
			break
		}
		when, err := parseMeteoTime(times[i])
		if err != nil {
			continue
		}
		if when.Before(now) {
			continue
		}
		out = append(out, HourPoint{
			When:      when,
			Condition: classifyWMO(codes[i]),
			TempC:     temps[i],
		})
	}
	return out
}

// parseMeteoTime parses an Open-Meteo timestamp, which is minute-precision
// local time ("2006-01-02T15:04"); seconds are rare but handled too.
func parseMeteoTime(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02T15:04", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

// parseMeteoDaily converts Open-Meteo's parallel daily arrays into
// DayPoint entries.
func parseMeteoDaily(times []string, codes []int, max, min []float64) []DayPoint {
	out := make([]DayPoint, 0, 3)
	for i := range times {
		if i >= len(codes) || i >= len(max) || i >= len(min) {
			break
		}
		d, err := time.Parse("2006-01-02", times[i])
		if err != nil {
			continue
		}
		out = append(out, DayPoint{
			Day:       d,
			Condition: classifyWMO(codes[i]),
			HighC:     max[i],
			LowC:      min[i],
		})
	}
	return out
}

// classifyWMO buckets an Open-Meteo WMO weather code into our condition set.
// Codes: https://open-meteo.com/en/docs
func classifyWMO(code int) Condition {
	switch {
	case code == 0 || code == 1:
		return Clear
	case code == 2 || code == 3:
		return Clouds
	case code == 45 || code == 48:
		return Mist
	case code >= 51 && code <= 57, code >= 61 && code <= 67, code >= 80 && code <= 82:
		return Rain
	case code >= 71 && code <= 77, code == 85 || code == 86:
		return Snow
	case code == 95 || code == 96 || code == 99:
		return Thunderstorm
	default:
		return Clouds
	}
}

// wmoDescription returns a short human description for a WMO code.
func wmoDescription(code int) string {
	desc := map[int]string{
		0: "clear sky", 1: "mainly clear", 2: "partly cloudy", 3: "overcast",
		45: "fog", 48: "rime fog",
		51: "light drizzle", 53: "drizzle", 55: "dense drizzle",
		56: "freezing drizzle", 57: "dense freezing drizzle",
		61: "light rain", 63: "rain", 65: "heavy rain",
		66: "freezing rain", 67: "heavy freezing rain",
		71: "light snow", 73: "snow", 75: "heavy snow",
		77: "snow grains",
		80: "light rain showers", 81: "rain showers", 82: "violent rain showers",
		85: "snow showers", 86: "heavy snow showers",
		95: "thunderstorm", 96: "thunderstorm with hail", 99: "thunderstorm with heavy hail",
	}
	if d, ok := desc[code]; ok {
		return d
	}
	return "mixed conditions"
}

// parseLocalTime converts a wall-clock time string (Open-Meteo format
// "2006-01-02T15:04", local time) to a UTC unix timestamp given the
// location's UTC offset in seconds.
func parseLocalTime(s string, offset int64) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse("2006-01-02T15:04", s)
	if err != nil {
		return 0
	}
	return t.Unix() - offset
}

func first(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

// ---------------------------------------------------------------------------
// Poller
// ---------------------------------------------------------------------------

// Poller fetches on a fixed interval and reports results (or errors)
// on channels. It keeps retrying on failure so the app can recover
// automatically once connectivity returns.
type Poller struct {
	fetcher  Fetcher
	interval time.Duration
	Reports  chan Report
	Errors   chan error
}

func NewPoller(fetcher Fetcher, interval time.Duration) *Poller {
	// A non-positive interval would make time.NewTicker panic, killing the
	// process while the terminal is in raw mode. Fall back to a sane default.
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Poller{
		fetcher:  fetcher,
		interval: interval,
		Reports:  make(chan Report, 1),
		Errors:   make(chan error, 1),
	}
}

// Run fetches immediately, then every interval, until ctx is canceled.
// A failed fetch does not stop the loop — it reports the error and
// tries again next tick, so a later recovery is picked up automatically.
func (p *Poller) Run(ctx context.Context) {
	p.tick(ctx)
	t := time.NewTicker(p.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	fctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	r, err := p.fetcher.Fetch(fctx)
	if err != nil {
		// Drain-then-send so the consumer always sees the *latest* failure
		// reason (mirrors Reports): if the app is still showing an older
		// error, replace it rather than dropping the more current one.
		select {
		case p.Errors <- err:
		default:
			select {
			case <-p.Errors:
			default:
			}
			p.Errors <- err
		}
		return
	}
	select {
	case p.Reports <- r:
	default:
		// drop if the previous report hasn't been consumed yet
		select {
		case <-p.Reports:
		default:
		}
		p.Reports <- r
	}
}
