// awapp is a dependency-free terminal weather visualizer for
// Linux. It polls a weather source every 5 minutes and renders the
// current condition as a full-screen animation. Weather can come from
// OpenWeatherMap (API key, optional) or from your IP location via
// ipinfo + Open-Meteo (no key needed).
//
// Configuration precedence: command-line flags > environment variables >
// config file (~/.config/awapp/config.json) > defaults. A config
// file means you never have to export your API key.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"awapp/internal/app"
)

// version is stamped at build time via
//
//	-ldflags "-X main.version=<tag>"
//
// and defaults to the current release for local builds. `awapp -version`
// prints it.
var version = "1.0.5"

// fileConfig mirrors the JSON config file. Pointer fields distinguish
// "not set" from an explicit false/zero value.
type fileConfig struct {
	APIKey               *string  `json:"api_key"`
	City                 *string  `json:"city"`
	Provider             *string  `json:"provider"`
	Interval             *string  `json:"interval"`
	FPS                  *int     `json:"fps"`
	Units                *string  `json:"units"`
	Color                *bool    `json:"color"`
	Stars                *string  `json:"stars"`
	LightKey             *string  `json:"light_key"`
	Moon                 *string  `json:"moon"`
	Phase                *float64 `json:"phase"`
	Eclipse              *bool    `json:"eclipse"`
	EclipseDuration      *string  `json:"eclipse_duration"`
	SolarEclipse         *bool    `json:"solar_eclipse"`
	SolarEclipseDuration *string  `json:"solar_eclipse_duration"`
	UseIP                *bool    `json:"use_ip"`
	Size                 *float64 `json:"size"`
	Season               *string  `json:"season"`
	Leaves               *bool    `json:"leaves"`
	Theme                *string  `json:"theme"`
}

// knownConfigKeys is the set of accepted JSON keys, used to warn about
// typos instead of silently ignoring them.
var knownConfigKeys = map[string]bool{
	"api_key": true, "city": true, "provider": true, "interval": true,
	"fps": true, "units": true,
	"color": true, "stars": true, "light_key": true, "moon": true, "phase": true,
	"eclipse": true, "eclipse_duration": true, "solar_eclipse": true,
	"solar_eclipse_duration": true, "use_ip": true, "size": true, "season": true,
	"leaves": true, "theme": true,
}

// warnConfig flags values that are almost certainly mistakes. The app
// falls back to defaults for them anyway, but a warning beats a
// mysteriously ignored setting.
func warnConfig(fc fileConfig, path string) {
	if fc.Size != nil && (*fc.Size < 4 || *fc.Size > 60) {
		fmt.Fprintf(os.Stderr, "awapp: warning: config \"size\" = %v is out of range (4..60); using 15\n", *fc.Size)
	}
	if fc.Stars != nil && !oneOf(*fc.Stars, "light", "full", "off") {
		fmt.Fprintf(os.Stderr, "awapp: warning: config \"stars\" = %q is invalid (want light, full, or off)\n", *fc.Stars)
	}
	if fc.Moon != nil && !oneOf(*fc.Moon, "auto", "on", "off") {
		fmt.Fprintf(os.Stderr, "awapp: warning: config \"moon\" = %q is invalid (want auto, on, or off)\n", *fc.Moon)
	}
	if fc.Provider != nil && !oneOf(*fc.Provider, "auto", "openweather", "open-meteo", "weatherapi", "tomorrowio") {
		fmt.Fprintf(os.Stderr, "awapp: warning: config \"provider\" = %q is invalid (want auto, openweather, open-meteo, weatherapi, or tomorrowio)\n", *fc.Provider)
	}
	if fc.Theme != nil && !oneOf(*fc.Theme, "default", "sunset", "ocean", "forest") {
		fmt.Fprintf(os.Stderr, "awapp: warning: config \"theme\" = %q is invalid (want default, sunset, ocean, or forest)\n", *fc.Theme)
	}
}

