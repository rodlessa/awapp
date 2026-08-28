package anim

import (
	"math"
	"math/rand"
	"strings"
	"time"

	"weatherterm/internal/render"
)

// Season is the (hemisphere-aware) time of year. It picks the ambient
// drifting-leaf effect and (in winter) whether snow feels snowier.
type Season int

const (
	SeasonSpring Season = iota
	SeasonSummer
	SeasonFall
	SeasonWinter
)

func (s Season) String() string {
	switch s {
	case SeasonSpring:
		return "spring"
	case SeasonSummer:
		return "summer"
	case SeasonFall:
		return "fall"
	default:
		return "winter"
	}
}

// SeasonFor maps a calendar month + latitude to a Season. lat > 0 is the
// northern hemisphere, lat < 0 the southern (so Fortaleza, -3.7, flips
// the months).
func SeasonFor(month time.Month, lat float64) Season {
	south := lat < 0
	switch month {
	case time.December, time.January, time.February:
		if south {
			return SeasonSummer
		}
		return SeasonWinter
	case time.March, time.April, time.May:
		if south {
			return SeasonFall
		}
		return SeasonSpring
	case time.June, time.July, time.August:
		if south {
			return SeasonWinter
		}
		return SeasonSummer
	default: // September, October, November
		if south {
			return SeasonSpring
		}
		return SeasonFall
	}
}

// brailleBlank is the empty braille cell; sprites use it as transparent
// padding.
const brailleBlank = '\u2800'

func leafBlank(r rune) bool { return r == ' ' || r == brailleBlank }

func leafRowBlank(row []rune) bool {
	for _, r := range row {
		if !leafBlank(r) {
			return false
		}
	}
	return true
}

// trimArt removes blank padding rows/columns from a sprite so the leaf
// sits tightly on screen. Cells that are a space or the empty braille
// cell count as blank.
func trimArt(rows []string) []string {
	grid := make([][]rune, len(rows))
	for i, s := range rows {
		grid[i] = []rune(s)
	}
	top := 0
	for top < len(grid) && leafRowBlank(grid[top]) {
		top++
	}
	bot := len(grid)
	for bot > top && leafRowBlank(grid[bot-1]) {
		bot--
	}
	grid = grid[top:bot]
	if len(grid) == 0 {
		return nil
	}
	minC, maxC := len(grid[0]), -1
	for _, row := range grid {
		for c, r := range row {
			if !leafBlank(r) {
				if c < minC {
					minC = c
				}
				if c > maxC {
					maxC = c
				}
			}
		}
	}
	if maxC < minC {
		return nil
	}
	out := make([]string, len(grid))
	for i, row := range grid {
		for len(row) <= maxC {
			row = append(row, ' ')
		}
		out[i] = string(row[minC : maxC+1])
	}
	return out
}

// The leaf art below is the user's braille leaf designs (U+2800 empty
// cells are padding). They are trimmed to their bounding box at init.

// leafSummerRaw: a fresh green summer leaf.
var leafSummerRaw = []string{
	"⠀⠀⠀⠀⠀⠀⠀⠀⣀⣠⣤⣤⣄⣀⣀⡀⠀⠀⠀",
	"⠀⠀⠀⠀⠀⢀⣶⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣶⠶",
	"⠀⠀⠀⠀⢠⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠃⠀",
	"⠀⠀⠀⢀⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠋⠀⠀⠀",
	"⢀⣠⠞⠋⠉⠛⠻⠿⣿⣿⣿⠿⠟⠋⠀⠀⠀⠀⠀",
	"⠞⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
}

