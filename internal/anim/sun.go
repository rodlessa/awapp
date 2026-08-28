package anim

import (
	"math"
	"math/rand"
	"time"

	"weatherterm/internal/render"
)

// SkyTimes carries a location's sunrise/sunset so the Sun and Moon can be
// positioned on screen according to their real time of day.
type SkyTimes struct {
	Sunrise, Sunset int64 // unix UTC seconds (0 = unknown)
	Timezone        int64
}

type sparkle struct {
	x, y  float64
	vy    float64
	phase float64
	shade uint8
}

// Sun renders clear skies.
//
// The Sun rises from the bottom of the screen, arcs across, and sets —
// its on-screen position follows the real sunrise/sunset times. It's a
// clean ASCII sun ('@' disc like the Moon, with '*' rays) over a
// warm-to-cool gradient; during a solar eclipse it gets eaten by the
// Moon's silhouette, the sky darkens, and a corona shows at totality.
//
// At night it becomes a twinkling starfield whose density reflects the
// local light pollution, plus a simple phase-accurate ASCII Moon that
// likewise rises and arcs across the night sky.
//
// The '+' and '-' keys resize the Sun and Moon (default: 15% of the
// terminal width).
type Sun struct {
	Night bool
	Opts  MoonOptions
	Times SkyTimes
	w, h  int

	sparkles []sparkle
	rng      *rand.Rand
	t        float64 // animation clock

	// starFactor (0..1) scales how many stars/motes are drawn. It is set
	// from the estimated local light pollution (see internal/light).
	starFactor float64
	// topMargin reserves rows at the top of the screen for the status
	// panel, so the Sun/Moon arc below it and never hide behind it.
	topMargin int
	// sizePct is the Sun/Moon diameter as a % of the terminal width.
	sizePct float64
	// Solar eclipse state (day scene): the Sun is covered by the Moon.
	SolarEclipse         bool
	SolarEclipseStart    time.Time
	SolarEclipseDuration time.Duration
}

func NewSun(night bool, opts MoonOptions) *Sun {
	return &Sun{
		Night:      night,
		Opts:       opts,
		rng:        rand.New(rand.NewSource(3)),
		starFactor: 0.5, // default until light-pollution data arrives
		sizePct:    15,  // default: 15% of the terminal width
	}
}

// SetTimes stores the location's sunrise/sunset for positioning the Sun
// and Moon by time of day.
func (s *Sun) SetTimes(t SkyTimes) { s.Times = t }

// SetSize sets the Sun/Moon diameter as a % of terminal width (clamped).
func (s *Sun) SetSize(pct float64) {
	if pct < 4 {
		pct = 4
	}
	if pct > 60 {
		pct = 60
	}
	s.sizePct = pct
}

// SetTopMargin reserves `rows` screen rows at the top (below which the
// Sun/Moon arc is drawn) so the status panel never covers them.
func (s *Sun) SetTopMargin(rows int) {
	if rows < 0 {
		rows = 0
	}
	s.topMargin = rows
}

// SizePct returns the current Sun/Moon diameter as a % of terminal width.
func (s *Sun) SizePct() float64 { return s.sizePct }

// SetMoon updates the Moon rendering options at runtime.
func (s *Sun) SetMoon(opts MoonOptions) { s.Opts = opts }

// Phase returns the current lunar phase fraction (0=new .. 0.5=full),
// honouring an override set for testing.
func (s *Sun) Phase() float64 {
	if s.Opts.PhaseOverride >= 0 {
		return s.Opts.PhaseOverride
	}
	return MoonPhaseAt(time.Now())
}

// MoonVisible reports whether the Moon should be drawn right now. In
// auto mode a new moon hides itself (there is nothing to see); the on/off
// modes override that.
func (s *Sun) MoonVisible() bool {
	if s.Opts.Auto {
		return MoonIllum(s.Phase()) > 0.02
	}
	return s.Opts.Visible
}

// Eclipse returns the current lunar-eclipse progress (0..1) and whether
// a lunar eclipse is active right now.
func (s *Sun) Eclipse() (float64, bool) {
	if !s.Opts.Eclipse {
		return 0, false
	}
	return EclipseProgress(s.Opts.EclipseStart, s.Opts.EclipseDuration, time.Now())
}

