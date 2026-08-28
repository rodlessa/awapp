// Package light estimates the local night-sky brightness (light
// pollution) for a location, so the night-sky star field can reflect how
// many stars a viewer would actually be able to see there.
//
// It works in two tiers:
//   - Exact: if an optional lightpollutionmap.info API key is configured,
//     it queries the actual sky radiance at the coordinates.
//   - Estimate: otherwise it geocodes the city with Open-Meteo (free, no
//     key) and maps its population to a Bortle class, which correlates
//     strongly with light pollution.
package light

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

const (
	defaultRadianceURL = "https://www.lightpollutionmap.info/QueryRaster/"
	defaultGeoURL      = "https://geocoding-api.open-meteo.com/v1/search"
)

// StarFactor maps a Bortle class (1 = pristine dark sky .. 9 = inner
// city) to the fraction of stars that remain visible there.
var StarFactor = [10]float64{0, 1.0, 0.85, 0.70, 0.55, 0.40, 0.28, 0.18, 0.10, 0.04}

// Report is an estimate of local night-sky brightness.
type Report struct {
	Bortle   int
	Radiance float64 // nW/cm2/sr when known, else 0
	Label    string  // short human description for the status panel
}

// Client fetches light-pollution estimates.
type Client struct {
	HTTP        *http.Client
	RadianceKey string // optional lightpollutionmap.info API key
	radianceURL string // overridable for tests
	geoURL      string // overridable for tests
}

func NewClient(radianceKey string) *Client {
	return &Client{
		HTTP:        &http.Client{Timeout: 8 * time.Second},
		RadianceKey: radianceKey,
		radianceURL: defaultRadianceURL,
		geoURL:      defaultGeoURL,
	}
}

// Estimate returns a Bortle-class estimate for a location. It prefers an
// exact radiance reading when a key is set, falling back to a population
// estimate on any failure.
func (c *Client) Estimate(ctx context.Context, city string, lat, lon float64) (Report, error) {
	if c.RadianceKey != "" {
		if r, err := c.radiance(ctx, lat, lon); err == nil {
			return r, nil
		}
		// fall through to the population estimate
	}
	pop, err := c.population(ctx, city)
	if err != nil {
		return Report{}, err
	}
	b := bortleFromPopulation(pop)
	return Report{
		Bortle: b,
		Label:  fmt.Sprintf("Bortle %d (city, pop %d)", b, pop),
	}, nil
}

// radiance queries the light-pollution map for the exact sky brightness
// at a coordinate (VIIRS night-lights radiance, nW/cm2/sr).
func (c *Client) radiance(ctx context.Context, lat, lon float64) (Report, error) {
	u := c.radianceURL + "?" + url.Values{
		"q":   {"WGS84"},
		"ql":  {"wa_2015"},
		"qt":  {"point"},
		"lat": {fmt.Sprintf("%.5f", lat)},
		"lon": {fmt.Sprintf("%.5f", lon)},
		"key": {c.RadianceKey},
	}.Encode()
	body, err := c.get(ctx, u)
	if err != nil {
		return Report{}, err
	}
	rad := parseRadiance(string(body))
	if rad < 0 {
		return Report{}, fmt.Errorf("lightpollutionmap: bad response %q", strings.TrimSpace(string(body)))
	}
	b := bortleFromRadiance(rad)
	return Report{
		Bortle:   b,
		Radiance: rad,
		Label:    fmt.Sprintf("Bortle %d (radiance %.2f)", b, rad),
	}, nil
}

// population geocodes a city name with Open-Meteo's free geocoder and
// returns its population.
func (c *Client) population(ctx context.Context, city string) (int, error) {
	if city == "" {
		return 0, fmt.Errorf("light: no city to geocode")
	}
	u := c.geoURL + "?" + url.Values{
		"name":  {city},
		"count": {"1"},
	}.Encode()
	body, err := c.get(ctx, u)
	if err != nil {
		return 0, err
	}
	var g struct {
		Results []struct {
			Population *int `json:"population"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &g); err != nil {
		return 0, err
	}
	if len(g.Results) == 0 || g.Results[0].Population == nil {
		return 0, fmt.Errorf("light: open-meteo has no result for %q", city)
	}
	return *g.Results[0].Population, nil
}

func (c *Client) get(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "weatherterm/1.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("light: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// parseRadiance parses the plain-number response from the light-pollution
// map; returns -1 when the value is unusable (error text, "no data", ...).
func parseRadiance(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return -1
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return -1
	}
	if f < 0 {
		return -1
	}
	return f
}

// bortleFromRadiance maps VIIRS night-lights radiance (nW/cm2/sr) to the
// Bortle dark-sky scale (1..9).
func bortleFromRadiance(rad float64) int {
	switch {
	case rad <= 0.05:
		return 1
	case rad <= 0.18:
		return 2
	case rad <= 0.36:
		return 3
	case rad <= 0.72:
		return 4
	case rad <= 1.50:
		return 5
	case rad <= 2.97:
		return 6
	case rad <= 6.0:
		return 7
	case rad <= 12.0:
		return 8
	default:
		return 9
	}
}

// bortleFromPopulation estimates a Bortle class from a city population
// (population size correlates strongly with artificial light at night).
func bortleFromPopulation(pop int) int {
	switch {
	case pop < 2000:
		return 3
	case pop < 20000:
		return 4
	case pop < 200000:
		return 5
	case pop < 1000000:
		return 6
	case pop < 5000000:
		return 7
	default:
		return 8
	}
}