// leafFallRaw: the three autumn leaves (yellow/orange, dry & curling).
var leafFallRaw = [][]string{
	{
		"⠀⠀⠀⠀⠀⠀⠀⣶⣄⠀⠀⢀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⢸⣿⣿⣷⣴⣿⡄⠀⠀⠀⠀⠀⢀⡀⠀",
		"⠀⠀⠀⠀⠰⣶⣾⣿⣿⣿⣿⣿⡇⠀⢠⣷⣤⣶⣿⡇⠀",
		"⠀⠀⠀⠀⠀⠙⣿⣿⣿⣿⣿⣿⣿⣀⣿⣿⣿⣿⣿⣧⣀",
		"⠀⠀⠀⣷⣦⣀⠘⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠃",
		"⢲⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠁⠀",
		"⠀⠙⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡟⠁⠀⠀",
		"⠀⠚⠻⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠿⠿⠂⠀⠀",
		"⠀⠀⠀⠀⠀⠉⠙⢻⣿⣿⡿⠛⠉⡇⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠘⠋⠁⠀⠀⠀⠸⡄⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢳⡀⠀⠀⠀⠀⠀",
	},
	{
		"⠀⠀⠀⠀⠀⠀⠀⠀⢠⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠐⢼⣡⡴⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⢀⠲⣶⢦⣼⠋⣀⣴⢀⠀⠀⡀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⢽⣴⣄⣹⡟⠋⣅⣯⢀⣀⠖⠀⠀⠀⠀",
		"⠀⠀⠢⣤⣤⣄⠌⠉⠻⣿⣼⡿⠇⣷⠷⡓⠀⠀⡄⠀",
		"⠀⠀⠀⠁⠈⠻⠷⢼⣼⣯⣾⣄⣷⣓⣉⡀⣰⡿⠡⠀",
		"⠀⢵⣢⣤⢠⢀⡀⠀⡌⣿⡻⠛⢣⢸⡀⣗⠿⠔⠂⠀",
		"⠀⠀⠀⠭⣾⢿⣵⣼⣴⣿⠀⣄⣸⣆⣿⣟⡉⡉⣠⠂",
		"⠀⠀⠀⠀⠁⢪⠽⡛⠉⠚⣷⣿⠷⠿⢷⢦⣼⠷⡛⠀",
		"⠤⣤⣤⣢⣱⡸⣄⢥⢻⣴⣼⣇⢴⢼⣾⣼⡽⠥⠀⠀",
		"⠀⣀⣜⣕⣯⢟⣷⢿⢏⢉⡷⣿⠟⢻⢿⣹⢲⣾⡖⠂",
		"⠀⠀⠘⠟⢛⣞⣿⡿⠿⢾⣿⢣⢦⣾⣾⣿⡛⠁⠀⠀",
		"⠀⠀⠀⠀⠁⠉⠘⠀⠀⠀⢙⠾⠛⠛⠣⠿⠖⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡆⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠐⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀",
	},
	{
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⠀⣸⣷",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡀⢰⡄⣾⣿⢿⠃",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⡄⢠⣆⠀⣴⣼⣷⣾⣷⣜⣛⡟⠋",
		"⠀⠀⠀⣤⣄⠀⠀⢀⡄⠀⣾⣗⣀⣿⡏⣼⣿⣿⣿⡓⣿⣧⠿⠿⠿⠟⠀",
		"⠀⠀⢰⣿⡇⠀⢀⣿⡧⠔⣿⣯⣮⣿⡥⣿⣿⢶⣿⣧⠞⢷⣿⣾⣷⡶⠀",
		"⠀⠀⢸⡿⡀⠀⢿⣿⡃⣺⣿⡿⢻⣿⣔⣺⣿⡿⠟⣿⣿⣷⣶⣶⡷⠀⠀",
		"⠀⠀⢹⣿⡂⠀⢵⣿⠁⢬⣿⣏⢚⣿⣿⡾⣿⣿⣶⣤⣭⣈⡁⠁⠀⠀⠀",
		"⠀⠀⠀⣿⢅⠀⢹⣿⢅⣼⣿⣿⣏⠙⣿⣷⣷⣏⠟⠟⠻⠛⠿⠿⠖⠀⠀",
		"⠀⠀⠀⠘⣿⣠⠾⢿⣿⣶⡾⠿⣿⣿⣦⣯⣛⠿⣿⣷⣶⣦⣀⠀⠀⠀⠀",
		"⢠⣤⠴⠞⢿⣷⣄⠹⠿⢿⣿⣦⣌⠈⠛⠿⣷⣷⣦⣌⠉⠙⠙⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠈⠿⣿⣷⣦⣀⠈⠛⢿⣷⣦⣀⠀⠀⠉⠉⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠙⠻⢿⣷⣶⣦⣌⡙⠻⠆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
		"⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠈⠈⠙⠛⠛⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀",
	},
}

// Precomputed trimmed sprites (trimmed lazily via sync.Once-like init).
var (
	leafSummer = trimArt(leafSummerRaw)
	leafFall   = [][]string{trimArt(leafFallRaw[0]), trimArt(leafFallRaw[1]), trimArt(leafFallRaw[2])}
)

// leafScale is how "close" a leaf is: large leaves drift in the
// foreground at full size, small ones sit far back for a sense of depth.
type leafScale int

const (
	scaleLarge  leafScale = iota // full-size foreground (~5%)
	scaleMedium                  // half-size mid-ground (~25%)
	scaleSmall                   // third-size background (~70%)
)

// leafSizes holds the same sprite at three depths.
type leafSizes struct {
	large, medium, small []string
}

func (s leafSizes) at(sc leafScale) []string {
	switch sc {
	case scaleMedium:
		return s.medium
	case scaleSmall:
		return s.small
	default:
		return s.large
	}
}

