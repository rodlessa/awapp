package main

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestEnvOr(t *testing.T) {
	t.Setenv("AWAPP_TEST_ENV", "hello")
	if got := envOr("AWAPP_TEST_ENV", "def"); got != "hello" {
		t.Errorf("envOr should return the env value, got %q", got)
	}
	if got := envOr("AWAPP_TEST_MISSING", "def"); got != "def" {
		t.Errorf("envOr should return the default when unset, got %q", got)
	}
}

func TestStrOr(t *testing.T) {
	s := "x"
	if got := strOr(&s); got != "x" {
		t.Errorf("strOr(ptr) = %q, want %q", got, "x")
	}
	if got := strOr(nil); got != "" {
		t.Errorf("strOr(nil) = %q, want empty", got)
	}
}

func TestLoadFileConfig(t *testing.T) {
	dir := t.TempDir()
	// Missing file -> zero fileConfig, no error.
	if fc := loadFileConfig(filepath.Join(dir, "nope.json")); fc.APIKey != nil {
		t.Error("missing config should return an empty fileConfig")
	}
	// Valid JSON is parsed.
	good := filepath.Join(dir, "good.json")
	if err := os.WriteFile(good, []byte(`{"api_key":"k","city":"Berlin"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fc := loadFileConfig(good)
	if fc.APIKey == nil || *fc.APIKey != "k" || fc.City == nil || *fc.City != "Berlin" {
		t.Errorf("valid config not parsed: %+v", fc)
	}
	// Malformed JSON -> falls back to a zero config (and prints to stderr,
	// which we don't assert here) rather than panicking or corrupting state.
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"api_key": `), 0o600); err != nil {
		t.Fatal(err)
	}
	if fc := loadFileConfig(bad); fc.APIKey != nil {
		t.Error("malformed config should fall back to a zero config")
	}
}

func TestConfigFileTooOpen(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on Windows")
	}
	dir := t.TempDir()
	if configFileTooOpen(filepath.Join(dir, "missing.json")) {
		t.Error("missing file must not warn")
	}
	open := filepath.Join(dir, "open.json")
	if err := os.WriteFile(open, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !configFileTooOpen(open) {
		t.Error("0644 config should be flagged as too open")
	}
	locked := filepath.Join(dir, "locked.json")
	if err := os.WriteFile(locked, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if configFileTooOpen(locked) {
		t.Error("0600 config must not be flagged")
	}
}

func boolPtr(b bool) *bool        { return &b }
func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }

func captureStderr(fn func()) string {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		return ""
	}
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	var buf strings.Builder
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestLoadFileConfigWarnsOnUnknownKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(p, []byte(`{"api_key":"k","cityy":"oops"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stderr := captureStderr(func() {
		fc := loadFileConfig(p)
		if fc.APIKey == nil || *fc.APIKey != "k" {
			t.Errorf("known key should still parse: %+v", fc)
		}
	})
	if !strings.Contains(stderr, "cityy") {
		t.Errorf("expected an unknown-key warning mentioning cityy, got stderr=%q", stderr)
	}
}

func TestWarnConfigFlagsBadValues(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(p, []byte(`{"size":99,"stars":"banana","moon":"sometimes"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fc := loadFileConfig(p)
	stderr := captureStderr(func() { warnConfig(fc, p) })
	for _, want := range []string{"size", "stars", "moon"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("expected a warning mentioning %q, got stderr=%q", want, stderr)
		}
	}
}

func TestWarnConfigSilentWhenValid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(p, []byte(`{"size":15,"stars":"light","moon":"auto"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fc := loadFileConfig(p)
	if stderr := captureStderr(func() { warnConfig(fc, p) }); stderr != "" {
		t.Errorf("valid config should warn nothing, got stderr=%q", stderr)
	}
}
