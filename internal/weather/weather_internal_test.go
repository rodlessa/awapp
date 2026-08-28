package weather

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := map[int]Condition{
		200: Thunderstorm,
		301: Rain,
		500: Rain,
		600: Snow,
		701: Mist,
		800: Clear,
		802: Clouds,
	}
	for id, want := range cases {
		if got := classify(id); got != want {
			t.Errorf("classify(%d) = %v, want %v", id, got, want)
		}
	}
}

func TestClassifyWMO(t *testing.T) {
	cases := map[int]Condition{
		0: Clear, 1: Clear, 2: Clouds, 3: Clouds,
		45: Mist, 48: Mist,
		51: Rain, 61: Rain, 66: Rain, 80: Rain, 82: Rain,
		71: Snow, 77: Snow, 85: Snow, 86: Snow,
		95: Thunderstorm, 96: Thunderstorm, 99: Thunderstorm,
	}
	for code, want := range cases {
		if got := classifyWMO(code); got != want {
			t.Errorf("classifyWMO(%d) = %v, want %v", code, got, want)
		}
	}
}

func TestTempConversion(t *testing.T) {
	r := Report{TempKelvin: 300}
	if c := r.TempC(); c < 26.8 || c > 26.9 {
		t.Errorf("TempC = %v, want ~26.85", c)
	}
	if f := r.TempF(); f < 80.3 || f > 80.4 {
		t.Errorf("TempF = %v, want ~80.33", f)
	}
}

func TestClientFetchErrorsOnMissingConfig(t *testing.T) {
	c := NewClient("", "")
	if _, err := c.Fetch(nil); err == nil { //nolint:staticcheck // nil ctx fine, fails before use
		t.Fatal("expected error for missing API key")
	}
}

// router routes requests by their original host to test servers, so a
// client can talk to multiple "real-looking" endpoints in one test.
type router struct{ routes map[string]string }

func (r router) RoundTrip(req *http.Request) (*http.Response, error) {
	if target, ok := r.routes[req.URL.Host]; ok {
		u, err := url.Parse(target)
		if err != nil {
			return nil, err
		}
		req.URL.Scheme = u.Scheme
		req.URL.Host = u.Host
	}
	return http.DefaultTransport.RoundTrip(req)
}