// brailleDots counts the lit dots in a braille cell (0 for a blank,
// 1 for a non-braille glyph).
func brailleDots(r rune) int {
	if r >= 0x2801 && r <= 0x28FF {
		b := uint8(r - 0x2800)
		n := 0
		for b != 0 {
			n += int(b & 1)
			b >>= 1
		}
		return n
	}
	if leafBlank(r) {
		return 0
	}
	return 1
}

// blockDarkest returns the darkest non-blank cell in a k×k block of the
// sprite grid, or a space if the whole block is empty.
func blockDarkest(grid [][]rune, x0, y0, k int) rune {
	best, bestDots := rune(' '), 0
	for y := y0; y < y0+k && y < len(grid); y++ {
		for x := x0; x < x0+k && x < len(grid[y]); x++ {
			if d := brailleDots(grid[y][x]); d > bestDots {
				best, bestDots = grid[y][x], d
			}
		}
	}
	return best
}

// scaleSprite downsamples a braille sprite by factor k (k=1 returns it
// unchanged), keeping the darkest braille cell of each k×k block so the
// leaf shape survives at a smaller size.
func scaleSprite(art []string, k int) []string {
	if k <= 1 || len(art) == 0 {
		return art
	}
	grid := make([][]rune, len(art))
	w := 0
	for i, s := range art {
		grid[i] = []rune(s)
		if len(grid[i]) > w {
			w = len(grid[i])
		}
	}
	var out []string
	for y := 0; y < len(grid); y += k {
		var row []rune
		for x := 0; x < w; x += k {
			row = append(row, blockDarkest(grid, x, y, k))
		}
		out = append(out, strings.TrimRight(string(row), " "))
	}
	return out
}

// Sized variants of each sprite (large/medium/small) for depth.
var (
	leafSummerSizes = leafSizes{large: leafSummer, medium: scaleSprite(leafSummer, 2), small: scaleSprite(leafSummer, 3)}
	leafFallSizes   = []leafSizes{
		{large: leafFall[0], medium: scaleSprite(leafFall[0], 2), small: scaleSprite(leafFall[0], 3)},
		{large: leafFall[1], medium: scaleSprite(leafFall[1], 2), small: scaleSprite(leafFall[1], 3)},
		{large: leafFall[2], medium: scaleSprite(leafFall[2], 2), small: scaleSprite(leafFall[2], 3)},
	}
)

// flakeSizes makes a leafSizes where every depth is the same single-cell
// snowflake glyph (snowflakes are all small).
func flakeSizes(g rune) leafSizes {
	s := []string{string(g)}
	return leafSizes{large: s, medium: s, small: s}
}

// winterFlakeSizes: the ambient winter snowflake sprites (the 'l' toggle
// hides them like the leaves).
var winterFlakeSizes = []leafSizes{
	flakeSizes('❄'),
	flakeSizes('*'),
	flakeSizes('.'),
	flakeSizes('·'),
}

// leaf is a single drifting leaf on screen.
type leaf struct {
	x, y    float64
	speed   float64 // vertical fall speed (cells/tick)
	wind    float64 // horizontal wind drift
	sway    float64 // flutter phase
	swaySpd float64 // flutter speed
	art     []string
	scale   leafScale // depth: large/medium/small
	shade   uint8
}

// Leaves is an ambient layer that drifts leaf sprites across the scene,
// driven by the season (which art + how many) and the wind (direction +
// speed push the leaves sideways).
type Leaves struct {
	Season  Season
	On      bool // master on/off ('l' key); default on
	wind    float64
	windDir float64 // degrees, where the wind comes FROM
	w, h    int
	t       float64 // animation clock
	leaves  []leaf
	rng     *rand.Rand
}

// NewLeaves creates an empty leaf layer. Start it in winter (no leaves)
// until a weather report sets the real season.
func NewLeaves() *Leaves {
	return &Leaves{Season: SeasonWinter, On: true, rng: rand.New(rand.NewSource(7))}
}

// SetOn enables/disables the whole leaf layer (the 'l' key).
func (l *Leaves) SetOn(on bool) { l.On = on }

// SetSeason switches the leaf art and density. Changing it reshuffles
// the drifting leaves.
func (l *Leaves) SetSeason(s Season) {
	if l.Season == s {
		return
	}
	l.Season = s
	l.repopulate()
}

// SetWind stores the reported wind so leaves lean with its direction.
func (l *Leaves) SetWind(wind, windDir float64) {
	l.wind = wind
	l.windDir = windDir
}

func (l *Leaves) Resize(w, h int) {
	l.w, l.h = w, h
	l.repopulate()
}

