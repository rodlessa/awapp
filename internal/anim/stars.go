package anim

import (
	"math"
	"time"

	"awapp/internal/render"
)

// star is one entry in the embedded bright-star catalog (J2000 equinox).
// RA is in hours (0..24), Dec in degrees (-90..90), Vmag the apparent
// visual magnitude (smaller = brighter).
type star struct {
	raH  float64
	dec  float64
	vmag float64
}

// starCatalog holds the ~50 brightest stars in the sky — enough to draw a
// recognizable real sky (Sirius, Orion, the Big Dipper, the Southern
// Cross, …) from any latitude. Positions are fixed (J2000); the apparent
// drift from precession is far too small to matter at terminal
// resolution. It is embedded in the binary, so the night sky needs no
// network access at all.
var starCatalog = []star{
	{6.7525, -16.716, -1.46},  // Sirius
	{6.3992, -52.696, -0.74},  // Canopus
	{14.2608, 19.182, -0.05},  // Arcturus
	{14.6603, -60.834, -0.01}, // Rigil Kentaurus (Alpha Cen)
	{18.6156, 38.784, 0.03},   // Vega
	{5.2767, 45.998, 0.08},    // Capella
	{5.2423, -8.202, 0.18},    // Rigel
	{7.6550, 5.225, 0.40},     // Procyon
	{1.6294, -57.237, 0.45},   // Achernar
	{5.9195, 7.407, 0.50},     // Betelgeuse
	{14.0617, -60.373, 0.61},  // Hadar (Beta Cen)
	{19.8476, 8.868, 0.76},    // Altair
	{12.4438, -63.099, 0.77},  // Acrux
	{4.5998, 16.509, 0.87},    // Aldebaran
	{13.4199, -11.161, 0.97},  // Spica
	{16.4932, -26.432, 1.09},  // Antares
	{7.7553, 28.026, 1.14},    // Pollux
	{22.9615, -29.622, 1.16},  // Fomalhaut
	{20.6909, 45.280, 1.25},   // Deneb
	{12.7925, -59.689, 1.25},  // Mimosa (Beta Cru)
	{10.1397, 11.967, 1.35},   // Regulus
	{6.9774, -28.972, 1.50},   // Adhara
	{7.5785, 31.888, 1.58},    // Castor
	{17.5609, -37.104, 1.62},  // Shaula
	{12.5189, -57.114, 1.63},  // Gacrux
	{5.4195, 6.350, 1.64},     // Bellatrix
	{5.4372, 28.607, 1.65},    // Elnath
	{9.2206, -69.717, 1.69},   // Miaplacidus
	{5.6042, -1.202, 1.69},    // Alnilam
	{22.1407, -46.961, 1.74},  // Alnair
	{5.6793, -1.943, 1.77},    // Alnitak
	{12.9006, 55.960, 1.77},   // Alioth
	{3.4057, 49.861, 1.80},    // Mirfak
	{11.0629, 61.751, 1.81},   // Dubhe
	{7.1358, -26.393, 1.83},   // Wezen
	{18.4077, -34.385, 1.85},  // Kaus Australis
	{13.7877, 49.313, 1.85},   // Alkaid
	{8.3727, -59.510, 1.86},   // Avior
	{17.6083, -42.998, 1.86},  // Sargas
	{5.9860, 44.947, 1.90},    // Menkalinan
	{16.8100, -69.028, 1.91},  // Atria
	{6.6301, 16.399, 1.92},    // Alhena
	{20.4289, -56.735, 1.94},  // Peacock
	{8.7434, -54.709, 1.95},   // Alsephina (Delta Vel)
	{6.3722, -17.956, 1.98},   // Mirzam
	{9.4624, -8.659, 1.98},    // Alphard
	{2.5317, 89.264, 1.98},    // Polaris
	{2.1109, 23.462, 2.00},    // Hamal
	{10.3352, 19.842, 2.01},   // Algieba
	{0.7245, -17.987, 2.02},   // Diphda
	// Second tier: the next-brightest stars that fill in the familiar
	// constellations (Orion's belt, the Big Dipper, Cassiopeia, …). They
	// only show under darker skies (see the light-pollution gating in
	// Draw).
	{17.5822, 12.560, 2.08},  // Rasalhague
	{5.7959, -9.670, 2.07},   // Saiph
	{5.5334, -0.299, 2.25},   // Mintaka
	{13.3988, 54.925, 2.23},  // Mizar
	{11.0307, 56.382, 2.34},  // Merak
	{11.8972, 53.695, 2.41},  // Phecda
	{12.2571, 57.033, 3.32},  // Megrez
	{0.6751, 56.537, 2.24},   // Schedar
	{0.1530, 59.150, 2.28},   // Caph
	{23.0794, 15.205, 2.49},  // Markab
	{23.0629, 28.083, 2.42},  // Scheat
	{21.7364, 9.875, 2.39},   // Enif
	{11.8177, 14.572, 2.14},  // Denebola
	{17.9434, 51.489, 2.24},  // Eltanin
	{20.3705, 40.257, 2.23},  // Sadr
	{19.5120, 27.960, 3.18},  // Albireo
	{18.9211, -26.297, 2.05}, // Nunki
	{8.1589, -47.337, 1.75},  // Regor
	{0.4381, -42.306, 2.39},  // Ankaa
	{3.1361, 40.956, 2.12},   // Algol
	{1.9107, 20.808, 2.64},   // Sheratan
	{8.0598, -40.003, 2.21},  // Naos
	{9.1333, -43.433, 2.21},  // Suhail
	{1.4302, 60.235, 2.68},   // Ruchbah
	{0.9451, 60.717, 2.47},   // Navi
	{14.0731, 64.376, 3.65},  // Thuban
	{3.7914, 24.105, 2.87},   // Alcyone
}

