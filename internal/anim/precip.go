package anim

import (
	"math"
	"math/rand"

	"awapp/internal/render"
)

// Mode selects the glyph/behavior family for Precip.
type Mode int

const (
	ModeRain Mode = iota
	ModeSnow
	ModeSleet // rain + snow mixed (freezing rain)
)

type drop struct {
	x, y  float64
	speed float64
	glyph rune
	shade uint8
	drift float64 // horizontal drift per tick (wind), snow only
	mode  Mode
}

// wDrop is a droplet running slowly down the "window" in front of the
// scene — much slower than the falling rain, slightly meandering, with a
// brighter leading edge, like condensation running down a pane.
type wDrop struct {
	x, y   float64
	speed  float64
	length int     // streak length in cells
	wob    float64 // wobble phase for a gentle side-to-side meander
	shade  uint8
}

// Precip animates falling rain or snow under a drifting cloud deck, with
// an optional lightning flash pass for thunderstorms, and slow droplets
// running down the window pane in front. The cloud deck uses the
// cloud-art library (tall storm clouds for thunder, a flat nimbostratus
// deck for plain rain, stratus for snow) and drifts with the reported
// wind. The deck can be hidden with the 'o' key.
type Precip struct {
	Mode     Mode
	Heavy    bool
	Thunder  bool
	CloudsOn bool // draw the cloud deck ('o' hides it, leaving just rain/snow)
	wind     float64
	windDir  float64 // degrees, where the wind comes FROM
	w, h     int
	// topMargin reserves rows at the top of the screen (the status panel)
	// so the cloud deck sits below it instead of hiding behind it.
	topMargin int
	drops     []drop
	wDrops    []wDrop
	flashLeft int
	t         float64 // animation clock
	rng       *rand.Rand
}

func NewPrecip(mode Mode, heavy, thunder bool, wind, windDir float64) *Precip {
	return &Precip{Mode: mode, Heavy: heavy, Thunder: thunder, CloudsOn: true, wind: wind, windDir: windDir, rng: rand.New(rand.NewSource(1))}
}

// SetTopMargin reserves rows at the top of the screen (the status panel)
// so the cloud deck drifts below it and never hides behind it.
func (p *Precip) SetTopMargin(rows int) {
	if rows < 0 {
		rows = 0
	}
	p.topMargin = rows
}

// SetClouds toggles the cloud deck (the 'o' key hides it, leaving just
// the falling rain or snow).
func (p *Precip) SetClouds(on bool) { p.CloudsOn = on }

func (p *Precip) Resize(w, h int) {
	p.w, p.h = w, h
	density := 0.08
	if p.Heavy {
		density = 0.14
	}
	if p.Mode == ModeSnow {
		density *= 0.5
	}
	n := int(float64(w*h) * density / 3)
	if n < 10 {
		n = 10
	}
	p.drops = make([]drop, n)
	for i := range p.drops {
		p.drops[i] = p.newDrop(true)
	}
	// Window droplets: a handful of slow streaks on the "glass", always
	// present during rain.
	wn := w / 6
	if wn < 3 {
		wn = 3
	}
	if wn > 24 {
		wn = 24
	}
	p.wDrops = make([]wDrop, wn)
	for i := range p.wDrops {
		p.wDrops[i] = p.newWDrop(true)
	}
}

func (p *Precip) newDrop(randomY bool) drop {
	mode := p.Mode
	if mode == ModeSleet {
		// Sleet: roughly half the drops fall as rain, half as snow.
		if p.rng.Float64() < 0.5 {
			mode = ModeRain
		} else {
			mode = ModeSnow
		}
	}
	d := drop{
		x:    p.rng.Float64() * float64(p.w),
		mode: mode,
	}
	if randomY {
		d.y = p.rng.Float64() * float64(p.h)
	} else {
		d.y = 0
	}
	if mode == ModeSnow {
		d.speed = 0.15 + p.rng.Float64()*0.25
		d.drift = (p.rng.Float64()-0.5)*0.3 + p.wind*0.03*windEastward(p.windDir)
		glyphs := []rune{'*', '.', '\'', '❄'}
		d.glyph = glyphs[p.rng.Intn(len(glyphs))]
		shades := []uint8{255, 253, 251}
		d.shade = shades[p.rng.Intn(len(shades))]
	} else {
		d.speed = 0.6 + p.rng.Float64()*0.8
		if p.Heavy {
			d.speed += 0.5
		}
		d.drift = p.wind * 0.03 * windEastward(p.windDir) // rain leans with the wind direction
		glyphs := []rune{'|', '¦', '\''}
		d.glyph = glyphs[p.rng.Intn(len(glyphs))]
		shades := []uint8{74, 73, 67, 111}
		d.shade = shades[p.rng.Intn(len(shades))]
	}
	return d
}