// countFor decides how many leaves to keep on screen for the season. The
// totals are generous so the ~5% foreground leaves have background company.
func (l *Leaves) countFor() int {
	switch l.Season {
	case SeasonFall:
		return 10 // big detailed sprites — most are downscaled for depth
	case SeasonSummer, SeasonSpring:
		return 18
	default: // winter — ambient snowflakes instead of leaves
		n := l.w / 3
		if n < 10 {
			n = 10
		}
		if n > 60 {
			n = 60
		}
		return n
	}
}

func (l *Leaves) arts() []leafSizes {
	switch l.Season {
	case SeasonFall:
		return leafFallSizes
	case SeasonWinter:
		return winterFlakeSizes
	default: // spring & summer use the same green leaf
		return []leafSizes{leafSummerSizes}
	}
}

// pickScale decides how close a new leaf is: 5% full-size foreground,
// 25% medium, the rest small background.
func (l *Leaves) pickScale() leafScale {
	switch r := l.rng.Float64(); {
	case r < 0.05:
		return scaleLarge
	case r < 0.30:
		return scaleMedium
	default:
		return scaleSmall
	}
}

// leafShade picks a colour for the season, dimmed the further back the
// leaf sits (monochrome mode ignores these anyway).
func (l *Leaves) leafShade(sc leafScale) uint8 {
	var base uint8
	switch l.Season {
	case SeasonFall:
		base = []uint8{178, 208, 214, 202}[l.rng.Intn(4)] // yellow → orange → red
	case SeasonWinter:
		base = []uint8{255, 253, 251, 189}[l.rng.Intn(4)] // whites + icy blue
	default:
		base = []uint8{34, 40, 71, 114}[l.rng.Intn(4)] // greens
	}
	switch sc {
	case scaleMedium:
		return base - 8
	case scaleSmall:
		return base - 16
	default:
		return base
	}
}

func (l *Leaves) newLeaf(randomY bool, sizes leafSizes) leaf {
	sc := l.pickScale()
	art := sizes.at(sc)
	speed, swaySpd := 0.12+l.rng.Float64()*0.2, 0.4+l.rng.Float64()*0.5
	if l.Season == SeasonWinter {
		speed, swaySpd = 0.04+l.rng.Float64()*0.10, 0.3+l.rng.Float64()*0.4 // flakes fall slow & flutter gently
	}
	lf := leaf{
		x:       l.rng.Float64() * float64(l.w),
		speed:   speed,
		wind:    windEastward(l.windDir) * (0.2 + math.Abs(l.wind)*0.05),
		sway:    l.rng.Float64() * math.Pi * 2,
		swaySpd: swaySpd,
		art:     art,
		scale:   sc,
		shade:   l.leafShade(sc),
	}
	if randomY {
		lf.y = l.rng.Float64() * float64(l.h)
	} else {
		lf.y = -float64(len(art)) // start just above the top edge
	}
	return lf
}

func (l *Leaves) repopulate() {
	if l.w == 0 || l.h == 0 {
		return
	}
	n := l.countFor()
	if n == 0 {
		l.leaves = nil
		return
	}
	sizes := l.arts()
	ls := make([]leaf, n)
	for i := range ls {
		ls[i] = l.newLeaf(true, sizes[l.rng.Intn(len(sizes))])
	}
	l.leaves = ls
}

// Tick advances the drifting leaves one frame; call once per animation
// tick before Draw.
func (l *Leaves) Tick() {
	l.t += 0.15
	if !l.On || len(l.leaves) == 0 || l.w == 0 {
		return
	}
	sizes := l.arts()
	for i := range l.leaves {
		lf := &l.leaves[i]
		lf.y += lf.speed
		// sideways: wind drift + a gentle flutter, always moving down-wind
		lf.x += lf.wind + math.Sin(l.t*lf.swaySpd+lf.sway)*0.4
		if lf.y > float64(l.h) || lf.x < -60 || lf.x > float64(l.w)+60 {
			*lf = l.newLeaf(false, sizes[l.rng.Intn(len(sizes))])
			lf.x = l.rng.Float64() * float64(l.w)
		}
	}
}

// Draw paints the leaf layer over the scene (spaces and empty braille
// cells are transparent; the buffer clips off-screen). Does nothing when
// leaves are toggled off.
func (l *Leaves) Draw(buf *render.Buffer) {
	if !l.On {
		return
	}
	for _, lf := range l.leaves {
		x, y := int(lf.x), int(lf.y)
		for row, line := range lf.art {
			for col, r := range line {
				if leafBlank(r) {
					continue
				}
				buf.Set(x+col, y+row, r, lf.shade, false)
			}
		}
	}
}