// StarField renders the real night sky: every catalog star is moved from
// its equatorial position (RA/Dec) into the observer's horizontal frame
// (altitude/azimuth) using the local sidereal time, then projected onto
// the terminal. It needs no network — just the clock and the observer's
// coordinates, both of which the app already has from the weather
// lookup.
type StarField struct {
	lat, lon  float64
	set       bool // location has been provided
	visible   bool // hidden by the app's Stars:off mode
	w, h      int
	topMargin int
	skyline   int   // 0=none .. 3=dense downtown (scaled from population/pollution)
	starMax   uint8 // brightest star shade (theme-dependent)
}

func NewStarField() *StarField {
	return &StarField{visible: true, starMax: 245}
}

// SetVisible hides (false) or shows (true) the field — used by the
// app's Stars:off toggle, which hides stars without touching the Moon.
func (f *StarField) SetVisible(v bool) { f.visible = v }

// SetLocation updates the observer's latitude/longitude (east positive).
func (f *StarField) SetLocation(lat, lon float64) {
	f.lat = lat
	f.lon = lon
	f.set = true
}

func (f *StarField) Resize(w, h int) { f.w, f.h = w, h }

func (f *StarField) SetTopMargin(rows int) { f.topMargin = rows }

// SetSkyline sets the city-skyline level (0 = none, 1..3 = growing
// city). The app scales it from the light-pollution/population estimate,
// so bigger cities get a taller, denser silhouette.
func (f *StarField) SetSkyline(level int) {
	if level < 0 {
		level = 0
	}
	if level > 3 {
		level = 3
	}
	f.skyline = level
}

// SetStarMax caps the brightest star shade (used by themes).
func (f *StarField) SetStarMax(v uint8) {
	if v < 60 {
		v = 60
	}
	f.starMax = v
}

