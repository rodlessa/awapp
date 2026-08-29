package app

import "testing"

// TestNextModeTransitions covers the whole state machine (loading → live /
// offline / manual) with deterministic inputs — no terminal, no clock.
func TestNextModeTransitions(t *testing.T) {
	cases := []struct {
		name string
		cur  mode
		ev   appEvent
		want mode
	}{
		// loading: first successful report → live
		{"loading+report→live", modeLoading, evReport, modeLive},
		// loading: first failure → offline picker
		{"loading+error→offline", modeLoading, evError, modeOffline},
		// loading: user can also pick a condition manually before any report
		{"loading+manual→manual", modeLoading, evManual, modeManual},

		// live: a refresh keeps it live; a failure keeps the last good frame;
		// manual picks are ignored while a live report is shown
		{"live+report→live", modeLive, evReport, modeLive},
		{"live+error→live", modeLive, evError, modeLive},
		{"live+manual→live", modeLive, evManual, modeLive},

		// offline: a report recovers to live; an error keeps it offline; a
		// manual pick switches to manual
		{"offline+report→live", modeOffline, evReport, modeLive},
		{"offline+error→offline", modeOffline, evError, modeOffline},
		{"offline+manual→manual", modeOffline, evManual, modeManual},

		// manual: a report recovers to live; an error keeps the manual pick;
		// another manual pick stays manual
		{"manual+report→live", modeManual, evReport, modeLive},
		{"manual+error→manual", modeManual, evError, modeManual},
		{"manual+manual→manual", modeManual, evManual, modeManual},
	}
	for _, c := range cases {
		if got := nextMode(c.cur, c.ev); got != c.want {
			t.Errorf("%s: nextMode(%v,%v) = %v, want %v", c.name, c.cur, c.ev, got, c.want)
		}
	}
}
