// Package app contains the main event loop: it owns the terminal,
// the animation buffer, the weather poller, and the small state
// machine that switches between "live" rendering and the offline
// manual-selection fallback.
package app

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"weatherterm/internal/anim"
	"weatherterm/internal/light"
	"weatherterm/internal/overlay"
	"weatherterm/internal/render"
	"weatherterm/internal/term"
	"weatherterm/internal/weather"
)

type Config struct {
	City         string
	APIKey       string
	Interval     time.Duration
	FPS          int
	StartCelsius bool
	// Color enables 256-color output. It defaults to false (plain
	// monochrome); toggle anytime with 'c'.
	Color bool
	// Stars selects the star-field mode: "light" (density follows the
	// local light pollution), "full" (ignore pollution), or "off" (hide
	// stars). Toggle anytime with 't'.
	Stars string
	// LightKey is an optional lightpollutionmap.info API key for exact
	// light-pollution readings; without it a population-based estimate is
	// used.
	LightKey string
	// UseIP uses the keyless IP-location weather source (ipinfo + Open-Meteo)
	// instead of an API key. When no API key is set, the app asks first.
	UseIP bool
	// MoonMode controls Moon visibility: "auto" (the phase decides, and a
	// new moon hides itself), "on", or "off". MoonPhase pins the phase to
	// a fixed value (0..1) for testing, -1 computes it from the date.
	MoonMode  string
	MoonPhase float64
	// Eclipse starts a simulated lunar eclipse at launch; it is otherwise
	// toggled with 'e'. EclipseDuration is the full start-to-end length.
	Eclipse         bool
	EclipseDuration time.Duration
	// SolarEclipse starts a simulated solar eclipse at launch (day scene,
	// toggled with 'x'). SolarEclipseDuration is its full length.
	SolarEclipse         bool
	SolarEclipseDuration time.Duration
	// SizePct is the Sun/Moon diameter as a % of the terminal width
	// (default 15). Toggle anytime with '+' / '-'.
	SizePct float64
	// Season overrides the ambient leaf effect: "auto" (default) computes
	// it from today's date + the location's hemisphere; or "spring",
	// "summer", "fall", "winter" to force it.
	Season string
	// Leaves enables the seasonal leaf/snow layer (default on; toggle
	// anytime with 'l').
	Leaves bool
}

type mode int

const (
	modeLoading mode = iota // waiting on the very first fetch attempt
	modeLive                // rendering the real, API-reported condition
	modeOffline             // comms unavailable, waiting on the user to pick
	modeManual              // rendering a user-picked condition, still retrying in background
)

const defaultStarFactor = 0.5 // used until light-pollution data arrives

type App struct {
	cfg Config
	t   *term.Term
	buf *render.Buffer

	poller      *weather.Poller
	lightClient *light.Client

	mode       mode
	report     weather.Report
	hasReport  bool
	lastErr    error
	animator   animatorIface
	moonOpts   anim.MoonOptions
	showPanel  bool
	fahrenheit bool
	color      bool
	frame      int

	// Light pollution / stars.
	stars    string // "light" | "full" | "off"
	lightRpt light.Report
	hasLight bool

	// Sun/Moon size, as a % of terminal width.
	sizePct float64

	// CloudsOn toggles whether clouds/cloud-decks are drawn ('o' key).
	cloudsOn bool
	// cloudsToggled is true once the user has pressed 'o'; until then each
	// scene keeps its own default (rain starts with clouds hidden).
	cloudsToggled bool

	// Season override from -season / config ("auto" computes from date +
	// hemisphere); empty/auto means compute.
	seasonOverride string

	// Leaves is the ambient season-driven drifting-leaf layer.
	leaves *anim.Leaves
	// leavesOn toggles the whole leaf layer ('l' key); default on.
	leavesOn bool
	// season is the currently rendered season (for the status panel).
	season anim.Season

	// Solar eclipse (day scene).
	solarActive bool
	solarStart  time.Time
}

type animatorIface interface {
	Resize(w, h int)
	SetTopMargin(rows int)
	Tick()
	Draw(buf *render.Buffer)
}

