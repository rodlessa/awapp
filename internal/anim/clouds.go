package anim

import (
	"math"
	"math/rand"

	"awapp/internal/render"
)

// A library of ASCII cloud types so different weather can use different
// cloud art: puffy fair-weather cumulus, a flat overcast stratus layer,
// tall dark cumulonimbus storm clouds with rain curtains, thin wispy
// cirrus for haze, and a low nimbostratus rain deck.

// cloudArtCumulus: fair-weather puffy clouds.
var cloudArtCumulus = []string{
	"   .--.      ",
	".-(    ).    ",
	"(___.__)__)  ",
}

// cloudArtStratus: a flat, wide overcast layer.
var cloudArtStratus = []string{
	"     __________     ",
	"  .-'          '-.  ",
	".-'              '-.",
	"'-------------------'",
}

// windEastward returns the signed eastward component of the wind: >0 means
// the air moves east (toward the right of the screen), <0 westward (left).
// windDir is the meteorological direction the wind comes FROM, in degrees
// (0=N, 90=E, 180=S, 270=W).
func windEastward(windDir float64) float64 {
	return -math.Sin(windDir * math.Pi / 180)
}

// stormArt picks the directional storm-cloud art for the current wind: the
// rain curtains lean with the wind — right when it blows east, left when it
// blows west. Dead-calm / unknown wind falls back to the right-leaning art.
func stormArt(windDir float64) []string {
	if windEastward(windDir) >= 0 {
		return cloudArtStormRight
	}
	return cloudArtStormLeft
}

// cloudArtStormRight: cumulonimbus with rain curtains leaning right (wind
// blowing eastward).
var cloudArtStormRight = []string{
	"         ___         ",
	"     .-(    )-.      ",
	"  .-(         )-.    ",
	".-'             '-.  ",
	"---------------------",
	`  \ \   \ \   \ \    `,
	`   \ \   \ \   \ \   `,
}

var cloudArtStormLeft = []string{
	"         ___         ",
	"     .-(    )-.      ",
	"  .-(         )-.    ",
	".-'             '-.  ",
	"---------------------",
	"  / /   / /   / /    ",
	"   / /   / /   / /   ",
}

// cloudArtCirrus: thin, wispy high clouds (haze / thin conditions).
var cloudArtCirrus = []string{
	"   .~~~.      ",
	" .'     '.    ",
	" '         '  ",
}

type puff struct {
	x, y  float64
	speed float64
	shade uint8
	art   []string
}

// Clouds animates drifting cloud blobs. Which art is used and how fast
// the clouds drift is driven by the weather: Storm renders tall dark
// cumulonimbus, Dense a flat overcast layer, Misty thin wispy cirrus,
// and the drift speed scales with the reported wind (m/s).
type Clouds struct {
	Dense    bool // fully overcast vs. scattered
	Misty    bool // low-contrast haze look
	Storm    bool // heavy dark storm clouds
	CloudsOn bool // draw the cloud puffs ('o' hides them, leaving a clear sky)
	wind     float64
	windDir  float64 // degrees, where the wind comes FROM
	w, h     int
	// topMargin reserves rows at the top of the screen (the status panel)
	// so clouds drift below it instead of hiding behind it.
	topMargin int
	puffs     []puff
	t         float64 // animation clock (fog drift)
	rng       *rand.Rand
}

func NewClouds(dense, misty, storm bool, wind, windDir float64) *Clouds {
	return &Clouds{Dense: dense, Misty: misty, Storm: storm, CloudsOn: true, wind: wind, windDir: windDir, rng: rand.New(rand.NewSource(2))}
}

func (c *Clouds) Resize(w, h int) {
	c.w, c.h = w, h
	n := w / 14
	if c.Dense {
		n = w / 8
	}
	if n < 2 {
		n = 2
	}
	c.puffs = make([]puff, n)
	for i := range c.puffs {
		c.puffs[i] = c.newPuff(true)
	}
}

// SetTopMargin reserves rows at the top of the screen (the status panel)
// so clouds drift below it and never hide behind it.
func (c *Clouds) SetTopMargin(rows int) {
	if rows < 0 {
		rows = 0
	}
	c.topMargin = rows
}

// SetClouds toggles the cloud puffs (the 'o' key hides them, leaving a
// clear sky).
func (c *Clouds) SetClouds(on bool) { c.CloudsOn = on }