// Solar returns the current solar-eclipse progress (0..1) and whether a
// solar eclipse is active right now.
func (s *Sun) Solar() (float64, bool) {
	if !s.SolarEclipse {
		return 0, false
	}
	return EclipseProgress(s.SolarEclipseStart, s.SolarEclipseDuration, time.Now())
}

// SetSolar starts/stops a simulated solar eclipse.
func (s *Sun) SetSolar(active bool, start time.Time, dur time.Duration) {
	s.SolarEclipse = active
	s.SolarEclipseStart = start
	s.SolarEclipseDuration = dur
}

// SetStarFactor scales the star/mote density to match local light
// pollution and regenerates the field.
func (s *Sun) SetStarFactor(f float64) {
	if f < 0 {
		f = 0
	}
	if f > 1 {
		f = 1
	}
	s.starFactor = f
	s.reseedStars()
}

func (s *Sun) Resize(w, h int) {
	s.w, s.h = w, h
	s.reseedStars()
}

func (s *Sun) reseedStars() {
	if s.w == 0 || s.h == 0 {
		return
	}
	base := float64(s.w*s.h) / 60
	n := int(base * s.starFactor)
	if n < 4 && s.starFactor > 0 {
		n = 4 // even the most light-polluted sky still shows a few bright stars
	}
	if n > 250 {
		n = 250
	}
	s.sparkles = make([]sparkle, n)
	for i := range s.sparkles {
		s.sparkles[i] = s.newSparkle(true)
	}
}

func (s *Sun) newSparkle(randomY bool) sparkle {
	sp := sparkle{
		x:     s.rng.Float64() * float64(s.w),
		phase: s.rng.Float64() * math.Pi * 2,
	}
	if randomY {
		sp.y = s.rng.Float64() * float64(s.h)
	} else {
		sp.y = float64(s.h) - 1
	}
	if s.Night {
		sp.vy = 0 // stars don't fall, they twinkle in place
	} else {
		sp.vy = -(0.02 + s.rng.Float64()*0.04) // gentle upward drift, like heat shimmer/light motes
	}
	return sp
}

func (s *Sun) Tick() {
	s.t += 0.15
	for i := range s.sparkles {
		sp := &s.sparkles[i]
		sp.phase += 0.2
		if !s.Night {
			sp.y += sp.vy
			if sp.y < 0 {
				*sp = s.newSparkle(false)
			}
		}
	}
}

// discRadius returns the Sun/Moon half-width and half-height in cells for
// the configured size (% of terminal width), clamped to fit the screen.
func (s *Sun) discRadius() (rw, rh int) {
	rw = int(s.sizePct / 100 * float64(s.w) / 2)
	if rw < 3 {
		rw = 3
	}
	if maxRh := s.h / 3; rw/2 > maxRh {
		rw = maxRh * 2
	}
	rh = rw / 2
	if rh < 1 {
		rh = 1
	}
	return rw, rh
}

// sunProgress returns how far through the day the Sun is (0..1, rising to
// setting) and whether it is above the horizon.
func (s *Sun) sunProgress(now time.Time) (float64, bool) {
	if s.Times.Sunrise > 0 && s.Times.Sunset > s.Times.Sunrise {
		dayLen := float64(s.Times.Sunset - s.Times.Sunrise)
		el := float64(now.Unix()) - float64(s.Times.Sunrise)
		if el < 0 || el > dayLen {
			return 0, false
		}
		return el / dayLen, true
	}
	// No sunrise/sunset (e.g. offline pick): fall back to the local clock,
	// treating the day as 06:00-18:00.
	h := float64(now.Hour()) + float64(now.Minute())/60
	if h < 6 || h >= 18 {
		return 0, false
	}
	return (h - 6) / 12, true
}

