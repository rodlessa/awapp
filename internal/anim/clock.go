package anim

import "time"

// clock is swappable so tests (and golden-file tests) can pin "now" to a
// known instant instead of whatever the wall clock says.
var clock = time.Now

// SetClock overrides the package clock. Pass nil to restore the real
// clock. Not safe to call concurrently with Draw; tests call it before
// rendering.
func SetClock(fn func() time.Time) {
	if fn == nil {
		clock = time.Now
		return
	}
	clock = fn
}