func oneOf(v string, opts ...string) bool {
	for _, o := range opts {
		if v == o {
			return true
		}
	}
	return false
}

// saveToggles merges the final runtime toggles into the config file,
// preserving any existing keys. Writes are mode 0600 because the file may
// hold a plaintext API key.
func saveToggles(path string, f app.FinalSettings) error {
	if path == "" {
		return nil
	}
	fc := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &fc)
	}
	fc["color"] = f.Color
	if f.Fahrenheit {
		fc["units"] = "f"
	} else {
		fc["units"] = "c"
	}
	fc["stars"] = f.Stars
	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func defaultConfigPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "awapp", "config.json")
}

func loadFileConfig(path string) fileConfig {
	var fc fileConfig
	if path == "" {
		return fc
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fc // missing file is fine; we fall back to defaults
	}
	// Unknown keys are typos waiting to happen; call them out instead of
	// silently ignoring them.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		for k := range raw {
			if !knownConfigKeys[k] {
				fmt.Fprintln(os.Stderr, "awapp: warning: unknown config key \""+k+"\" in "+path)
			}
		}
	}
	if err := json.Unmarshal(data, &fc); err != nil {
		fmt.Fprintln(os.Stderr, "awapp: bad config file "+path+":", err)
	}
	return fc
}

// configFileTooOpen reports whether the config file is readable by other
// users — it may hold a plaintext API key, so the app warns about it.
// Windows has no owner/group/other permission bits (files always report
// 0666), so the check only applies on Unix-likes.
func configFileTooOpen(path string) bool {
	if runtime.GOOS == "windows" {
		return false
	}
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0o077 != 0
}

func strOr(p *string) string {
	if p != nil {
		return *p
	}
	return ""
}

// --- precedence helpers: CLI (if given) > env > config file > default ---

func pickStr(cliSet bool, cli, env, cfg, def string) string {
	switch {
	case cliSet:
		return cli
	case env != "":
		return env
	case cfg != "":
		return cfg
	default:
		return def
	}
}

func pickBool(cliSet bool, cli bool, cfg *bool, def bool) bool {
	if cliSet {
		return cli
	}
	if cfg != nil {
		return *cfg
	}
	return def
}

func pickInt(cliSet bool, cli int, cfg *int, def int) int {
	if cliSet {
		return cli
	}
	if cfg != nil {
		return *cfg
	}
	return def
}

func pickFloat(cliSet bool, cli float64, cfg *float64, def float64) float64 {
	if cliSet {
		return cli
	}
	if cfg != nil {
		return *cfg
	}
	return def
}

func pickDur(cliSet bool, cli time.Duration, cfg string, def time.Duration) time.Duration {
	if cliSet {
		return cli
	}
	if cfg != "" {
		if d, err := time.ParseDuration(cfg); err == nil {
			return d
		}
	}
	return def
}

// srcKind names where a resolved setting came from.
type srcKind string

const (
	srcFlag    srcKind = "flag"
	srcEnv     srcKind = "env"
	srcFile    srcKind = "file"
	srcDefault srcKind = "default"
)

// resolved is one line of `-list-config` output.
type resolved struct {
	name   string
	value  string
	source srcKind
}

// resolver resolves settings while recording their winning source, so
// `-list-config` can show exactly where every value came from.
type resolver struct {
	list []resolved
}

func (r *resolver) add(name, value string, source srcKind) {
	r.list = append(r.list, resolved{name, value, source})
}

func (r *resolver) str(name string, cliSet bool, cli, env, file, def string) string {
	v, s := def, srcDefault
	switch {
	case cliSet:
		v, s = cli, srcFlag
	case env != "":
		v, s = env, srcEnv
	case file != "":
		v, s = file, srcFile
	}
	r.add(name, v, s)
	return v
}