// moonProgress returns how far through the night the Moon is (0..1) and
// whether it is above the horizon.
func (s *Sun) moonProgress(now time.Time) (float64, bool) {
	if s.Times.Sunrise > 0 && s.Times.Sunset > s.Times.Sunrise {
		set := float64(s.Times.Sunset)
		nightLen := float64(s.Times.Sunrise) + 86400 - set // next dawn (approx.)
		el := float64(now.Unix()) - set
		if el < 0 || el > nightLen {
			return 0, false
		}
		return el / nightLen, true
	}
	// Fallback: night is 18:00-06:00 local.
	h := float64(now.Hour()) + float64(now.Minute())/60
	if h >= 6 && h < 18 {
		return 0, false
	}
	var p float64
	if h >= 18 {
		p = (h - 18) / 12
	} else {
		p = (h + 6) / 12 // past local midnight: e.g. 02:00 -> 8/12
	}
	return p, true
}

// skyArc maps a body's progress through its day/night (0..1) to a screen
// position: it rises on the left near the bottom, arcs to the top at the
// middle, and sets on the right near the bottom. The arc is drawn below
// topMargin so the status panel doesn't cover the Sun/Moon at their peak.
func (s *Sun) skyArc(p float64) (int, int) {
	x0 := int(float64(s.w) * 0.12)
	x1 := int(float64(s.w) * 0.88)
	x := x0 + int(p*float64(x1-x0))
	alt := math.Sin(math.Pi * p)
	horizon := int(float64(s.h) * 0.80)
	usable := s.h - s.topMargin
	if usable < 4 {
		usable = 4
	}
	top := s.topMargin + int(float64(usable)*0.20)
	// Keep the whole disc below the top margin (the panel's bottom row),
	// so at the peak of the arc the Sun/Moon never grazes the panel.
	_, rh := s.discRadius()
	if minTop := s.topMargin + rh; top < minTop {
		top = minTop
	}
	if top > horizon-2 {
		top = horizon - 2
	}
	y := horizon - int(alt*float64(horizon-top))
	return x, y
}

func (s *Sun) Draw(buf *render.Buffer) {
	if s.Night {
		s.drawNight(buf)
		return
	}
	prog, active := s.Solar()
	s.drawDay(buf, prog, active)
}

// sunRamp is the Sun's braille density ramp, dense at the centre (⣿)
// and sparse toward the limb (⠁): "bigger dots in the middle, smaller
// dots at the borders", matching the braille Moon.
var sunRamp = []rune{'⠁', '⠃', '⠇', '⡇', '⣇', '⣧', '⣷', '⣿'}

func (s *Sun) drawDay(buf *render.Buffer, solarProg float64, solarActive bool) {
	// The Sun's on-screen position follows its real progress through the
	// day: near the bottom at sunrise, arcing up, and down to the horizon
	// at sunset.
	p, up := s.sunProgress(time.Now())
	if !up {
		p = 0.5 // the day scene should only run while the Sun is up
	}
	cx, cy := s.skyArc(p)

	// Darkness factor: 0 normally, ~0.85 at solar-eclipse totality.
	dark := 0.0
	if solarActive {
		dark = 0.85 * math.Sin(math.Pi*solarProg)
	}

	// Warm gradient sky, warmest near the Sun, cooled toward the edges and
	// darkened during an eclipse (the Sun is the light source).
	for y := 0; y < s.h; y++ {
		for x := 0; x < s.w; x++ {
			d := dist(x, y, cx, cy) / float64(s.w)
			buf.Set(x, y, ' ', darken(gradientShade(d), dark), false)
		}
	}

	// A braille disc (dense ⣿ at the centre, sparse ⠁ at the limb) with
	// fixed '*' rays. The rays hide at totality so the corona can shine.
	rw, rh := s.discRadius()
	total := solarActive && dark > 0.6
	if !total {
		rays := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}}
		for _, d := range rays {
			buf.Set(cx+d[0]*(rw+1), cy+d[1]*(rh+1), '*', 220, false)
			buf.Set(cx+d[0]*(rw+2), cy+d[1]*(rh+2), '*', 220, false)
		}
	}

	// '@' disc — with the Moon's silhouette sweeping across it during a
	// solar eclipse.
	var shX float64
	if solarActive {
		shX = 2.1 * (1 - 2*solarProg)
	}
	for dy := -rh; dy <= rh; dy++ {
		for dx := -rw; dx <= rw; dx++ {
			if dx*dx+4*dy*dy > rw*rw {
				continue
			}
			if solarActive {
				u := float64(dx) / float64(rw)
				v := float64(dy) / float64(rh)
				du := u - shX
				if du*du+v*v <= 1.05*1.05 {
					continue // covered by the Moon -> the Sun is blocked
				}
			}
			// Braille radial gradient: dense ⣿ at the centre, sparse ⠁ at
			// the limb — "bigger dots in the middle, smaller at the border".
			norm := math.Sqrt(float64(dx*dx)+float64(4*dy*dy)) / float64(rw)
			t := 1.0 - norm // 1 at the centre, 0 at the edge
			idx := int(t*float64(len(sunRamp)-1) + 0.5)
			if idx < 0 {
				idx = 0
			}
			if idx >= len(sunRamp) {
				idx = len(sunRamp) - 1
			}
			buf.Set(cx+dx, cy+dy, sunRamp[idx], 226, true)
		}
	}

	// Corona during totality: a sparse ring of light around the hidden Sun.
	if total {
		drawCorona(buf, cx, cy, rw, rh)
	}
}