func (p *Precip) newWDrop(randomY bool) wDrop {
	wd := wDrop{
		x:      p.rng.Float64() * float64(p.w),
		speed:  0.06 + p.rng.Float64()*0.14, // much slower than falling rain
		length: 2 + p.rng.Intn(3),           // 2..4-cell streak
		wob:    p.rng.Float64() * math.Pi * 2,
		shade:  117,
	}
	if p.rng.Float64() < 0.3 {
		wd.shade = 111
	}
	if randomY {
		wd.y = p.rng.Float64() * float64(p.h)
	} else {
		wd.y = 0
	}
	return wd
}

// Tick advances physics by one frame; call once per animation tick
// before Draw.
func (p *Precip) Tick() {
	p.t += 0.15
	if p.w == 0 {
		return
	}
	for i := range p.drops {
		d := &p.drops[i]
		d.y += d.speed
		d.x += d.drift
		if d.y >= float64(p.h) {
			*d = p.newDrop(false)
		}
		if d.x < 0 {
			d.x = float64(p.w) - 1
		} else if d.x >= float64(p.w) {
			d.x = 0
		}
	}
	for i := range p.wDrops {
		wd := &p.wDrops[i]
		wd.y += wd.speed
		wd.x += math.Sin(p.t*0.4+wd.wob) * 0.02 // gentle meander down the glass
		if wd.y >= float64(p.h) {
			*wd = p.newWDrop(false)
			wd.x = p.rng.Float64() * float64(p.w)
		}
	}
	if p.Thunder {
		if p.flashLeft > 0 {
			p.flashLeft--
		} else if p.rng.Float64() < 0.01 {
			p.flashLeft = 2
		}
	}
}

func (p *Precip) Draw(buf *render.Buffer) {
	skyBase := uint8(238)
	if p.Thunder {
		skyBase = 236
	}
	if p.flashLeft > 0 {
		skyBase = 255
	}
	buf.Clear(skyBase)

	// Cloud deck across the top, drifting with the wind. During a
	// Cloud deck across the top, drifting with the wind and sitting below
	// the status panel (topMargin).
	p.drawCloudDeck(buf)

	if p.flashLeft > 0 {
		// Bright flash: keep drops visible but everything reads as lit.
		for _, d := range p.drops {
			buf.Set(int(d.x), int(d.y), d.glyph, 255, true)
		}
		return
	}

	for _, d := range p.drops {
		buf.Set(int(d.x), int(d.y), d.glyph, d.shade, d.mode == ModeSnow)
	}

	// Rain on the window: slow streaks running down the pane in front of
	// the scene, with a brighter leading edge.
	if p.Mode == ModeRain {
		for _, wd := range p.wDrops {
			for i := 0; i < wd.length; i++ {
				buf.Set(int(wd.x), int(wd.y)-i, '¦', wd.shade, false)
			}
			buf.Set(int(wd.x), int(wd.y), '\'', 255, false)
		}
	}
}

// drawCloudDeck tiles the matching cloud art along the top of the scene,
// just below the status panel.
func (p *Precip) drawCloudDeck(buf *render.Buffer) {
	if !p.CloudsOn {
		return // clouds hidden ('o' key)
	}
	var art []string
	var shade uint8
	switch {
	case p.Thunder:
		art, shade = stormArt(p.windDir), 235
	case p.Mode == ModeSnow:
		art, shade = cloudArtStratus, 245
	default:
		art, shade = stormArt(p.windDir), 238 // rain deck leans with the wind
	}
	if len(art) >= p.h-p.topMargin {
		return // terminal too short for a cloud deck
	}
	drawCloudBand(buf, p.topMargin, art, p.w, p.t, 0.3+p.wind*0.05, shade)
}