func Run(ctx context.Context, cfg Config) (err error) {
	// Convert any panic into a plain error return instead of crashing. The
	// deferred Restore()/ExitFullscreen() below are registered after this
	// defer, so they run first and the terminal is never left in raw
	// mode / alt-screen.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	t, err := term.Open()
	if err != nil {
		return fmt.Errorf("open terminal: %w", err)
	}
	defer t.Restore()

	term.EnterFullscreen()
	defer term.ExitFullscreen()

	size, err := t.Size()
	if err != nil {
		return fmt.Errorf("terminal size: %w", err)
	}

	fetcher, offlineAtStart := resolveFetcher(cfg)
	poller := weather.NewPoller(fetcher, cfg.Interval)

	stars := cfg.Stars
	switch stars {
	case "full", "off":
		// ok
	default:
		stars = "light"
	}

	a := &App{
		cfg:            cfg,
		t:              t,
		buf:            render.NewBuffer(size.Cols, size.Rows),
		poller:         poller,
		lightClient:    light.NewClient(cfg.LightKey),
		mode:           modeLoading,
		fahrenheit:     !cfg.StartCelsius,
		color:          cfg.Color,
		moonOpts:       moonOptionsFrom(cfg),
		stars:          stars,
		sizePct:        cfg.SizePct,
		cloudsOn:       true,
		seasonOverride: cfg.Season,
		leaves:         anim.NewLeaves(),
		leavesOn:       cfg.Leaves,
	}
	if a.sizePct <= 0 {
		a.sizePct = 15 // default: 15% of the terminal width
	}
	a.leaves.Resize(size.Cols, size.Rows)
	a.leaves.SetOn(a.leavesOn)
	if cfg.SolarEclipse {
		a.solarActive = true
		a.solarStart = time.Now()
	}
	a.buf.Color = a.color
	a.setAnimator(anim.NewClouds(false, false, false, 0, 0)) // neutral placeholder while loading

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		defer func() {
			// A panic inside the poller must never take the whole app down
			// while the terminal is in raw mode; surface it as a fetch error
			// and let the offline picker take over.
			if r := recover(); r != nil {
				reportError(poller.Errors, fmt.Errorf("weather poller: %v", r))
			}
		}()
		poller.Run(ctx)
	}()

	keys := make(chan byte, 16)
	go func() {
		for {
			b, err := t.ReadByte()
			if err != nil {
				close(keys)
				return
			}
			keys <- b
		}
	}()

	fps := cfg.FPS
	if fps <= 0 {
		fps = 15
	}
	ticker := time.NewTicker(time.Second / time.Duration(fps))
	defer ticker.Stop()

	// If we have no way to reach a live source (no API key and the user
	// declined IP-based weather), go straight to the offline picker.
	if offlineAtStart {
		a.enterOffline(fmt.Errorf("not configured: set an API key, use -use-ip, or write a config file"))
	}

	lightCh := make(chan light.Report, 1)
	lightDone := false

	for {
		select {
		case <-ctx.Done():
			return nil

		case b, ok := <-keys:
			if !ok {
				return nil
			}
			if quit := a.handleKey(b); quit {
				return nil
			}

		case <-t.Resized():
			sz, err := t.Size()
			if err == nil {
				a.buf.Resize(sz.Cols, sz.Rows)
				a.animator.Resize(sz.Cols, sz.Rows)
				a.leaves.Resize(sz.Cols, sz.Rows)
			}

		case r := <-poller.Reports:
			a.report = r
			a.hasReport = true
			a.lastErr = nil
			a.mode = modeLive
			a.setAnimator(animatorFor(r, a.moonOpts, a.cloudsOn))
			a.season = a.resolveSeason(r.Lat)
			a.leaves.SetSeason(a.season)
			a.leaves.SetWind(r.WindMS, r.WindDir)
			if !lightDone && r.City != "" {
				lightDone = true
				city, lat, lon := r.City, r.Lat, r.Lon
				go func() {
					defer func() { recover() }() // a light-pollution lookup must never crash the app
					rpt, err := a.lightClient.Estimate(ctx, city, lat, lon)
					if err == nil {
						lightCh <- rpt
					}
				}()
			}

		case err := <-poller.Errors:
			a.lastErr = err
			if a.mode == modeLoading {
				a.enterOffline(err)
			}
			// In modeLive/modeManual we just keep the last good frame
			// on screen and silently retry next tick.

		case lr := <-lightCh:
			a.lightRpt = lr
			a.hasLight = true
			a.applyStars()

		case <-ticker.C:
			a.frame++
			a.checkEclipseEnd()
			a.checkSolarEclipseEnd()
			a.animator.Tick()
			a.leaves.Tick()
			var info overlay.Info
			if a.showPanel {
				info = a.overlayInfo()
				a.setTopMargin(overlay.PanelHeight(info) + 1)
			} else {
				a.setTopMargin(0)
			}
			a.animator.Draw(a.buf)
			a.leaves.Draw(a.buf)
			if a.showPanel {
				overlay.Draw(a.buf, info)
			}
			if _, werr := fmt.Print(a.buf.Frame()); werr != nil {
				// stdout broke (e.g. piped away): stop cleanly — the deferred
				// Restore()/ExitFullscreen() put the terminal back.
				return werr
			}
		}
	}
}

