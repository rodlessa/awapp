// Package air fetches the current Air Quality Index (US AQI) for a
// location from Open-Meteo's free air-quality API — no API key required.
package air

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultAQIURL = "https://air-quality-api.open-meteo.com/v1/air-quality"

// Report is a current US-AQI reading.
type Report struct {
	USAQI int
	Label string
}

// Client fetches air-quality readings.
type Client struct {
	HTTP   *http.Client
	aqiURL string // overridable for tests
}

func NewClient() *Client {
	return &Client{
		HTTP:   &http.Client{Timeout: 8 * time.Second},
		aqiURL: defaultAQIURL,
	}
}

// Fetch returns the US AQI at the given coordinates.
func (c *Client) Fetch(ctx context.Context, lat, lon float64) (Report, error) {
	u := c.aqiURL + "?" + url.Values{
		"latitude":  {fmt.Sprintf("%.4f", lat)},
		"longitude": {fmt.Sprintf("%.4f", lon)},
		"current":   {"us_aqi"},
		"timezone":  {"auto"},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Report{}, err
	}
	req.Header.Set("User-Agent", "awapp/1.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Report{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Report{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Report{}, fmt.Errorf("air: HTTP %d", resp.StatusCode)
	}
	var a struct {
		Current struct {
			USAQI float64 `json:"us_aqi"`
		} `json:"current"`
	}
	if err := json.Unmarshal(body, &a); err != nil {
		return Report{}, fmt.Errorf("air: decode: %w", err)
	}
	aqi := int(a.Current.USAQI)
	return Report{USAQI: aqi, Label: labelFor(aqi)}, nil
}

// labelFor maps a US AQI value to its EPA category name.
func labelFor(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good"
	case aqi <= 100:
		return "Moderate"
	case aqi <= 150:
		return "Unhealthy for sensitive groups"
	case aqi <= 200:
		return "Unhealthy"
	case aqi <= 300:
		return "Very unhealthy"
	default:
		return "Hazardous"
	}
}
