# awapp

A full-screen terminal weather visualizer for Linux. It renders the
current condition as an animation: rain, drifting clouds, or a
creative clear-sky scene. On a clear day the **Sun rises from the
bottom of the terminal, arcs across the screen, and sets** — its
position follows the real sunrise/sunset times for your location; by
night a **starfield** whose density reflects the **local light
pollution** sits under a **braille-textured Moon** that likewise
rises and arcs across the night sky. Both bodies are drawn in a
**braille dot-pattern style** — the Sun is a dense `⣿`-centred,
sparse-edged braille disc with rays, the Moon a hand-drawn braille
sprite lit by its real phase. The Sun and Moon default to **15%
of the terminal width** and can be resized on the fly with **`+` / `-`**.
Clouds drift with the **reported wind speed** and come in different
ASCII kinds — puffy cumulus, flat overcast stratus, tall dark storm
clouds — so heavy weather actually looks heavy.

**No API key is required**: run it and it asks whether to use your **IP
location** (ipinfo + Open-Meteo, zero registration); an OpenWeatherMap
key is an optional, more accurate alternative. It runs in **plain
monochrome by default** (no color codes) — press `c` or pass `-color`
for 256-color output. If no weather source can be reached, it drops
into a manual picker and keeps retrying in the background.

Zero third-party dependencies. It's a single static-ish Go binary
that talks to the terminal directly via raw-mode syscalls and plain
ANSI/VT100 escapes, so it isn't tied to any particular terminal
emulator — it'll run in xterm, foot, kitty, alacritty, the Linux
console, tmux, whatever, as long as it understands ANSI and reports
its size via `TIOCGWINSZ`.

## Build

Requires Go 1.21+ and nothing else (no `go mod download` needed —
there are no external dependencies).

```sh
go build -o awapp .
```

## Run

Easiest — **no API key, no registration** (uses your IP location):

```sh
./awapp
```

It asks once: `No API key found. Use weather from your IP location
instead? [Y/n]` — press Enter. Or skip the question with `-use-ip`.

Want a specific place instead of your IP? Just name it — **still no key
needed**:

```sh
./awapp -city "Fortaleza,BR"
# or put only a city in the config file:
echo '{"city": "Berlin"}' > ~/.config/awapp/config.json
./awapp
```

That resolves the city with Open-Meteo's free geocoder and fetches its
weather; no API key, no prompt.

With an OpenWeatherMap key (optional, more accurate):

```sh
./awapp -apikey your_key_here -city "Fortaleza,BR"
# or, so you never have to export the key again:
mkdir -p ~/.config/awapp
echo '{"api_key": "your_key_here", "city": "Fortaleza,BR"}' > ~/.config/awapp/config.json
./awapp
```

Get a free OpenWeatherMap key at <https://openweathermap.org/api> (the
IP- and city-based sources need nothing at all).

If no weather source is reachable, it goes straight into offline mode
and waits for you to press a number key to pick a condition manually.

## Configuration

Settings are resolved in this order: **command-line flags > environment
variables > config file > defaults**.

The config file lives at `~/.config/awapp/config.json` (or
`$XDG_CONFIG_HOME/awapp/config.json`); point elsewhere with
`-config`. See `config.example.json`:

```json
{
  "api_key": "your_openweathermap_key",
  "city": "Fortaleza,BR",
  "use_ip": false,
  "units": "c",
  "color": false,
  "stars": "light",
  "interval": "5m",
  "moon": "auto",
  "light_key": "",
  "eclipse": false,
  "solar_eclipse": false,
  "size": 15
}
```

Recognized keys: `api_key`, `city`, `use_ip`, `units` (`c`/`f`),
`color`, `stars`, `interval` (duration), `fps`, `moon`, `phase`,
`light_key`, `eclipse`, `eclipse_duration`, `solar_eclipse`,
`solar_eclipse_duration`, `size`, `season`, `leaves`.

### Flags