// Draw paints the stars currently above the observer's horizon. `factor`
// is the light-pollution level (0 = bright city, 1 = pristine sky) and
// controls the faintest star that survives the glow: roughly magnitude
// 1 in a city, 5 under a dark sky.
func (f *StarField) Draw(buf *render.Buffer, factor float64, now time.Time) {
	if f.w == 0 || f.h == 0 || !f.set || !f.visible {
		return
	}
	jd := julianDay(now.UTC())
	horizon := f.horizonY()
	span := float64(horizon - f.topMargin)
	center := float64(f.w) / 2
	halfSpan := float64(f.w) * 0.40 // the sky spans ~80% of the width
	magLimit := 1.0 + factor*4.0

	for i, st := range starCatalog {
		if st.vmag > magLimit {
			continue // washed out by light pollution
		}
		alt, az := altAz(f.lat, f.lon, st.raH, st.dec, jd)
		if alt <= 0 {
			continue // below the horizon
		}
		x := f.projectX(az, center, halfSpan)
		y := f.projectY(alt, horizon, span)
		if x < 0 || y < 0 || x >= f.w || y >= f.h {
			continue
		}
		// Brightness falls off from the brightest (Sirius) down to the
		// magnitude limit; brighter stars are drawn '*' and near-white.
		bright := 1 - (st.vmag+1.5)/(magLimit+1.5)
		if bright < 0 {
			bright = 0
		}
		if bright > 1 {
			bright = 1
		}
		// Stable per-star phase so the whole sky doesn't twinkle in sync.
		tw := (math.Sin(float64(i)*1.7+float64(now.UnixNano())/4e8) + 1) / 2
		shade := uint8(60 + bright*185)
		if shade > f.starMax {
			shade = f.starMax
		}
		ch := '.'
		if bright > 0.45 {
			ch = '*'
		}
		buf.Set(x, y, ch, shade, tw > 0.8 && bright > 0.3)
	}

	// City skyline silhouette, scaled by the local light pollution.
	if f.skyline > 0 {
		f.drawSkyline(buf)
	}
	// Meteor-shower easter egg on known peak dates.
	if meteorActive(now) {
		f.drawMeteor(buf, now)
	}
}

// horizonY returns the screen row of the horizon: near the bottom, below
// the reserved top-margin rows (status panel).
func (f *StarField) horizonY() int {
	usable := f.h - f.topMargin
	if usable < 4 {
		usable = 4
	}
	h := f.topMargin + int(float64(usable)*0.85)
	if h >= f.h {
		h = f.h - 1
	}
	if h < f.topMargin {
		h = f.topMargin
	}
	return h
}

// projectX maps azimuth (0=N, 90=E, 180=S, 270=W) to a screen column:
// east on the left (where stars rise), south at the centre, west on the
// right (where they set).
func (f *StarField) projectX(azDeg, center, halfSpan float64) int {
	a := azDeg - 180 // S at 0, E at -90, W at +90
	if a < -180 {
		a += 360
	}
	if a > 180 {
		a -= 360
	}
	return int(center + a/180*halfSpan)
}

// projectY maps altitude (0..90) to a screen row: the horizon near the
// bottom, the zenith near the top (below the status panel).
func (f *StarField) projectY(altDeg float64, horizon int, span float64) int {
	if altDeg < 0 {
		altDeg = 0
	}
	if altDeg > 90 {
		altDeg = 90
	}
	return int(float64(horizon) - altDeg/90*span)
}

// gmstHours returns Greenwich Mean Sidereal Time in hours for a Julian
// date (UTC). The linear approximation is accurate to well under a
// minute over a century — far more than a terminal sky needs.
func gmstHours(jd float64) float64 {
	const j2000 = 2451545.0
	g := 18.697374558 + 24.06570982441908*(jd-j2000)
	g = math.Mod(g, 24)
	if g < 0 {
		g += 24
	}
	return g
}

// localSiderealHours returns Local Sidereal Time in hours for an
// observer at longitude lonDeg (east positive).
func localSiderealHours(jd, lonDeg float64) float64 {
	l := gmstHours(jd) + lonDeg/15
	l = math.Mod(l, 24)
	if l < 0 {
		l += 24
	}
	return l
}

