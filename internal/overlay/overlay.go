// Package overlay draws the "S" hotkey status panel on top of the
// current animation frame: city, temperature (C or F), current
// condition, and the Moon state during a clear night. It uses plain
// ASCII only so it renders identically with or without colors.
package overlay

import (
	"fmt"
	"time"
	"unicode/utf8"

	"awapp/internal/render"
)

type Info struct {
	City       string
	Desc       string // e.g. "light rain", or "manual selection" offline
	TempC      float64
	TempF      float64
	Fahrenheit bool
	HasTemp    bool    // a real temperature reading is available
	WindMS     float64 // wind speed, m/s (0 = unknown)
	Live       bool    // false when rendering a user-picked condition (no comms)
	UpdatedAt  time.Time
	HasData    bool   // false before the first successful fetch
	Moon       string // optional: current Moon description (clear night)
	Sun        string // optional: solar-eclipse description (clear day)
	Stars      string // optional: star-field / light-pollution description (clear night)
	Err        string // optional: why the last weather fetch failed (offline picker)
	Season     string // optional: current season ("summer", "winter", ...)
	LeavesOn   bool   // whether the seasonal leaf/snow layer is enabled
}

// PanelHeight returns the number of rows the status panel would occupy
// for the given info (border included). The app uses it to keep the
// Sun/Moon arc below the panel so they never hide behind it.
func PanelHeight(info Info) int {
	return len(panelLines(info)) + 2
}

// panelLines builds the text rows of the status panel.
func panelLines(info Info) []string {
	lines := []string{}
	if info.HasData {
		if info.HasTemp {
			temp := info.TempC
			unit := "C"
			if info.Fahrenheit {
				temp = info.TempF
				unit = "F"
			}
			lines = append(lines,
				" "+info.City,
				fmt.Sprintf(" %.1f %s - %s", temp, unit, info.Desc),
			)
		} else {
			lines = append(lines,
				" "+info.City,
				" "+info.Desc,
			)
		}
	} else {
		lines = append(lines, " No data yet")
	}
	if info.HasData && info.WindMS > 0 {
		lines = append(lines, fmt.Sprintf(" wind %.1f m/s", info.WindMS))
	}
	if info.Live {
		lines = append(lines, " [live] updates every 5 min")
	} else {
		lines = append(lines, " [offline] manual selection")
	}
	if info.Err != "" {
		lines = append(lines, " ! "+info.Err)
	}
	// Always show the current unit so 'u' visibly toggles C <-> F.
	unit := "C"
	if info.Fahrenheit {
		unit = "F"
	}
	lines = append(lines, " units: "+unit+" (press u)")
	if info.Moon != "" {
		lines = append(lines, " "+info.Moon)
	}
	if info.Sun != "" {
		lines = append(lines, " "+info.Sun)
	}
	if info.Stars != "" {
		lines = append(lines, " "+info.Stars)
	}
	if info.Season != "" {
		state := "off"
		if info.LeavesOn {
			state = "on"
		}
		lines = append(lines, " Season: "+info.Season+"  leaves "+state+" (l)")
	}
	lines = append(lines, " [u] unit  [c] color  [t] stars  [o] clouds  [l] leaves  [+]/[-] size  [m] moon  [e] lunar  [x] solar  [s] hide  [q] quit")
	return lines
}

// Draw paints a bordered panel in the top-left corner. w/h are the
// full buffer dimensions so the panel can clamp itself on tiny
// terminals instead of overflowing.
func Draw(buf *render.Buffer, info Info) {
	lines := panelLines(info)

	panelW := 0
	for _, l := range lines {
		// Measure in runes, not bytes, so multi-byte UTF-8 (°, ●, ⠀ …)
		// doesn't inflate the width or get split mid-rune.
		if n := utf8.RuneCountInString(l) + 2; n > panelW {
			panelW = n
		}
	}
	if panelW > buf.W-2 {
		panelW = buf.W - 2
	}
	panelH := len(lines) + 2
	if panelW < 4 || panelH < 3 || buf.W < 6 || buf.H < 5 {
		return // terminal too small to render the overlay legibly
	}

	x, y := 1, 1
	fg := uint8(255)

	buf.FillBox(x, y, panelW, panelH, fg)
	// Plain ASCII border.
	for i := 0; i < panelW; i++ {
		buf.Set(x+i, y, '-', fg, false)
		buf.Set(x+i, y+panelH-1, '-', fg, false)
	}
	for j := 0; j < panelH; j++ {
		buf.Set(x, y+j, '|', fg, false)
		buf.Set(x+panelW-1, y+j, '|', fg, false)
	}
	buf.Set(x, y, '+', fg, false)
	buf.Set(x+panelW-1, y, '+', fg, false)
	buf.Set(x, y+panelH-1, '+', fg, false)
	buf.Set(x+panelW-1, y+panelH-1, '+', fg, false)

	for i, l := range lines {
		// Truncate on rune boundaries so a multi-byte character is never
		// cut in half (which would render as a U+FFFD replacement glyph).
		if rl := []rune(l); len(rl) > panelW-2 {
			l = string(rl[:panelW-2])
		}
		buf.SetString(x+1, y+1+i, l, fg, false)
	}
}