// resolveFetcher picks the weather source. An API key wins; otherwise a
// configured city gives keyless Open-Meteo city weather; failing that it
// asks whether to use the keyless IP-location source. It returns the
// fetcher and whether the app should start in offline (picker) mode.
func resolveFetcher(cfg Config) (weather.Fetcher, bool) {
	if cfg.APIKey != "" {
		return weather.NewClient(cfg.APIKey, cfg.City), false
	}
	if cfg.City != "" {
		fmt.Fprintln(os.Stderr, "weatherterm: using keyless city weather for "+cfg.City+" (no API key)")
		return weather.NewCityClient(cfg.City), false
	}
	if cfg.UseIP || askUseIP() {
		fmt.Fprintln(os.Stderr, "weatherterm: using IP-location weather (no API key)")
		return weather.NewIPClient(), false
	}
	return weather.NewClient("", cfg.City), true
}

// askUseIP prompts the user to opt into keyless IP-based weather. When
// stdin isn't a terminal it can't ask, so it defaults to yes (the least
// effort, least invasive choice).
func askUseIP() bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return true
	}
	fmt.Fprint(os.Stderr, "\nNo API key found. Use weather from your IP location instead? [Y/n] ")
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "" || line == "y" || line == "yes"
}

// moonOptionsFrom converts the CLI config into anim.MoonOptions.
func moonOptionsFrom(cfg Config) anim.MoonOptions {
	o := anim.MoonOptions{
		PhaseOverride:   cfg.MoonPhase,
		Eclipse:         cfg.Eclipse,
		EclipseDuration: cfg.EclipseDuration,
	}
	if o.Eclipse {
		o.EclipseStart = time.Now()
	}
	switch cfg.MoonMode {
	case "on":
		o.Auto, o.Visible = false, true
	case "off":
		o.Auto, o.Visible = false, false
	default: // "auto" — let the phase decide
		o.Auto = true
	}
	return o
}

func (a *App) enterOffline(err error) {
	a.mode = modeOffline
	a.lastErr = err
	a.showPanel = true // surface the picker instructions immediately
}

func (a *App) setAnimator(an animatorIface) {
	a.animator = an
	an.Resize(a.buf.W, a.buf.H)
	a.applyStars()
	a.applySolar()
	a.applySize()
	a.applyClouds()
}

// handleKey processes one input byte and returns true if the app
// should quit.
func (a *App) handleKey(b byte) bool {
	switch b {
	case 'q', 'Q', 0x03: // Ctrl+C
		return true
	case 's', 'S':
		a.showPanel = !a.showPanel
	case 'u', 'U':
		a.fahrenheit = !a.fahrenheit
		a.showPanel = true // make the unit change visible
	case 'c', 'C':
		a.color = !a.color
		a.buf.Color = a.color
	case 't', 'T':
		a.cycleStars()
	case 'm', 'M':
		a.cycleMoon()
	case 'e', 'E':
		a.toggleEclipse()
	case 'x', 'X':
		a.toggleSolarEclipse()
	case '+', '=':
		a.sizePct += 2
		if a.sizePct > 60 {
			a.sizePct = 60
		}
		a.applySize()
	case '-', '_':
		a.sizePct -= 2
		if a.sizePct < 4 {
			a.sizePct = 4
		}
		a.applySize()
	case 'o', 'O':
		a.toggleClouds()
	case 'l', 'L':
		a.leavesOn = !a.leavesOn
		a.leaves.SetOn(a.leavesOn)
	case '1':
		a.pickManual(weather.Clear, "clear sky (manual)")
	case '2':
		a.pickManual(weather.Clouds, "clouds (manual)")
	case '3':
		a.pickManual(weather.Rain, "rain (manual)")
	case '4':
		a.pickManual(weather.Snow, "snow (manual)")
	case '5':
		a.pickManual(weather.Thunderstorm, "thunderstorm (manual)")
	}
	return false
}

