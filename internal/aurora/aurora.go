// Package aurora estimates how likely the aurora is to be visible at a
// location, from NOAA's free planetary Kp index (no API key required).
// The Kp index is fetched once; it changes slowly, so the app re-polls it
// on the normal weather interval.
package aurora

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

const defaultKpURL = "https://services.swpc.noaa.gov/json/planetary_k_index_forecast.json"

// Report carries the current/forecast Kp index.
type Report struct {
	Kp float64 // current (or nearest forecast) planetary K-index, 0..9
	OK bool    // data actually loaded
}

// Client fetches the NOAA Kp forecast.
type Client struct {
	HTTP  *http.Client
	kpURL string // overridable for tests
}

func NewClient() *Client {
	return &Client{
		HTTP:  &http.Client{Timeout: 8 * time.Second},
		kpURL: defaultKpURL,
	}
}

// Fetch loads the forecast Kp index (the maximum predicted over the next
// few hours, which is what matters for aurora visibility tonight).
func (c *Client) Fetch(ctx context.Context) (Report, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.kpURL, nil)
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
		return Report{}, fmt.Errorf("aurora: HTTP %d", resp.StatusCode)
	}
	kp, err := parseKpForecast(body)
	if err != nil {
		return Report{}, err
	}
	return Report{Kp: kp, OK: true}, nil
}

// parseKpForecast extracts the highest predicted Kp in the forecast.
func parseKpForecast(body []byte) (float64, error) {
	var items []struct {
		Kp float64 `json:"predicted_kp_index"`
	}
	if err := json.Unmarshal(body, &items); err != nil {
		return 0, fmt.Errorf("aurora: decode: %w", err)
	}
	max := 0.0
	for _, it := range items {
		if it.Kp > max {
			max = it.Kp
		}
	}
	return max, nil
}

// Likelihood returns 0..1 for how visible the aurora is at a geographic
// latitude given the Kp index. It approximates the geomagnetic latitude
// (ignoring longitude) and the equatorward expansion of the oval with Kp:
// higher Kp pushes the aurora further from the pole.
func Likelihood(lat, kp float64) float64 {
	geo := math.Abs(lat)
	ovLat := 65 - kp*2.2 // approx equatorward edge of the oval
	if geo >= ovLat {
		d := (geo - ovLat) / (90 - ovLat)
		if d > 1 {
			d = 1
		}
		return 0.3 + 0.7*d
	}
	return 0
}