| Flag                      | Default | Description                                             |
|---------------------------|---------|----------------------------------------------------------|
| `-config`                 | (auto)  | Path to a config.json file (see Configuration)          |
| `-apikey`                 | (env)   | OpenWeatherMap API key                                   |
| `-city`                   | (env)   | City query (works keyless via Open-Meteo; optional if geolocating) |
| `-use-ip`                 | off     | Use IP-location weather (no API key) without asking     |
| `-interval`               | `5m`    | How often to poll the weather API                        |
| `-fps`                    | `15`    | Animation frame rate                                     |
| `-f`                      | off     | Start in Fahrenheit (toggle anytime with `u`)            |
| `-color`                  | off     | Enable 256-color output (default: monochrome, toggle `c`) |
| `-stars`                  | `light` | Star field: `light` (per pollution), `full`, `off` (toggle `t`) |
| `-light-key`              | (env)   | Optional lightpollutionmap.info key for exact radiance  |
| `-moon`                   | `auto`  | Moon visibility: `auto` (phase decides), `on`, `off`    |
| `-eclipse`                | off     | Start a simulated **lunar** eclipse (toggle `e`)        |
| `-eclipse-duration`       | `2m`    | Length of the simulated lunar eclipse, start to end     |
| `-solar-eclipse`          | off     | Start a simulated **solar** eclipse, day scene (toggle `x`) |
| `-solar-eclipse-duration` | `2m`    | Length of the simulated solar eclipse, start to end     |
| `-phase`                  | `-1`    | Pin the moon phase to `0..1` for testing (`-1` = compute) |
| `-size`                   | `15`    | Sun/Moon diameter as % of terminal width (`4`..`60`)      |
| `-season`                 | `auto`  | Leaf season: `auto` (date + hemisphere), `spring`, `summer`, `fall`, `winter` |
| `-leaves`                 | on      | Enable the seasonal leaf/snow layer (toggle anytime with `l`)    |

Env var fallbacks: `OPENWEATHERMAP_API_KEY` / `OPENWEATHER_API_KEY`,
`OPENWEATHERMAP_CITY` / `OPENWEATHER_CITY`,
`LIGHT_POLLUTION_MAP_API_KEY` (for `-light-key`).

 

## Keybindings

| Key     | Action                                                          |
|---------|-----------------------------------------------------------------|
| `s`     | Toggle the status panel (city, temp, condition, Moon/Sun state) |
| `u`     | Toggle C / F (reveals the panel so the change is visible)       |
| `c`     | Toggle color on/off (default: monochrome)                       |
| `t`     | Cycle star field: light-pollution sim -> off -> full -> sim     |
| `m`     | Cycle Moon: auto -> hidden -> shown -> auto                     |
| `e`     | Start/stop a simulated **lunar** eclipse (Moon disappears)      |
| `x`     | Start/stop a simulated **solar** eclipse (Sun is blocked, day)  |
| `o`     | Toggle clouds on/off (hides cloud decks & puffs, rain only)  |
| `l`     | Toggle leaves on/off (default: on)                        |
| `+` / `-` | Resize the Sun/Moon disc (default 15% of terminal width)   |
| `1`-`5` | Manually render Clear / Clouds / Rain / Snow / Thunderstorm (only while offline) |
| `q`     | Quit                                                             |

The status panel shows whether you're looking at **live** data
(`[live]`) or a **manual** offline pick (`[offline]`), an explicit
`units: C/F` line, the **wind** (e.g. `wind 4.5 m/s`), and the current
Moon/Sun/star state on a clear sky, e.g. `Moon: waxing gibbous - 72%
lit` or `Stars: Bortle 7 (city, pop 1702139)`.

## Clouds & wind

Cloud scenes are driven by the weather report's **wind speed**: the
stronger the wind, the faster the clouds drift (and the more rain or
snow leans sideways). Different conditions pick different ASCII cloud
art:

| Weather      | Cloud art                              |
|--------------|----------------------------------------|
| Partly cloudy | puffy fair-weather cumulus             |
| Overcast     | flat stratus layer                     |
| Mist / haze  | thin wispy cirrus                      |
| Rain         | low nimbostratus deck with rain streaks |
| Thunderstorm | tall dark cumulonimbus with rain curtains |

Rain starts with its cloud deck hidden (just the falling rain); press `o`
to show or hide clouds (drops the deck and puffs entirely, leaving the
plain sky and rain/snow).

## Leaves & seasons

The season is derived from today's date and the location's **latitude**
(the southern hemisphere flips the months — Fortaleza is summer in
January, winter in July). Depending on the season, braille **leaves**
drift across the scene, pushed sideways by the **wind direction**:

- **Spring / Summer** — fresh green leaves.
- **Fall** — big dry curling leaves (yellows, oranges, reds).
- **Winter** — bare trees (no leaves). Snow only falls when the weather is
  actually snowy; near-freezing rain shows as **sleet** (rain + snow mixed).

Force a season for testing with `-season fall` (or the `season` config
key). The leaves lean left/right with the wind, like the rain. Only about
**5%** of leaves are full-size; the rest are downscaled and dimmed so they
read as drifting in the background, giving the scene depth. Leaves default
to **on** — press `l` to toggle them off/on.

## Light pollution & stars

On a clear night the star field tries to reflect how many stars you'd
actually be able to see at your location:

- OpenWeatherMap's report includes your coordinates, so the app looks
  up the local light pollution once per run and converts it to a
  **Bortle class** (1 = pristine dark sky .. 9 = inner city). Star
  density then scales with that class.
- **Exact**: if you set a free `lightpollutionmap.info` API key
  (`-light-key` or `LIGHT_POLLUTION_MAP_API_KEY`), it reads the real
  VIIRS night-lights radiance at your coordinates.