// cycleStars walks light-pollution-sim -> hidden -> full -> sim, so the
// realistic star field can be shown, hidden, or replaced with an
// idealized full sky.
func (a *App) cycleStars() {
	switch a.stars {
	case "light":
		a.stars = "off"
	case "off":
		a.stars = "full"
	default:
		a.stars = "light"
	}
	a.applyStars()
}

// starFactorEffective resolves the current star density (0..1) from the
// star mode and the measured/estimated light pollution.
func (a *App) starFactorEffective() float64 {
	switch a.stars {
	case "full":
		return 1.0
	case "off":
		return 0.0
	default: // "light"
		if a.hasLight && a.lightRpt.Bortle >= 1 && a.lightRpt.Bortle <= 9 {
			return light.StarFactor[a.lightRpt.Bortle]
		}
		return defaultStarFactor
	}
}

func (a *App) applyStars() {
	if s, ok := a.animator.(*anim.Sun); ok {
		s.SetStarFactor(a.starFactorEffective())
	}
}

// toggleEclipse starts or stops a simulated lunar eclipse. Starting one
// sets the timeline's clock; the Moon then disappears and reappears over
// EclipseDuration without any polling.
func (a *App) toggleEclipse() {
	if a.moonOpts.Eclipse {
		a.moonOpts.Eclipse = false
	} else {
		a.moonOpts.Eclipse = true
		a.moonOpts.EclipseStart = time.Now()
	}
	a.applyMoon()
}

// checkEclipseEnd turns the lunar eclipse off automatically once its
// simulated duration has elapsed.
func (a *App) checkEclipseEnd() {
	if !a.moonOpts.Eclipse {
		return
	}
	if _, active := anim.EclipseProgress(a.moonOpts.EclipseStart, a.moonOpts.EclipseDuration, time.Now()); !active {
		a.moonOpts.Eclipse = false
		a.applyMoon()
	}
}

// toggleSolarEclipse starts or stops a simulated solar eclipse (day
// scene), with the same time-based simulation as the lunar one.
func (a *App) toggleSolarEclipse() {
	if a.solarActive {
		a.solarActive = false
	} else {
		a.solarActive = true
		a.solarStart = time.Now()
	}
	a.applySolar()
}

// checkSolarEclipseEnd turns the solar eclipse off once its duration
// has elapsed.
func (a *App) checkSolarEclipseEnd() {
	if !a.solarActive {
		return
	}
	if _, active := anim.EclipseProgress(a.solarStart, a.cfg.SolarEclipseDuration, time.Now()); !active {
		a.solarActive = false
		a.applySolar()
	}
}

// applySolar pushes the current solar-eclipse state into the animator.
func (a *App) applySolar() {
	if s, ok := a.animator.(*anim.Sun); ok {
		s.SetSolar(a.solarActive, a.solarStart, a.cfg.SolarEclipseDuration)
	}
}

// cycleMoon walks auto -> hidden -> shown -> auto, so the Moon can be
// forced off or on, or left to the phase (a new moon auto-hides).
func (a *App) cycleMoon() {
	keep := anim.MoonOptions{
		PhaseOverride:   a.moonOpts.PhaseOverride,
		Eclipse:         a.moonOpts.Eclipse,
		EclipseStart:    a.moonOpts.EclipseStart,
		EclipseDuration: a.moonOpts.EclipseDuration,
	}
	switch {
	case a.moonOpts.Auto:
		a.moonOpts = keep // forced hidden
	case a.moonOpts.Visible:
		a.moonOpts = keep
		a.moonOpts.Auto = true // back to phase-driven
	default:
		a.moonOpts = keep
		a.moonOpts.Visible = true // forced shown
	}
	a.applyMoon()
}

// applySize pushes the configured Sun/Moon size into the live animator so
// the '+' / '-' keys take effect immediately.
func (a *App) applySize() {
	if s, ok := a.animator.(*anim.Sun); ok {
		s.SetSize(a.sizePct)
	}
}

// toggleClouds flips whether clouds are drawn at all (the 'o' key hides
// the cloud deck / puffs, leaving just sky and falling rain). It flips
// whatever is actually on screen right now — rain starts with its deck
// hidden even though the app default is "shown", so the first 'o' in a
// rain scene brings the clouds back.
func (a *App) toggleClouds() {
	cur := a.cloudsOn
	switch an := a.animator.(type) {
	case *anim.Precip:
		cur = an.CloudsOn
	case *anim.Clouds:
		cur = an.CloudsOn
	}
	a.cloudsOn = !cur
	a.cloudsToggled = true
	a.applyClouds()
}