func (r *resolver) dur(name string, cliSet bool, cli time.Duration, file string, def time.Duration) time.Duration {
	if cliSet {
		r.add(name, cli.String(), srcFlag)
		return cli
	}
	if d, err := time.ParseDuration(file); err == nil && file != "" {
		r.add(name, d.String(), srcFile)
		return d
	}
	r.add(name, def.String(), srcDefault)
	return def
}

func (r *resolver) int(name string, cliSet bool, cli int, file *int, def int) int {
	if cliSet {
		r.add(name, strconv.Itoa(cli), srcFlag)
		return cli
	}
	if file != nil {
		r.add(name, strconv.Itoa(*file), srcFile)
		return *file
	}
	r.add(name, strconv.Itoa(def), srcDefault)
	return def
}

func (r *resolver) flt(name string, cliSet bool, cli float64, file *float64, def float64) float64 {
	if cliSet {
		r.add(name, strconv.FormatFloat(cli, 'g', -1, 64), srcFlag)
		return cli
	}
	if file != nil {
		r.add(name, strconv.FormatFloat(*file, 'g', -1, 64), srcFile)
		return *file
	}
	r.add(name, strconv.FormatFloat(def, 'g', -1, 64), srcDefault)
	return def
}

func (r *resolver) boolean(name string, cliSet bool, cli bool, file *bool, def bool) bool {
	if cliSet {
		r.add(name, strconv.FormatBool(cli), srcFlag)
		return cli
	}
	if file != nil {
		r.add(name, strconv.FormatBool(*file), srcFile)
		return *file
	}
	r.add(name, strconv.FormatBool(def), srcDefault)
	return def
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to a config.json (default: ~/.config/awapp/config.json)")
	apiKey := flag.String("apikey", "", "OpenWeatherMap API key (or config file / OPENWEATHERMAP_API_KEY)")
	city := flag.String("city", "", `City to fetch, e.g. "Fortaleza,BR"; no API key needed (or config file / env; else geolocated)`)
	provider := flag.String("provider", "", "weather provider: auto, openweather, open-meteo, weatherapi, tomorrowio")
	useIP := flag.Bool("use-ip", false, "use IP-location weather (ipinfo + Open-Meteo) instead of an API key")
	noIPPrompt := flag.Bool("no-ip-prompt", false, "skip the interactive IP-location prompt (for scripted/systemd use)")
	interval := flag.Duration("interval", 0, "how often to poll the weather API")
	fps := flag.Int("fps", 0, "animation frame rate")
	fahrenheit := flag.Bool("f", false, "start with Fahrenheit display")
	color := flag.Bool("color", false, "enable 256-color output (default: monochrome)")
	stars := flag.String("stars", "", "star field: light, full, off")
	lightKey := flag.String("light-key", "", "lightpollutionmap.info key for exact light-pollution readings")
	moonMode := flag.String("moon", "", "moon visibility: auto, on, off")
	eclipse := flag.Bool("eclipse", false, "start a simulated lunar eclipse")
	eclipseDur := flag.Duration("eclipse-duration", 0, "length of a simulated lunar eclipse")
	solarEclipse := flag.Bool("solar-eclipse", false, "start a simulated solar eclipse")
	solarDur := flag.Duration("solar-eclipse-duration", 0, "length of a simulated solar eclipse")
	phase := flag.Float64("phase", 0, "override moon phase 0..1 (default: compute from date)")
	size := flag.Float64("size", 0, "Sun/Moon diameter as %% of terminal width (default 15; adjust with '+'/'-')")
	season := flag.String("season", "", "leaf season: auto, spring, summer, fall, winter (default: auto from date + location)")
	theme := flag.String("theme", "", "color theme: default, sunset, ocean, forest")
	leaves := flag.Bool("leaves", false, "enable the seasonal leaf/snow layer (default on; toggle with 'l')")
	versionFlag := flag.Bool("version", false, "print version and exit")
	listConfig := flag.Bool("list-config", false, "print the resolved configuration (and its source) and exit")
	saveConfig := flag.Bool("save-config", false, "on quit, write the last-used color/units/stars toggles back to the config file")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("awapp %s\n", version)
		return
	}

	if configPath == "" {
		configPath = defaultConfigPath()
	}
	fc := loadFileConfig(configPath)
	if configFileTooOpen(configPath) {
		fmt.Fprintln(os.Stderr, "awapp: warning: "+configPath+" is readable by other users (it may hold an API key). Fix with: chmod 600 "+configPath)
	}
	warnConfig(fc, configPath)

	// Which flags were explicitly given on the command line.
	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	var unitsF *bool
	if fc.Units != nil {
		b := strings.EqualFold(*fc.Units, "f")
		unitsF = &b
	}

	res := &resolver{}
	cfg := app.Config{
		City:                 res.str("city", set["city"], *city, envOr("OPENWEATHERMAP_CITY", envOr("OPENWEATHER_CITY", "")), strOr(fc.City), ""),
		APIKey:               res.str("apikey", set["apikey"], *apiKey, envOr("OPENWEATHERMAP_API_KEY", envOr("OPENWEATHER_API_KEY", "")), strOr(fc.APIKey), ""),
		Provider:             res.str("provider", set["provider"], *provider, "", strOr(fc.Provider), "auto"),
		Interval:             res.dur("interval", set["interval"], *interval, strOr(fc.Interval), 5*time.Minute),
		FPS:                  res.int("fps", set["fps"], *fps, fc.FPS, 15),
		StartCelsius:         !res.boolean("units (f)", set["f"], *fahrenheit, unitsF, false),
		Color:                res.boolean("color", set["color"], *color, fc.Color, false),
		Stars:                res.str("stars", set["stars"], *stars, "", strOr(fc.Stars), "light"),
		LightKey:             res.str("light-key", set["light-key"], *lightKey, envOr("LIGHT_POLLUTION_MAP_API_KEY", ""), strOr(fc.LightKey), ""),
		MoonMode:             res.str("moon", set["moon"], *moonMode, "", strOr(fc.Moon), "auto"),
		MoonPhase:            res.flt("phase", set["phase"], *phase, fc.Phase, -1),
		Eclipse:              res.boolean("eclipse", set["eclipse"], *eclipse, fc.Eclipse, false),
		EclipseDuration:      res.dur("eclipse-duration", set["eclipse-duration"], *eclipseDur, strOr(fc.EclipseDuration), 2*time.Minute),
		SolarEclipse:         res.boolean("solar-eclipse", set["solar-eclipse"], *solarEclipse, fc.SolarEclipse, false),
		SolarEclipseDuration: res.dur("solar-eclipse-duration", set["solar-eclipse-duration"], *solarDur, strOr(fc.SolarEclipseDuration), 2*time.Minute),
		UseIP:                res.boolean("use-ip", set["use-ip"], *useIP, fc.UseIP, false),
		NoIPPrompt:           *noIPPrompt,
		SizePct:              res.flt("size", set["size"], *size, fc.Size, 15),
		Season:               res.str("season", set["season"], *season, "", strOr(fc.Season), "auto"),
		Theme:                res.str("theme", set["theme"], *theme, "", strOr(fc.Theme), "default"),
		Leaves:               res.boolean("leaves", set["leaves"], *leaves, fc.Leaves, true),
	}
	res.add("no-ip-prompt", strconv.FormatBool(*noIPPrompt), srcFlag)

	if *listConfig {
		fmt.Printf("# awapp %s resolved config (path: %s)\n", version, configPath)
		for _, r := range res.list {
			fmt.Printf("%-18s = %-18s (%s)\n", r.name, r.value, r.source)
		}
		return
	}

	if *saveConfig {
		cfg.OnQuit = func(f app.FinalSettings) {
			if err := saveToggles(configPath, f); err != nil {
				fmt.Fprintln(os.Stderr, "awapp: warning: could not save config: "+err.Error())
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()
	defer cancel()

	if err := app.Run(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "awapp:", err)
		os.Exit(1)
	}
}
