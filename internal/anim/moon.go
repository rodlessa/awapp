package anim

import (
	"math"
	"time"

	"awapp/internal/render"
)

// MoonOptions controls how the Moon is rendered in the night scene.
type MoonOptions struct {
	// Auto lets the phase decide: a new moon (essentially 0% lit) is
	// hidden because there's nothing to see. When Auto is false, Visible
	// decides directly ("on" always draws, "off" never does).
	Auto    bool
	Visible bool
	// Eclipse simulates a lunar eclipse without polling anything: while
	// active, Earth's shadow sweeps across the Moon and it visibly
	// disappears at totality, then returns. EclipseDuration is the full
	// start-to-end length of the simulated event.
	Eclipse         bool
	EclipseStart    time.Time
	EclipseDuration time.Duration
	// PhaseOverride, when >= 0, pins the phase to this value (0..1)
	// instead of computing it from the date. Handy for testing.
	PhaseOverride float64
}

// MoonPhaseAt returns the lunar phase fraction for t: 0 = new moon,
// 0.25 = first quarter, 0.5 = full moon, 0.75 = last quarter, and
// 1 = new again. It uses the standard mean-lunation approximation
// (reference new moon 2000-01-06 18:14 UTC), accurate to well within
// a day for visualization purposes, and needs no network or ephemeris.
func MoonPhaseAt(t time.Time) float64 {
	return phaseFrac(julianDay(t.UTC()))
}

// MoonPhaseName maps a phase fraction to its conventional name.
func MoonPhaseName(p float64) string {
	p = p - math.Floor(p)
	switch {
	case p < 0.03 || p > 0.97:
		return "new moon"
	case p < 0.22:
		return "waxing crescent"
	case p < 0.28:
		return "first quarter"
	case p < 0.47:
		return "waxing gibbous"
	case p < 0.53:
		return "full moon"
	case p < 0.72:
		return "waning gibbous"
	case p < 0.78:
		return "last quarter"
	default:
		return "waning crescent"
	}
}

// MoonIllum returns the fraction (0..1) of the Moon's disc that is lit.
func MoonIllum(p float64) float64 {
	p = p - math.Floor(p)
	return (1 - math.Cos(2*math.Pi*p)) / 2
}

// EclipseProgress reports how far through a simulated eclipse the moment
// `now` is (0..1: 0 = first contact, 0.5 = totality, 1 = last contact)
// and whether the eclipse is still active. It returns inactive once the
// duration has elapsed, so the app can switch the eclipse off by itself.
func EclipseProgress(start time.Time, dur time.Duration, now time.Time) (float64, bool) {
	if start.IsZero() || dur <= 0 {
		return 0, false
	}
	el := now.Sub(start)
	if el < 0 {
		return 0, true
	}
	if el >= dur {
		return 1, false
	}
	return float64(el) / float64(dur), true
}

// moonArtBlank is the braille blank cell used as transparent space in
// the moon sprite (the moon's craters and the spacing around its marks).
const moonArtBlank = '\u2800'