// applyClouds pushes the cloud-visibility choice into the live animator.
// Until the user presses 'o' (cloudsToggled), each scene keeps its own
// default — rain starts with its cloud deck hidden.
func (a *App) applyClouds() {
	if !a.cloudsToggled {
		return
	}
	if pr, ok := a.animator.(*anim.Precip); ok {
		pr.SetClouds(a.cloudsOn)
	}
	if c, ok := a.animator.(*anim.Clouds); ok {
		c.SetClouds(a.cloudsOn)
	}
}

// resolveSeason returns the season to render: an explicit -season override
// wins; otherwise it is computed from today's date and the report's
// latitude (which flips the months in the southern hemisphere).
func (a *App) resolveSeason(lat float64) anim.Season {
	switch a.seasonOverride {
	case "spring":
		return anim.SeasonSpring
	case "summer":
		return anim.SeasonSummer
	case "fall", "autumn":
		return anim.SeasonFall
	case "winter":
		return anim.SeasonWinter
	}
	return anim.SeasonFor(time.Now().Month(), lat)
}

// setTopMargin reserves screen rows at the top (for the status panel) so
// the Sun/Moon arc, clouds, and rain cloud deck are drawn below it.
func (a *App) setTopMargin(rows int) {
	a.animator.SetTopMargin(rows)
}

// applyMoon pushes the current Moon options into the live animator so
// the change takes effect immediately, without waiting for the next fetch.
func (a *App) applyMoon() {
	if s, ok := a.animator.(*anim.Sun); ok {
		s.SetMoon(a.moonOpts)
	}
}

func (a *App) pickManual(c weather.Condition, desc string) {
	if a.mode != modeOffline && a.mode != modeManual && a.mode != modeLoading {
		return // ignore digit keys while a live report is being shown
	}
	a.mode = modeManual
	a.report = weather.Report{City: a.cfg.City, Condition: c, Desc: desc, FetchedAt: time.Now()}
	if a.report.City == "" {
		a.report.City = "unknown"
	}
	a.hasReport = true
	a.setAnimator(animatorFor(a.report, a.moonOpts, a.cloudsOn))
	a.season = a.resolveSeason(a.report.Lat)
	a.leaves.SetSeason(a.season)
	a.leaves.SetWind(a.report.WindMS, a.report.WindDir)
}

func (a *App) overlayInfo() overlay.Info {
	info := overlay.Info{
		Fahrenheit: a.fahrenheit,
		Live:       a.mode == modeLive,
		HasData:    a.hasReport,
		HasTemp:    a.mode == modeLive && a.hasReport,
	}
	if a.hasReport {
		info.City = a.report.City
		info.Desc = a.report.Desc
		if a.mode == modeLive {
			info.TempC = a.report.TempC()
			info.TempF = a.report.TempF()
			info.WindMS = a.report.WindMS
		}
	}
	info.Err = a.errorLine()
	if a.hasReport {
		info.Season = a.season.String()
		info.LeavesOn = a.leavesOn
	}
	info.Moon = a.moonLine()
	info.Sun = a.sunLine()
	info.Stars = a.starsLine()
	return info
}

// errorLine describes the most recent weather fetch failure (shown in
// the status panel, so the offline picker explains *why* it went
// offline instead of silently showing "No data yet").
func (a *App) errorLine() string {
	if a.lastErr == nil {
		return ""
	}
	return a.lastErr.Error()
}

// reportError drains-then-sends an error onto ch so the consumer always
// sees the latest failure (used by goroutine recover handlers).
func reportError(ch chan error, err error) {
	select {
	case ch <- err:
	default:
		select {
		case <-ch:
		default:
		}
		ch <- err
	}
}

// moonLine describes the Moon in the status panel whenever the
// clear-sky night scene is showing.
func (a *App) moonLine() string {
	s, ok := a.animator.(*anim.Sun)
	if !ok || !s.Night {
		return ""
	}
	if !s.MoonVisible() {
		if a.moonOpts.Auto {
			return "Moon: new moon - nothing to see"
		}
		return "Moon: hidden (press m to show)"
	}
	if prog, active := s.Eclipse(); active {
		start := a.moonOpts.EclipseStart
		dur := a.moonOpts.EclipseDuration
		switch {
		case prog < 0.45:
			return "Moon: lunar eclipse - totality in " + shortDur(time.Until(start.Add(dur/2)))
		case prog <= 0.55:
			return "Moon: lunar eclipse - totality (moon hidden)"
		default:
			return "Moon: lunar eclipse - ends in " + shortDur(time.Until(start.Add(dur)))
		}
	}
	p := s.Phase()
	return fmt.Sprintf("Moon: %s - %d%% lit", anim.MoonPhaseName(p), int(anim.MoonIllum(p)*100+0.5))
}