func (c *Clouds) newPuff(randomX bool) puff {
	layerBack := c.rng.Float64() < 0.5
	p := puff{
		art: c.pickArt(),
	}
	maxY := c.h/2 - c.topMargin
	if maxY < 1 {
		maxY = 1
	}
	p.y = float64(c.topMargin) + float64(c.rng.Intn(maxY))
	scale := c.windScale()
	if layerBack {
		p.speed = (0.05 + c.rng.Float64()*0.05) * scale
		p.shade = c.shadeFor(245)
	} else {
		p.speed = (0.12 + c.rng.Float64()*0.10) * scale
		p.shade = c.shadeFor(252)
	}
	if randomX {
		p.x = c.rng.Float64() * float64(c.w+30)
	} else {
		p.x = -30
	}
	return p
}

// pickArt chooses the cloud art that matches the current weather mood.
func (c *Clouds) pickArt() []string {
	switch {
	case c.Storm:
		return stormArt(c.windDir)
	case c.Dense:
		return cloudArtStratus
	case c.Misty:
		return cloudArtCirrus
	default:
		return cloudArtCumulus
	}
}

// shadeFor darkens the clouds for stormy weather.
func (c *Clouds) shadeFor(base uint8) uint8 {
	switch {
	case c.Storm:
		return base - 12
	case c.Misty:
		return 238
	default:
		return base
	}
}

// windScale maps wind speed (m/s) to a drift multiplier.
func (c *Clouds) windScale() float64 {
	s := 1 + c.wind/10
	if s < 0.4 {
		s = 0.4
	}
	if s > 3 {
		s = 3
	}
	return s
}

func (c *Clouds) Tick() {
	if c.w == 0 {
		return
	}
	c.t += 0.15
	for i := range c.puffs {
		p := &c.puffs[i]
		p.x += p.speed
		if p.x > float64(c.w)+2 {
			*p = c.newPuff(false)
		}
	}
}

func (c *Clouds) Draw(buf *render.Buffer) {
	sky := uint8(252)
	switch {
	case c.Storm:
		sky = 237
	case c.Dense:
		sky = 246
	case c.Misty:
		sky = 250
	}
	buf.Clear(sky)

	if !c.CloudsOn {
		return // clouds hidden ('o' key) — just the sky
	}

	for _, p := range c.puffs {
		for row, line := range p.art {
			y := int(p.y) + row
			for i, ch := range line {
				if ch == ' ' {
					continue
				}
				x := int(p.x) + i
				buf.Set(x, y, ch, p.shade, false)
			}
		}
	}

	// Fog/mist: a slow-drifting ground bank across the lower third, denser
	// toward the floor — the actual "fog you walk through", not just the
	// wispy cirrus up top.
	if c.Misty {
		drawFogBank(buf, c.t, c.w, c.h)
	}
}

// drawFogBank paints a dense, slowly-drifting fog layer over the lower
// part of the scene. Density builds toward the bottom and undulates with
// time so it reads as rolling fog rather than a static band.
func drawFogBank(buf *render.Buffer, t float64, w, h int) {
	if w == 0 || h == 0 {
		return
	}
	start := h - h/3
	if start < 0 {
		start = 0
	}
	fogChars := []rune{'⠂', '⠃', '⠇', '⣿'}
	for y := start; y < h; y++ {
		depth := float64(y-start) / float64(h-start) // 0 at top of bank, 1 at floor
		for x := 0; x < w; x++ {
			n := math.Sin(float64(x)*0.35 + t*0.5 + float64(y)*0.8)
			dens := 0.35*depth + 0.25
			if n < dens {
				continue
			}
			idx := int((n+1)/2*float64(len(fogChars)-1) + 0.5)
			if idx >= len(fogChars) {
				idx = len(fogChars) - 1
			}
			buf.Set(x, y, fogChars[idx], 249, false)
		}
	}
}

// drawCloudBand tiles a single cloud art across the top of the screen
// (used by the rain/snow scenes for a solid cloud deck) and scrolls it
// with a wind-scaled speed.
func drawCloudBand(buf *render.Buffer, y int, art []string, w int, t float64, speed float64, shade uint8) {
	if len(art) == 0 || y >= buf.H {
		return
	}
	artW := len(art[0])
	if artW == 0 {
		return
	}
	start := -(int(t*speed) % artW)
	for x := start; x < w+artW; x += artW {
		for row, line := range art {
			yy := y + row
			if yy >= buf.H {
				continue
			}
			for i, ch := range line {
				if ch == ' ' {
					continue
				}
				buf.Set(x+i, yy, ch, shade, false)
			}
		}
	}
}