// moonArt is the full-moon sprite (braille dot patterns). It is drawn
// scaled onto the Moon's disc; blank cells are transparent, so the
// craters/spacing read as the night sky behind the Moon.
var moonArt = [][]rune{
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡀⣀⣀⢤⣤⣤⣤⣄⣀⣀⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠚⠙⢉⣉⣈⠍⠉⢙⠙⠙⠛⠿⠟⢛⢷⢦⡤⣀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠠⡵⡺⠋⠈⠉⠛⠐⠈⠂⠚⢲⣄⠀⢉⠊⠈⠿⢯⣟⣿⢦⣄⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠊⠘⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⡘⣯⡤⣊⡠⡤⠀⠀⠐⠹⣯⣝⡦⡀⠀⠀⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠀⡀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠁⠀⢀⠆⠜⠜⢫⡸⣖⣹⢷⠁⢁⡵⠖⢌⢊⡌⡄⡀⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠰⠁⠀⠀⠀⠀⠀⠀⠀⠀⢀⠀⠀⠀⢀⡂⣂⠀⡡⠔⠉⠘⠾⡀⢘⢱⠐⡅⡑⣭⠀⠘⡔⣄⠀⠀⠀⠀"),
	[]rune("⠀⠀⢀⠀⠀⠀⠀⠀⠀⠠⠀⠀⠀⠂⠀⠀⠄⠀⠀⠀⠀⠰⢧⠆⠈⠋⠀⠀⠀⠀⠀⠀⠀⠀⣀⠀⠀⢕⠊⣀⣢⣃⠵⣆⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠠⡀⠀⣜⡀⢤⣀⡠⣄⡀⡀⢀⠀⣤⡽⡷⡛⠁⠀⠀⠀⠀⠂⠀⠀⠈⢧⣝⣦⣻⡅⢷⠯⠀⢛⣢⠀⠀"),
	[]rune("⠀⠠⠀⠀⠀⠀⠀⠀⠈⠗⠀⢀⠩⠽⢼⠏⡺⠘⡌⠀⠀⠈⠐⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢘⠿⡭⢋⠙⡉⠟⣀⣱⣙⡆⠀"),
	[]rune("⠀⠌⠀⠀⠀⠀⠀⠀⠀⠀⠀⠉⠐⣀⠀⠈⠀⠘⠀⠀⢀⡤⡠⠀⠀⠀⠙⠀⠀⡃⠠⠀⠀⠀⠀⡁⠄⠰⣁⠎⡀⠄⠈⠐⡩⢳⠀"),
	[]rune("⠸⡀⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠆⠀⠀⠀⠀⠀⠘⠁⠁⣦⣀⣠⣐⡦⠘⠀⠀⠀⠀⠀⠀⠀⠀⠐⡪⢾⠆⠀⠀⠀⠠⣹⡆"),
	[]rune("⠐⢂⠀⠀⠀⠀⠀⠀⠀⠠⠀⠀⠀⢠⠘⠄⢠⣠⣐⡰⠀⣀⣩⣿⣼⣿⣟⠩⡂⠀⠀⠀⠀⠀⠀⠀⠀⠀⠐⢸⢆⠀⠀⠀⢠⠻⠃"),
	[]rune("⠀⠀⠄⠀⠒⠠⠤⠀⠀⠠⠀⠀⠀⠀⡠⠀⠀⢹⣽⢛⡿⢦⣻⣽⢿⣿⡾⣷⣅⡄⠀⠀⠀⠄⠀⠀⠀⠈⢠⠈⣋⠰⡠⣒⠜⡣⠀"),
	[]rune("⢠⢃⠁⠄⠮⠊⠀⠀⠀⢀⠄⠂⠀⠀⠜⠀⠀⠚⠑⢈⠀⢕⢼⣿⢿⣷⣽⣿⣿⡿⠳⠀⠠⢔⡯⡀⡠⠀⠀⠀⠘⠝⠹⠀⢀⡧⠄"),
	[]rune("⠀⠸⠁⢦⡀⠀⠀⠀⠉⠀⠂⠀⠀⠀⠀⠀⢀⡴⣷⡾⡞⣾⢻⣷⣿⣿⢿⣿⣷⣎⣄⠀⢜⣳⣌⡊⠄⠀⠀⠀⠀⠀⠈⢤⠡⡌⠀"),
	[]rune("⠀⠑⡌⠰⣤⠂⠀⢀⢠⠁⠀⡀⠀⠀⠈⠀⠊⣿⣯⣭⡿⡿⣟⣿⣿⡿⣯⣿⣿⣯⡱⠞⠀⡙⢿⣧⠀⠀⠀⠀⣀⡀⠈⠺⡘⠀⠀"),
	[]rune("⠀⠀⠄⠎⠠⠍⢐⠠⡅⢀⠃⡡⣀⣄⢠⣄⣴⣿⣽⣾⣿⣿⣿⣿⣿⣿⠷⣿⣿⣿⡏⡀⠀⠈⡰⠅⡰⢈⠀⠁⠪⣱⢷⠗⡇⠁⠀"),
	[]rune("⠀⠀⠘⠶⡀⠤⠦⣹⣬⣳⠾⣂⣥⠾⢻⣿⣿⣷⣿⣯⣿⣿⣿⣿⡽⣷⣿⣿⣿⣿⣧⣤⡶⢈⡀⠨⠀⠀⢀⠊⢨⢁⣍⠼⠁⠀⠀"),
	[]rune("⠀⠀⠀⠀⠑⠅⠂⠙⢻⣽⢶⣗⣼⡮⣛⣿⣿⣿⣿⣿⣿⣽⣿⣿⣾⣿⣿⣿⣿⣿⣿⣯⣶⣿⣿⡿⢂⢺⣟⠴⣀⣺⡸⠁⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠊⠲⠠⠈⣾⣼⣟⣿⣿⣿⣻⡿⣻⣿⣿⣿⣿⣿⣿⣿⣛⣿⣻⣿⣿⣿⣿⣿⣿⣿⣗⣿⣿⣹⠉⣛⠌⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠈⠬⣖⣴⡻⢿⣿⣿⡿⢿⢿⢛⣟⣾⣏⣻⣯⣟⣿⣿⣻⣿⣿⣿⣿⣿⣿⣿⢿⣿⢿⡿⠕⠊⠁⠀⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠒⢯⣻⢿⣿⣟⣿⣯⣿⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣻⠟⠫⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠓⠻⣙⡿⣻⣻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣵⣿⠋⢋⠉⠄⠂⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀"),
	[]rune("⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠉⠛⠛⠻⠿⢿⠿⠿⠿⠿⠿⠛⠋⠁⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀"),
}

