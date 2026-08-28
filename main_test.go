package main

import (
	"testing"
	"time"
)

func TestPickStrPrecedence(t *testing.T) {
	cases := []struct {
		name          string
		cliSet        bool
		cli, env, cfg string
		want          string
	}{
		{"cli wins", true, "cli", "env", "cfg", "cli"},
		{"env beats cfg", false, "", "env", "cfg", "env"},
		{"cfg beats default", false, "", "", "cfg", "cfg"},
		{"default", false, "", "", "", "def"},
	}
	for _, c := range cases {
		if got := pickStr(c.cliSet, c.cli, c.env, c.cfg, "def"); got != c.want {
			t.Errorf("%s: pickStr = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPickBoolIntFloat(t *testing.T) {
	if pickBool(false, true, boolPtr(true), false) != true {
		t.Error("cfg bool should win over default")
	}
	if pickBool(true, false, boolPtr(true), false) != false {
		t.Error("cli bool should win")
	}
	if pickBool(false, false, nil, true) != true {
		t.Error("default bool")
	}
	if pickInt(false, 0, intPtr(42), 7) != 42 {
		t.Error("cfg int should win")
	}
	if pickFloat(false, 0, floatPtr(0.5), -1) != 0.5 {
		t.Error("cfg float should win")
	}
}

func TestPickDur(t *testing.T) {
	if pickDur(false, 0, "3h30m", time.Minute) != 3*time.Hour+30*time.Minute {
		t.Error("cfg duration should be parsed")
	}
	if pickDur(true, 90*time.Second, "3h", time.Minute) != 90*time.Second {
		t.Error("cli duration should win")
	}
	if pickDur(false, 0, "garbage", time.Minute) != time.Minute {
		t.Error("bad duration should fall back to default")
	}
}

func boolPtr(b bool) *bool        { return &b }
func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }
