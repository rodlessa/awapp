package light

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBortleFromRadiance(t *testing.T) {
	cases := map[float64]int{
		0.01: 1, 0.20: 3, 1.0: 5, 4.0: 7, 15.0: 9,
	}
	for rad, want := range cases {
		if got := bortleFromRadiance(rad); got != want {
			t.Errorf("bortleFromRadiance(%v) = %d, want %d", rad, got, want)
		}
	}
}

func TestBortleFromPopulation(t *testing.T) {
	cases := map[int]int{
		100: 3, 10000: 4, 100000: 5, 500000: 6, 2000000: 7, 10000000: 8,
	}
	for pop, want := range cases {
		if got := bortleFromPopulation(pop); got != want {
			t.Errorf("bortleFromPopulation(%d) = %d, want %d", pop, got, want)
		}
	}
}

func TestStarFactorTable(t *testing.T) {
	if StarFactor[0] != 0 || StarFactor[1] != 1.0 || StarFactor[9] != 0.04 {
		t.Errorf("unexpected StarFactor table: %v", StarFactor)
	}
	for i := 1; i <= 9; i++ {
		if StarFactor[i] <= 0 || StarFactor[i] > 1 {
			t.Errorf("StarFactor[%d] = %v out of range", i, StarFactor[i])
		}
	}
}

func TestParseRadiance(t *testing.T) {
	if f := parseRadiance("12.34"); f != 12.34 {
		t.Errorf("parse(12.34) = %v", f)
	}
	if f := parseRadiance("-999"); f != -1 {
		t.Errorf("parse(-999) = %v, want -1", f)
	}
	if f := parseRadiance("Invalid or missing authentication"); f != -1 {
		t.Errorf("parse(error text) = %v, want -1", f)
	}
	if f := parseRadiance(""); f != -1 {
		t.Errorf("parse(empty) = %v, want -1", f)
	}
}

func TestEstimatePopulation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"results":[{"name":"Warsaw","population":1702139}]}`)
	}))
	defer srv.Close()
	c := NewClient("")
	c.geoURL = srv.URL + "/search"
	r, err := c.Estimate(context.Background(), "Warsaw", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Bortle != 7 {
		t.Errorf("bortle = %d, want 7 (pop ~1.7M)", r.Bortle)
	}
	if r.Radiance != 0 {
		t.Errorf("radiance should be 0 for a population estimate, got %v", r.Radiance)
	}
}

func TestEstimateRadianceWithKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "0.15")
	}))
	defer srv.Close()
	c := NewClient("secret")
	c.radianceURL = srv.URL + "/QueryRaster/"
	r, err := c.Estimate(context.Background(), "Warsaw", 52.2, 21.0)
	if err != nil {
		t.Fatal(err)
	}
	if r.Bortle != 2 {
		t.Errorf("bortle = %d, want 2 (radiance 0.15)", r.Bortle)
	}
	if r.Radiance != 0.15 {
		t.Errorf("radiance = %v, want 0.15", r.Radiance)
	}
}

func TestEstimateFallsBackOnBadRadiance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "" {
			fmt.Fprint(w, "Invalid or missing authentication")
			return
		}
		fmt.Fprint(w, `{"results":[{"population":5000}]}`)
	}))
	defer srv.Close()
	c := NewClient("bogus")
	c.radianceURL = srv.URL + "/QueryRaster/"
	c.geoURL = srv.URL + "/search"
	r, err := c.Estimate(context.Background(), "Smallville", 10, 10)
	if err != nil {
		t.Fatal(err)
	}
	if r.Bortle != 4 { // pop 5000 -> Bortle 4
		t.Errorf("bortle = %d, want 4", r.Bortle)
	}
}