func TestClientFetchParsesWindAndCoord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":"Warsaw","cod":200,"coord":{"lat":52.2,"lon":21.0},
			"main":{"temp":290.0,"humidity":60},"wind":{"speed":7.5},
			"sys":{"sunrise":1700000000,"sunset":1700036000},"timezone":3600,
			"weather":[{"id":802,"description":"scattered clouds"}]}`)
	}))
	defer srv.Close()
	c := NewClient("key", "Warsaw")
	c.HTTP = &http.Client{Transport: router{routes: map[string]string{"api.openweathermap.org": srv.URL}}}
	r, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.WindMS != 7.5 {
		t.Errorf("wind = %v, want 7.5", r.WindMS)
	}
	if r.Condition != Clouds {
		t.Errorf("condition = %v, want Clouds", r.Condition)
	}
	if r.Lat != 52.2 || r.Lon != 21.0 {
		t.Errorf("coord = %v,%v, want 52.2,21.0", r.Lat, r.Lon)
	}
}

func TestClientGeolocatesCityWhenEmpty(t *testing.T) {
	geo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"city":"Fortaleza","loc":"-3.7,-38.5"}`)
	}))
	defer geo.Close()
	owm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":"Fortaleza","cod":200,"coord":{"lat":-3.7,"lon":-38.5},
			"main":{"temp":300.0,"humidity":70},"wind":{"speed":3.0},
			"sys":{"sunrise":1700000000,"sunset":1700036000},"timezone":-10800,
			"weather":[{"id":800,"description":"clear sky"}]}`)
	}))
	defer owm.Close()

	c := NewClient("key", "") // no city -> should geolocate via IP
	c.HTTP = &http.Client{Transport: router{routes: map[string]string{
		"ipinfo.io":              geo.URL,
		"api.openweathermap.org": owm.URL,
	}}}
	r, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.City != "Fortaleza" {
		t.Errorf("city = %q, want Fortaleza (geolocated)", r.City)
	}
	if r.Condition != Clear {
		t.Errorf("condition = %v, want Clear", r.Condition)
	}
}

func TestCityClientFetch(t *testing.T) {
	geo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"results":[{"name":"Warsaw","latitude":52.2297,"longitude":21.0117}]}`)
	}))
	defer geo.Close()
	meteo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"utc_offset_seconds":3600,
			"current":{"temperature_2m":12.0,"relative_humidity_2m":70,"weather_code":61,"wind_speed_10m":3.1},
			"daily":{"sunrise":["2026-08-28T05:30"],"sunset":["2026-08-28T20:10"]}}`)
	}))
	defer meteo.Close()

	c := NewCityClient("Warsaw")
	c.geoURL = geo.URL + "/search"
	c.meteoURL = meteo.URL + "/forecast"

	r, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.City != "Warsaw" {
		t.Errorf("city = %q, want Warsaw", r.City)
	}
	if r.Condition != Rain {
		t.Errorf("condition = %v, want Rain (wmo 61)", r.Condition)
	}
	if c := r.TempC(); c < 11.9 || c > 12.1 {
		t.Errorf("TempC = %v, want ~12.0", c)
	}
	if r.WindMS != 3.1 {
		t.Errorf("wind = %v, want 3.1", r.WindMS)
	}
	if r.Lat != 52.2297 || r.Lon != 21.0117 {
		t.Errorf("coord = %v,%v", r.Lat, r.Lon)
	}
}

func TestGeocodeCityWithCountrySuffix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("name"); got != "Fortaleza" {
			t.Errorf("name = %q, want Fortaleza", got)
		}
		if got := r.URL.Query().Get("country"); got != "BR" {
			t.Errorf("country = %q, want BR", got)
		}
		fmt.Fprint(w, `{"results":[{"name":"Fortaleza","latitude":-3.7172,"longitude":-38.5431}]}`)
	}))
	defer srv.Close()
	g, err := geocodeCity(context.Background(), srv.Client(), srv.URL, "Fortaleza,BR")
	if err != nil {
		t.Fatal(err)
	}
	if g.City != "Fortaleza" || g.Lat != -3.7172 || g.Lon != -38.5431 {
		t.Errorf("geo = %+v", g)
	}
}

func TestIPClientFetch(t *testing.T) {
	geo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"city":"Warsaw","loc":"52.2297,21.0117"}`)
	}))
	defer geo.Close()
	meteo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"utc_offset_seconds":3600,
			"current":{"temperature_2m":18.5,"relative_humidity_2m":65,"weather_code":2,"wind_speed_10m":4.2},
			"daily":{"sunrise":["2026-08-28T05:30"],"sunset":["2026-08-28T20:10"]}}`)
	}))
	defer meteo.Close()

	c := NewIPClient()
	c.geoURL = geo.URL + "/json"
	c.meteoURL = meteo.URL + "/forecast"

	r, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.City != "Warsaw" {
		t.Errorf("city = %q, want Warsaw", r.City)
	}
	if r.Condition != Clouds {
		t.Errorf("condition = %v, want Clouds (wmo 2)", r.Condition)
	}
	if c := r.TempC(); c < 18.4 || c > 18.6 {
		t.Errorf("TempC = %v, want ~18.5", c)
	}
	if r.WindMS != 4.2 {
		t.Errorf("wind = %v, want 4.2", r.WindMS)
	}
	if r.Timezone != 3600 {
		t.Errorf("timezone = %v, want 3600", r.Timezone)
	}
	if r.Sunrise == 0 || r.Sunset == 0 {
		t.Error("expected sunrise/sunset to be parsed")
	}
}
