#compdef awapp

_awapp() {
    local -a opts
    opts=(
        '--apikey[OpenWeatherMap API key]'
        '--city[City to fetch]'
        '--use-ip[Use IP-location weather]'
        '--no-ip-prompt[Skip the IP-location prompt]'
        '--interval[Poll interval]'
        '--fps[Animation frame rate]'
        '-f[Start with Fahrenheit]'
        '--color[Enable 256-color output]'
        '--stars[Star field mode: light, full, off]'
        '--light-key[Light-pollution API key]'
        '--moon[Moon visibility: auto, on, off]'
        '--phase[Override moon phase]'
        '--eclipse[Simulated lunar eclipse]'
        '--eclipse-duration[Lunar eclipse length]'
        '--solar-eclipse[Simulated solar eclipse]'
        '--solar-eclipse-duration[Solar eclipse length]'
        '--size[Sun/Moon diameter %]'
        '--season[Leaf season]'
        '--leaves[Enable the leaf/snow layer]'
        '--save-config[Save toggles to config on quit]'
        '--list-config[Print resolved config and exit]'
        '--config[Path to config.json]'
        '--version[Print version and exit]'
    )
    _describe 'awapp options' opts
}

_awapp "$@"
