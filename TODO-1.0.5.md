# awapp 1.0.5 roadmap

Status legend: `[x]` done · `[~]` partial/deferred · `[ ]` todo

## Quick wins

- [x] Leaves toggle shows the status panel on `l` (like `u`)
- [x] AUR `PKGBUILD` maintainer email fixed (`rodlessa@users.noreply.github.com`)
- [x] Config validation warnings (unknown JSON keys, bad `size`/`stars`/`moon`/
      `provider`/`theme`) — `main_test.go`
- [x] `-no-ip-prompt` flag to skip the IP-location prompt (scripted/systemd)
- [x] `internal/app` state-machine tests (`nextMode`: loading → live/offline/manual)

## Weather / data features

- [x] Hourly/daily forecast strip in the status panel (`weather.HourPoint`/`DayPoint`,
      Open-Meteo + OWM 3-hourly; `forecast_test.go`)
- [~] Multi-day view — daily strip done; full forecast-driven animation preview
      deferred
- [x] Pluggable providers as `Fetcher`s + `-provider` flag: `openweather`,
      `open-meteo`, `weatherapi`, `tomorrowio` (`providers.go` + tests)
- [x] Air quality index overlay (keyless Open-Meteo, `internal/air`)
- [x] Severe weather alerts (panel `⚠` lines; audible bell not added)
- [x] UV index indicator (`uv_index` from Open-Meteo)

## Visual / animation features

- [x] Rainbow after rain (probability-based, sun + rain; `Precip.drawRainbow`)
- [x] Lightning flash tied to thunderstorm intensity (heavy storms flash ~3x,
      with a visible bolt) — `Precip.drawBolt`
- [x] Fog/mist density improvements (drifting ground-fog bank, `drawFogBank`)
- [x] Aurora at high latitudes via NOAA Kp (`internal/aurora` + night band)
- [x] Meteor shower easter egg on known shower dates (`meteorActive`)
- [x] Themeable color palettes (`-theme`: default/sunset/ocean/forest)
- [x] City skyline silhouette scaled by population/Bortle (`StarField.SetSkyline`)

## Config / UX

- [x] `-list-config` prints resolved settings + source (flag/env/file/default)
- [ ] Interactive first-run setup wizard (deferred — big interactive feature)
- [x] `-save-config` writes last-used color/units/stars toggles back to config
- [x] Man page (`packaging/man/awapp.1`) + bash/zsh/fish completions

## Packaging / distribution

- [x] Homebrew formula (`packaging/homebrew/awapp.rb`)
- [x] macOS support — `term_linux.go`/`term_darwin.go` TCGETS/TCSETS aliases;
      verified `GOOS=darwin` (amd64+arm64) builds; CI matrix now builds macOS
- [x] Nix flake (`flake.nix`), `.deb`/`.rpm` (`packaging/deb`, `packaging/rpm`,
      `package.sh`), Docker image (`Dockerfile`)

## Testing / CI

- [x] Golden-file snapshot tests for rendered frames (`internal/anim/golden_test.go`,
      pin the clock via `SetClock`)
- [x] Fuzz targets: config JSON, WeatherAPI/Tomorrow.io parsers, Open-Meteo
      forecast parsers
- [x] `golangci-lint` config (`.golangci.yml`) + CI step

---

## Not done (deferred)
- First-run setup wizard
- Audible bell on severe alerts
- Full forecast-driven animation preview