- **Fallback (no key needed)**: the city is geocoded with Open-Meteo's
  free API and its population is mapped to a Bortle class — population
  correlates strongly with artificial light at night.
- Press `t` to cycle the star field: `light` (the real simulation),
  `off` (hide them), or `full` (an idealized all-star sky).

## The Moon, the Sun & eclipses

On a clear day/night the Sun and Moon are drawn in a **braille
dot-pattern style** and **move across the terminal according to the
real time of day**:

- The Sun rises from the bottom-left at sunrise, peaks around noon
  just under the status panel, and sets back to the bottom-right at
  sunset — its position follows the **real sunrise/sunset times** for
  your location. Its disc is a braille radial gradient: **bigger, denser
dots (`⣿`) at the centre, smaller, sparser dots (`⠁`) at the borders**,
with eight `*` rays.
- The Moon does the same across the night sky, arcing from sunset to
  sunrise. It's drawn from a **braille dot-pattern sprite**: the Moon's
  disc is a textured braille moon, with the blank cells (its craters
  and spacing) staying transparent so the night sky shows through.
- **`+` / `-`** resize the disc on the fly (default **15% of the
  terminal width**; also `-size` / `size` in the config).
- Hiding the status panel with `s` lets the Sun/Moon arc higher up the
  screen; showing it keeps the arc just below the panel so it's never
  covered.

The Moon's phase is computed locally from the date (mean-lunation
approximation, no network or ephemeris needed), so the shape always
matches reality: crescent, quarter, gibbous, or full — lit on the
right while waxing, on the left while waning.
- A **new moon auto-hides** (there's nothing to see). Press `m` to
  force it `hidden`, force it `shown`, or return to `auto`.
- **Lunar eclipse** (`e`): a pure time-based simulation — Earth's
  shadow sweeps across the Moon and the braille Moon progressively
  disappears, fully hidden at totality, then returns. Because the Moon
  only *reflects* light, its loss darkens nothing else.
- **Solar eclipse** (`x`, day scene): the Moon's silhouette bites into
  the Sun (the *light source*), so the rays vanish, the whole sky
  darkens, and at totality a **corona** ring shines around the hidden
  Sun — then the Sun re-emerges. The two eclipses look deliberately
  different because the Sun and Moon behave differently.
- Both last `-eclipse-duration` / `-solar-eclipse-duration` (default 2
  minutes; real events run hours, e.g. `-eclipse-duration 3h30m`) and
  switch themselves off when done.
- Sunrise/sunset come straight from OpenWeatherMap, so the Moon only
  appears when the sky is actually dark at your location.

## How it behaves on failure

- No API key or city set at launch → goes straight to the offline
  picker, no waiting on a timeout.
- API reachable but errors out later (network drop, rate limit, DNS
  failure, etc.) → if this happens before the *first* successful
  fetch, drops to the offline picker. If it happens after you're
  already seeing live data, the last good frame just keeps animating
  and it quietly retries every interval — no interruption.
- Once a manual pick is active, the poller keeps retrying in the
  background; a successful fetch automatically switches back to live
  rendering.

## Resizing

The renderer rebuilds its animation state on every `SIGWINCH`
(terminal resize), so it adapts cleanly from a tiny tmux pane up to a
full ultrawide terminal — animator density (raindrop count, cloud
count, star count) scales with the new dimensions rather than
stretching fixed art.

## Layout

```
main.go                     CLI flags/env/config-file, wiring
internal/term/               raw terminal mode, size, resize signal, input (stdlib only)
internal/weather/             OWM client + keyless IP client (ipinfo + Open-Meteo) + poller
internal/light/               light-pollution lookup (radiance key or population)
internal/render/              character-grid buffer -> ANSI frame
internal/anim/                rain/snow, clouds (wind + art kinds), sun/moon, stars
internal/overlay/              the 's' status panel
internal/app/                  event loop / state machine
```

## Tests

```sh
go test ./...
```

Covers the weather classifiers (OWM + WMO), temperature conversion, the
keyless IP client, moon phase computation, eclipse timing, star scaling
with light pollution, cloud art selection and wind scaling, and
rendering every scene without clipping.

## Notes

- Weather can come from an OpenWeatherMap API key (most accurate) or
  from your IP location via ipinfo + Open-Meteo (no key needed). It is
  city-level, so it isn't invasive.
- Day/night for the clear-sky animation uses the real sunrise/sunset
  from the API; offline manual picks fall back to the local clock.
- Light pollution is estimated from city population (free, no key) or,
  with an optional `lightpollutionmap.info` API key, read exactly.
- Colors default to **off** (plain monochrome ASCII). When enabled
  (`c` key or `-color`) the scenes use 256-color ANSI codes (not
  truecolor) for the widest terminal-emulator compatibility.