// sunLine describes a running solar eclipse in the day scene.
func (a *App) sunLine() string {
	s, ok := a.animator.(*anim.Sun)
	if !ok || s.Night {
		return ""
	}
	if prog, active := s.Solar(); active {
		start := a.solarStart
		dur := a.cfg.SolarEclipseDuration
		switch {
		case prog < 0.45:
			return "Sun: solar eclipse - totality in " + shortDur(time.Until(start.Add(dur/2)))
		case prog <= 0.55:
			return "Sun: solar eclipse - totality (corona)"
		default:
			return "Sun: solar eclipse - ends in " + shortDur(time.Until(start.Add(dur)))
		}
	}
	return ""
}

// starsLine describes the star field (light pollution) in the panel.
func (a *App) starsLine() string {
	s, ok := a.animator.(*anim.Sun)
	if !ok || !s.Night {
		return ""
	}
	switch a.stars {
	case "off":
		return "Stars: hidden (t)"
	case "full":
		return "Stars: full sky (t)"
	default:
		if a.hasLight {
			return "Stars: " + a.lightRpt.Label + " (t)"
		}
		return "Stars: light-pollution sim (t)"
	}
}

// shortDur formats a duration compactly, e.g. 1m05s or 42s.
func shortDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// animatorFor picks the animator for a condition, deciding day/night
// from the API-reported sunrise/sunset when available and falling back
// to the local clock otherwise.
func animatorFor(r weather.Report, moonOpts anim.MoonOptions, cloudsOn bool) animatorIface {
	night := isNight(r)
	heavy := strings.Contains(strings.ToLower(r.Desc), "heavy")

	switch r.Condition {
	case weather.Clear:
		sun := anim.NewSun(night, moonOpts)
		sun.SetTimes(anim.SkyTimes{Sunrise: r.Sunrise, Sunset: r.Sunset, Timezone: r.Timezone})
		return sun
	case weather.Clouds:
		c := anim.NewClouds(strings.Contains(strings.ToLower(r.Desc), "overcast"), false, false, r.WindMS, r.WindDir)
		c.SetClouds(cloudsOn)
		return c
	case weather.Mist:
		c := anim.NewClouds(true, true, false, r.WindMS, r.WindDir)
		c.SetClouds(cloudsOn)
		return c
	case weather.Rain:
		pr := anim.NewPrecip(anim.ModeRain, heavy, false, r.WindMS, r.WindDir)
		pr.SetClouds(false) // rain starts with its cloud deck hidden ('o' brings it back)
		return pr
	case weather.Thunderstorm:
		return anim.NewPrecip(anim.ModeRain, true, true, r.WindMS, r.WindDir)
	case weather.Snow:
		return anim.NewPrecip(anim.ModeSnow, heavy, false, r.WindMS, r.WindDir)
	default:
		c := anim.NewClouds(false, false, false, r.WindMS, r.WindDir)
		c.SetClouds(cloudsOn)
		return c
	}
}

// isNight decides whether the sky is dark right now. It prefers the real
// sunrise/sunset from the weather source (converted to local civil time);
// for offline manual picks those fields are zero, so it falls back to the
// local wall clock.
func isNight(r weather.Report) bool {
	if r.Sunrise > 0 && r.Sunset > 0 && !r.FetchedAt.IsZero() {
		// FetchedAt is an absolute instant (time.Now(), possibly in the
		// machine's local zone), so convert to UTC before adding the
		// location's offset — otherwise the offset is applied to local time
		// and the answer is wrong by the machine's own UTC shift.
		nowLocal := secsOfDay(r.FetchedAt.UTC().Add(time.Duration(r.Timezone) * time.Second))
		sr := secsOfDay(time.Unix(r.Sunrise+r.Timezone, 0).UTC())
		ss := secsOfDay(time.Unix(r.Sunset+r.Timezone, 0).UTC())
		if sr < ss {
			return nowLocal < sr || nowLocal >= ss
		}
		return sr <= nowLocal && nowLocal < ss
	}
	h := time.Now().Hour()
	return h < 6 || h >= 18
}

func secsOfDay(t time.Time) int {
	return t.Hour()*3600 + t.Minute()*60 + t.Second()
}