// DrawMoon paints the Moon centred at (cx, cy), half-width rw and
// half-height rh cells. The moonArt sprite (both are ~2:1) is scaled
// onto that disc, then lit according to the real phase: for every
// on-disc cell we work out how much sunlight reaches that point on the
// near hemisphere — the Sun sits at angle θ = π − 2π·phase — giving the
// correct crescent / quarter / gibbous / full shapes, lit on the right
// while waxing and on the left while waning. Blank sprite cells stay
// transparent (the Moon's craters and spacing read as the night sky).
// During an eclipse Earth's shadow sweeps across, so the Moon shrinks,
// disappears at totality, then reappears.
func DrawMoon(buf *render.Buffer, cx, cy, rw, rh int, phase float64, eclipseProgress float64, eclipseActive bool) {
	if rw < 2 || rh < 1 {
		return
	}
	theta := math.Pi - 2*math.Pi*phase
	sinT, cosT := math.Sin(theta), math.Cos(theta)

	// Earth's shadow: a disc slightly larger than the Moon, swept across
	// the sky from right to left over the eclipse.
	var shX float64
	if eclipseActive {
		shX = 2.1 * (1 - 2*eclipseProgress)
	}
	const shR2 = 1.05 * 1.05

	discW := 2*rw + 1
	discH := 2*rh + 1
	artW := len(moonArt[0])
	artH := len(moonArt)

	for dy := -rh; dy <= rh; dy++ {
		for dx := -rw; dx <= rw; dx++ {
			u := float64(dx) / float64(rw)
			v := float64(dy) / float64(rh)
			if u*u+v*v > 1 {
				continue
			}
			z := math.Sqrt(1 - u*u - v*v)
			illum := u*sinT + z*cosT
			if illum <= 0.02 {
				continue
			}
			if eclipseActive {
				du := u - shX
				if du*du+v*v <= shR2 {
					continue // covered by Earth's shadow -> the Moon disappears
				}
			}
			// Map this disc cell onto the sprite (nearest-neighbour).
			sx := int(float64(dx+rw) * float64(artW) / float64(discW))
			sy := int(float64(dy+rh) * float64(artH) / float64(discH))
			if sx >= artW {
				sx = artW - 1
			}
			if sy >= artH {
				sy = artH - 1
			}
			ch := moonArt[sy][sx]
			if ch == moonArtBlank {
				continue // transparent: crater / spacing reads as the night sky
			}
			buf.Set(cx+dx, cy+dy, ch, 255, false)
		}
	}
}

// julianDay converts t (assumed UTC) to a Julian Day number using the
// standard Gregorian conversion.
func julianDay(t time.Time) float64 {
	y := float64(t.Year())
	m := float64(t.Month())
	d := float64(t.Day()) +
		float64(t.Hour())/24 +
		float64(t.Minute())/1440 +
		(float64(t.Second())+float64(t.Nanosecond())/1e9)/86400
	if m <= 2 {
		y--
		m += 12
	}
	a := math.Floor(y / 100)
	b := 2 - a + math.Floor(a/4)
	return math.Floor(365.25*(y+4716)) +
		math.Floor(30.6001*(m+1)) + d + b - 1524.5
}

func phaseFrac(jd float64) float64 {
	const (
		synodic      = 29.530588853 // mean synodic month, days
		newMoonRefJD = 2451550.1    // known new moon: 2000-01-06 18:14 UTC
	)
	lun := (jd - newMoonRefJD) / synodic
	f := lun - math.Floor(lun)
	if f < 0 {
		f++
	}
	return f
}