// altAz converts a star's equatorial coordinates (RA in hours, Dec in
// degrees) into the observer's horizontal frame. It returns altitude and
// azimuth in degrees; a positive altitude means the star is above the
// horizon.
func altAz(latDeg, lonDeg, raH, decDeg, jd float64) (float64, float64) {
	lat := latDeg * math.Pi / 180
	dec := decDeg * math.Pi / 180
	ha := (localSiderealHours(jd, lonDeg) - raH) * 15 * math.Pi / 180

	sinAlt := math.Sin(lat)*math.Sin(dec) + math.Cos(lat)*math.Cos(dec)*math.Cos(ha)
	alt := math.Asin(clampf(sinAlt, -1, 1))

	if math.Abs(math.Cos(alt)) < 1e-9 || math.Abs(math.Cos(lat)) < 1e-9 {
		// At the zenith or at a pole the azimuth is undefined; the
		// altitude alone still tells us whether the star is up.
		return alt * 180 / math.Pi, 0
	}
	cosAz := (math.Sin(dec) - math.Sin(alt)*math.Sin(lat)) / (math.Cos(alt) * math.Cos(lat))
	az := math.Acos(clampf(cosAz, -1, 1))
	if math.Sin(ha) > 0 {
		az = 2*math.Pi - az
	}
	return alt * 180 / math.Pi, az * 180 / math.Pi
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// drawSkyline paints a dark building silhouette along the bottom of the
// night scene. Taller/denser for larger skylines (1..3).
func (f *StarField) drawSkyline(buf *render.Buffer) {
	horizon := f.horizonY()
	ground := f.h
	if horizon >= ground-1 {
		return
	}
	maxH := ground - horizon - 1
	if maxH < 2 {
		return
	}
	seed := f.skyline * 7919
	x := 0
	for x < f.w {
		width := 2 + ((seed+x)%3)*2
		height := 2 + ((seed + x*7) % 4) + f.skyline
		if height > maxH {
			height = maxH
		}
		for xx := x; xx < x+width && xx < f.w; xx++ {
			for yy := ground - height; yy < ground; yy++ {
				if (xx*7+yy)%5 == 0 {
					buf.Set(xx, yy, '░', 214, false) // lit window
				} else {
					buf.Set(xx, yy, '█', 235, false)
				}
			}
		}
		x += width + 1
	}
}

// meteorActive reports whether `now` falls inside a known meteor-shower
// peak window (approximate dates; fine for an easter egg).
func meteorActive(now time.Time) bool {
	showers := []struct{ m, d1, d2 int }{
		{1, 3, 4},    // Quadrantids ~Jan 3
		{4, 21, 23},  // Lyrids ~Apr 22
		{8, 10, 13},  // Perseids ~Aug 12
		{10, 20, 23}, // Orionids ~Oct 21
		{11, 16, 18}, // Leonids ~Nov 17
		{12, 12, 15}, // Geminids ~Dec 14
	}
	m := int(now.Month())
	d := now.Day()
	for _, s := range showers {
		if m == s.m && d >= s.d1 && d <= s.d2 {
			return true
		}
	}
	return false
}

// drawMeteor occasionally streaks a meteor across the sky during shower
// windows (time-based, so it's deterministic enough to test).
func (f *StarField) drawMeteor(buf *render.Buffer, now time.Time) {
	t := now.Unix()
	if t%6 != 0 {
		return
	}
	x := int((t / 6) % int64(f.w))
	y := f.topMargin + 1 + int((t/11)%4)
	for i := 0; i < 5; i++ {
		if x+i >= f.w || y+i >= f.h {
			break
		}
		buf.Set(x+i, y+i, '░', 231, false)
	}
	buf.Set(x, y, '☆', 255, true)
}