func (s *Sun) drawNight(buf *render.Buffer) {
	buf.Clear(17) // deep night blue

	// Twinkling starfield: brightness pulses per-star via its phase. The
	// number of stars reflects the local light pollution (starFactor).
	for _, sp := range s.sparkles {
		twinkle := (math.Sin(sp.phase) + 1) / 2 // 0..1
		shade := uint8(59 + twinkle*180)
		ch := '.'
		if twinkle > 0.7 {
			ch = '*'
		}
		buf.Set(int(sp.x), int(sp.y), ch, shade, twinkle > 0.85)
	}

	// The Moon rises, arcs across the night sky, and sets — its on-screen
	// position follows the time since sunset.
	if s.MoonVisible() {
		p, up := s.moonProgress(time.Now())
		if !up {
			p = 0.5 // the night scene should only run while the Moon is up
		}
		cx, cy := s.skyArc(p)
		rw, rh := s.discRadius()
		prog, active := s.Eclipse()
		phase := s.Phase()
		if active {
			phase = 0.5 // lunar eclipses only happen at full moon
		}
		DrawMoon(buf, cx, cy, rw, rh, phase, prog, active)
	}
}

func dist(x, y, cx, cy int) float64 {
	dx, dy := float64(x-cx), float64(y-cy)*2 // correct for cell aspect ratio
	return math.Sqrt(dx*dx + dy*dy)
}

// gradientShade maps a normalized distance (0=sun, 1=far edge) to a
// warm-to-cool 256-color ramp.
func gradientShade(d float64) uint8 {
	ramp := []uint8{223, 222, 216, 209, 174, 139, 103, 67, 61}
	idx := int(d * float64(len(ramp)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ramp) {
		idx = len(ramp) - 1
	}
	return ramp[idx]
}

// darken scales a color toward black by f (0..1).
func darken(shade uint8, f float64) uint8 {
	if f <= 0 {
		return shade
	}
	if f >= 1 {
		return 0
	}
	return uint8(float64(shade) * (1 - f))
}

// drawCorona paints a sparse ring of light just outside the solar disc,
// as seen during the totality of a solar eclipse.
func drawCorona(buf *render.Buffer, cx, cy, rw, rh int) {
	for dy := -rh - 2; dy <= rh+2; dy++ {
		for dx := -rw - 3; dx <= rw+3; dx++ {
			u := float64(dx) / float64(rw)
			v := float64(dy) / float64(rh)
			rr := u*u + v*v
			if rr < 1.04*1.04 || rr > 1.7*1.7 {
				continue
			}
			if (dx*13+dy*7)%3 != 0 {
				continue
			}
			buf.Set(cx+dx, cy+dy, '*', 226, false)
		}
	}
	// Bright inner rim (like the "diamond ring" right at totality).
	for dy := -rh; dy <= rh; dy++ {
		for dx := -rw; dx <= rw; dx++ {
			u := float64(dx) / float64(rw)
			v := float64(dy) / float64(rh)
			rr := u*u + v*v
			if rr < 0.9*0.9 || rr > 1.02*1.02 {
				continue
			}
			if (dx+dy)%2 != 0 {
				continue
			}
			buf.Set(cx+dx, cy+dy, '.', 226, false)
		}
	}
}
